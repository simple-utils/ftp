# goftp #

[![Units tests](https://github.com/simple-utils/ftp/actions/workflows/unit_tests.yaml/badge.svg)](https://github.com/simple-utils/ftp/actions/workflows/unit_tests.yaml)
[![Go Reference](https://pkg.go.dev/badge/github.com/simple-utils/ftp.svg)](https://pkg.go.dev/github.com/simple-utils/ftp)

A FTP client package for Go.

## About this fork ##

This is a fork of [`github.com/jlaffaye/ftp`](https://github.com/jlaffaye/ftp),
created because the upstream repository has become inactive.

The goal of this fork is to keep the package maintained: integrate useful
community patches, keep dependencies and the supported Go versions up to date,
and continue accepting fixes.

## Install ##

```
go get -u github.com/simple-utils/ftp
```

## Documentation ##

https://pkg.go.dev/github.com/simple-utils/ftp

## Breaking changes since `jlaffaye/ftp` ##

If you are migrating from `github.com/jlaffaye/ftp`, the public API differs in
the following ways:

- `Entry` no longer exposes its fields. Use the `os.FileInfo` accessors
  `Name()`, `Size()`, `Mode()`, `ModTime()`, `IsDir()` and `Sys()`. Symbolic
  link targets are returned by `Target()`.
- `Entry.Size()` returns `int64` (was `uint64`) to match `os.FileInfo`.
- The `EntryType` enum and its constants (`EntryTypeFile`, `EntryTypeFolder`,
  `EntryTypeLink`) are gone. Use `entry.IsDir()` or
  `entry.Mode()&os.ModeSymlink != 0` instead.
- `Entry.Mode()` now also carries `os.ModeDir`, `os.ModeSymlink`,
  `os.ModeSetuid`, `os.ModeSetgid` and `os.ModeSticky` when the server
  reports them.
- `Chown` takes `(path, owner, group string)`. Pass an empty `group` to send
  just the owner.

## Example ##

```go
c, err := ftp.Dial("ftp.example.org:21", ftp.DialWithTimeout(5*time.Second))
if err != nil {
    log.Fatal(err)
}

if err := c.Login("anonymous", "anonymous"); err != nil {
    log.Fatal(err)
}
defer c.Quit()
```

### Store a file ###

```go
data := bytes.NewBufferString("Hello World")
if err := c.Stor("test-file.txt", data); err != nil {
    log.Fatal(err)
}
```

### Read a file ###

```go
r, err := c.Retr("test-file.txt")
if err != nil {
    log.Fatal(err)
}
defer r.Close()

buf, err := io.ReadAll(r)
if err != nil {
    log.Fatal(err)
}
fmt.Println(string(buf))
```

### List a directory ###

```go
entries, err := c.List("/pub")
if err != nil {
    log.Fatal(err)
}
for _, e := range entries {
    fmt.Printf("%s  %10d  %s\n", e.Mode(), e.Size(), e.Name())
}
```

### Walk a directory tree ###

```go
w := c.Walk("/pub")
for w.Next() {
    if err := w.Err(); err != nil {
        log.Fatal(err)
    }
    fmt.Println(w.Path())
}
```

### Change permissions and ownership ###

`Chmod` and `Chown` use the non-standard `SITE CHMOD` / `SITE CHOWN`
extensions and are not supported by every server.

```go
if err := c.Chmod("test-file.txt", 0o644); err != nil {
    log.Fatal(err)
}

// owner only
if err := c.Chown("test-file.txt", "alice", ""); err != nil {
    log.Fatal(err)
}

// owner and group
if err := c.Chown("test-file.txt", "alice", "staff"); err != nil {
    log.Fatal(err)
}
```

If the server enumerated its `SITE` subcommands in `FEAT` and the requested
one is missing, the call returns an error wrapping
`ftp.ErrSiteCommandNotSupported` without contacting the server. You can probe
support up front:

```go
if !c.IsSiteCommandSupported("CHMOD") {
    // server did not advertise SITE CHMOD; calling Chmod may still work
    // but is not guaranteed
}
```

## License ##

Distributed under the ISC license, the same as the upstream project.
See [LICENSE](LICENSE) for details.
