package ftp

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// permMap maps each position in an ls-style permission string to its bit.
var permMap = []struct {
	char byte
	bit  os.FileMode
}{
	{'r', 0400}, {'w', 0200}, {'x', 0100}, // owner
	{'r', 0040}, {'w', 0020}, {'x', 0010}, // group
	{'r', 0004}, {'w', 0002}, {'x', 0001}, // others
}

// unixModeToFileMode converts a 12-bit unix mode into os.FileMode.
func unixModeToFileMode(mode uint32) os.FileMode {
	fm := os.FileMode(mode & 0777)
	if mode&04000 != 0 {
		fm |= os.ModeSetuid
	}
	if mode&02000 != 0 {
		fm |= os.ModeSetgid
	}
	if mode&01000 != 0 {
		fm |= os.ModeSticky
	}
	return fm
}

// fileModeToUnixMode is the inverse of unixModeToFileMode.
func fileModeToUnixMode(fm os.FileMode) uint32 {
	mode := uint32(fm.Perm())
	if fm&os.ModeSetuid != 0 {
		mode |= 04000
	}
	if fm&os.ModeSetgid != 0 {
		mode |= 02000
	}
	if fm&os.ModeSticky != 0 {
		mode |= 01000
	}
	return mode
}

var (
	errUnsupportedListLine  = errors.New("unsupported LIST line")
	errUnsupportedListDate  = errors.New("unsupported LIST date")
	errUnknownListEntryType = errors.New("unknown entry type")
)

type parseFunc func(string, time.Time, *time.Location) (*Entry, error)

var listLineParsers = []parseFunc{
	parseRFC3659ListLine,
	parseLsListLine,
	parseDirListLine,
	parseHostedFTPLine,
}

var dirTimeFormats = []string{
	"01-02-06  03:04PM",
	"2006-01-02  15:04",
	"01-02-2006  03:04PM",
	"01-02-2006  15:04",
}

// parseRFC3659ListLine parses the style of directory line defined in RFC 3659.
func parseRFC3659ListLine(line string, _ time.Time, loc *time.Location) (*Entry, error) {
	return parseNextRFC3659ListLine(line, loc, &Entry{})
}

func parseNextRFC3659ListLine(line string, loc *time.Location, e *Entry) (*Entry, error) {
	iSemicolon := strings.Index(line, ";")
	iWhitespace := strings.Index(line, " ")

	if iSemicolon < 0 || iSemicolon > iWhitespace {
		return nil, errUnsupportedListLine
	}

	name := line[iWhitespace+1:]
	if e.name == "" {
		e.name = name
	} else if e.name != name {
		// All lines must have the same name
		return nil, errUnsupportedListLine
	}

	for _, field := range strings.Split(line[:iWhitespace-1], ";") {
		i := strings.Index(field, "=")
		if i < 1 {
			return nil, errUnsupportedListLine
		}

		key := strings.ToLower(field[:i])
		value := field[i+1:]

		switch key {
		case "unix.mode":
			mode, err := strconv.ParseUint(value, 8, 32)
			if err != nil {
				return nil, err
			}
			// Don't clobber type bits set by an earlier "type" fact.
			const lowBits = os.ModePerm | os.ModeSetuid | os.ModeSetgid | os.ModeSticky
			e.mode = (e.mode &^ lowBits) | unixModeToFileMode(uint32(mode))
		case "modify":
			var err error
			e.time, err = time.ParseInLocation("20060102150405", value, loc)
			if err != nil {
				return nil, err
			}
		case "type":
			switch value {
			case "dir", "cdir", "pdir":
				e.mode |= os.ModeDir
			}
		case "size":
			if err := e.setSize(value); err != nil {
				return nil, err
			}
		}
	}
	return e, nil
}

// parseLsListLine parses a directory line in a format based on the output of
// the UNIX ls command.
func parseLsListLine(line string, now time.Time, loc *time.Location) (*Entry, error) {

	// Has the first field a length of exactly 10 bytes
	// - or 10 bytes with an additional '+' character for indicating ACLs?
	// If not, return.
	if i := strings.IndexByte(line, ' '); i != 10 && (i != 11 || line[10] != '+') {
		return nil, errUnsupportedListLine
	}

	scanner := newScanner(line)
	fields := scanner.NextFields(6)

	if len(fields) < 6 {
		return nil, errUnsupportedListLine
	}

	// Decode the permission column, e.g. "-rwxr-xr-x" or "drwxr-xr-x+"
	// (with an ACL marker). Uppercase 'S'/'T' mean the special bit is set
	// without the execute bit; lowercase implies execute too.
	var fileMode os.FileMode
	if len(fields[0]) >= 10 {
		for i, pm := range permMap {
			c := fields[0][i+1]
			switch {
			case c == pm.char:
				fileMode |= pm.bit
			case pm.char == 'x' && (c == 's' || c == 'S'):
				if i < 3 {
					fileMode |= os.ModeSetuid
				} else {
					fileMode |= os.ModeSetgid
				}
				if c == 's' {
					fileMode |= pm.bit
				}
			case pm.char == 'x' && (c == 't' || c == 'T'):
				fileMode |= os.ModeSticky
				if c == 't' {
					fileMode |= pm.bit
				}
			}
		}
		switch fields[0][0] {
		case 'd':
			fileMode |= os.ModeDir
		case 'l':
			fileMode |= os.ModeSymlink
		}
	}

	if fields[1] == "folder" && fields[2] == "0" {
		e := &Entry{
			mode: fileMode | os.ModeDir,
			name: scanner.Remaining(),
		}
		if err := e.setTime(fields[3:6], now, loc); err != nil {
			return nil, err
		}

		return e, nil
	}

	if fields[1] == "0" {
		fields = append(fields, scanner.Next())
		e := &Entry{
			mode: fileMode,
			name: scanner.Remaining(),
		}

		if err := e.setSize(fields[2]); err != nil {
			return nil, errUnsupportedListLine
		}
		if err := e.setTime(fields[4:7], now, loc); err != nil {
			return nil, err
		}

		return e, nil
	}

	// Read two more fields
	fields = append(fields, scanner.NextFields(2)...)
	if len(fields) < 8 {
		return nil, errUnsupportedListLine
	}

	e := &Entry{
		mode: fileMode,
		name: scanner.Remaining(),
	}
	switch fields[0][0] {
	case '-':
		if err := e.setSize(fields[4]); err != nil {
			return nil, errUnsupportedListLine
		}
	case 'd':
		// type bit already set above
	case 'l':
		if i := strings.Index(e.name, " -> "); i > 0 {
			e.target = e.name[i+4:]
			e.name = e.name[:i]
		}
	default:
		return nil, errUnknownListEntryType
	}

	if err := e.setTime(fields[5:8], now, loc); err != nil {
		return nil, err
	}

	return e, nil
}

// parseDirListLine parses a directory line in a format based on the output of
// the MS-DOS DIR command.
func parseDirListLine(line string, now time.Time, loc *time.Location) (*Entry, error) {
	e := &Entry{}
	var err error

	// Try various time formats that DIR might use, and stop when one works.
	for _, format := range dirTimeFormats {
		if len(line) > len(format) {
			e.time, err = time.ParseInLocation(format, line[:len(format)], loc)
			if err == nil {
				line = line[len(format):]
				break
			}
		}
	}
	if err != nil {
		// None of the time formats worked.
		return nil, errUnsupportedListLine
	}

	line = strings.TrimLeft(line, " ")
	if strings.HasPrefix(line, "<DIR>") {
		e.mode |= os.ModeDir
		line = strings.TrimPrefix(line, "<DIR>")
	} else {
		space := strings.Index(line, " ")
		if space == -1 {
			return nil, errUnsupportedListLine
		}
		e.size, err = strconv.ParseInt(line[:space], 10, 64)
		if err != nil {
			return nil, errUnsupportedListLine
		}
		line = line[space:]
	}

	e.name = strings.TrimLeft(line, " ")
	return e, nil
}

// parseHostedFTPLine parses a directory line in the non-standard format used
// by hostedftp.com
// -r--------   0 user group     65222236 Feb 24 00:39 UABlacklistingWeek8.csv
// (The link count is inexplicably 0)
func parseHostedFTPLine(line string, now time.Time, loc *time.Location) (*Entry, error) {
	// Has the first field a length of 10 bytes?
	if strings.IndexByte(line, ' ') != 10 {
		return nil, errUnsupportedListLine
	}

	scanner := newScanner(line)
	fields := scanner.NextFields(2)

	if len(fields) < 2 || fields[1] != "0" {
		return nil, errUnsupportedListLine
	}

	// Set link count to 1 and attempt to parse as Unix.
	return parseLsListLine(fields[0]+" 1 "+scanner.Remaining(), now, loc)
}

// parseListLine parses the various non-standard format returned by the LIST
// FTP command.
func parseListLine(line string, now time.Time, loc *time.Location) (*Entry, error) {
	for _, f := range listLineParsers {
		e, err := f(line, now, loc)
		if err != errUnsupportedListLine {
			return e, err
		}
	}
	return nil, errUnsupportedListLine
}

func (e *Entry) setSize(str string) (err error) {
	e.size, err = strconv.ParseInt(str, 0, 64)
	return
}

func (e *Entry) setTime(fields []string, now time.Time, loc *time.Location) (err error) {
	if strings.Contains(fields[2], ":") { // contains time
		thisYear, _, _ := now.Date()
		timeStr := fmt.Sprintf("%s %s %d %s", fields[1], fields[0], thisYear, fields[2])
		e.time, err = time.ParseInLocation("_2 Jan 2006 15:04", timeStr, loc)

		/*
			On unix, `info ls` shows:

			10.1.6 Formatting file timestamps
			---------------------------------

			A timestamp is considered to be “recent” if it is less than six
			months old, and is not dated in the future.  If a timestamp dated today
			is not listed in recent form, the timestamp is in the future, which
			means you probably have clock skew problems which may break programs
			like ‘make’ that rely on file timestamps.
		*/
		if !e.time.Before(now.AddDate(0, 6, 0)) {
			e.time = e.time.AddDate(-1, 0, 0)
		}

	} else { // only the date
		if len(fields[2]) != 4 {
			return errUnsupportedListDate
		}
		timeStr := fmt.Sprintf("%s %s %s 00:00", fields[1], fields[0], fields[2])
		e.time, err = time.ParseInLocation("_2 Jan 2006 15:04", timeStr, loc)
	}
	return
}
