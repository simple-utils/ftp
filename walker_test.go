package ftp

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWalkReturnsCorrectlyPopulatedWalker(t *testing.T) {
	mock, err := newFtpMock(t, "127.0.0.1")
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	c, cErr := Connect(mock.Addr())
	if cErr != nil {
		t.Fatal(err)
	}

	w := c.Walk("root")

	assert.Equal(t, "root/", w.root)
	assert.Equal(t, &c, &w.serverConn)
}

func TestFieldsReturnCorrectData(t *testing.T) {
	w := Walker{
		cur: &item{
			path: "/root/",
			err:  fmt.Errorf("this is an error"),
			entry: &Entry{
				name: "root",
				size: 123,
				time: time.Now(),
				mode: os.ModeDir,
			},
		},
	}

	assert.Equal(t, "this is an error", w.Err().Error())
	assert.Equal(t, "/root/", w.Path())
	assert.True(t, w.Stat().IsDir())
}

func TestSkipDirIsCorrectlySet(t *testing.T) {
	w := Walker{}

	w.SkipDir()

	assert.Equal(t, false, w.descend)
}

func TestNoDescendDoesNotAddToStack(t *testing.T) {
	mock, err := newFtpMock(t, "127.0.0.1")
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	c, cErr := Connect(mock.Addr())
	if cErr != nil {
		t.Fatal(err)
	}

	w := c.Walk("/root")
	w.cur = &item{
		path: "/root/",
		err:  nil,
		entry: &Entry{
			name: "root",
			size: 123,
			time: time.Now(),
			mode: os.ModeDir,
		},
	}

	w.stack = []*item{
		{
			path: "file",
			err:  nil,
			entry: &Entry{
				name: "file",
				size: 123,
				time: time.Now(),
			},
		},
	}

	w.SkipDir()

	result := w.Next()

	assert.Equal(t, true, result, "Result should return true")
	assert.Equal(t, 0, len(w.stack))
	assert.Equal(t, true, w.descend)
}

func TestEmptyStackReturnsFalse(t *testing.T) {
	assert, require := assert.New(t), require.New(t)

	mock, err := newFtpMock(t, "127.0.0.1")
	require.Nil(err)
	defer mock.Close()

	c, cErr := Connect(mock.Addr())
	require.Nil(cErr)

	w := c.Walk("/root")

	w.cur = &item{
		path: "/root/",
		err:  nil,
		entry: &Entry{
			name: "root",
			size: 123,
			time: time.Now(),
			mode: os.ModeDir,
		},
	}

	w.stack = []*item{}

	w.SkipDir()

	result := w.Next()

	assert.Equal(false, result, "Result should return false")
}

func TestCurAndStackSetCorrectly(t *testing.T) {
	assert, require := assert.New(t), require.New(t)

	mock, err := newFtpMock(t, "127.0.0.1")
	require.Nil(err)
	defer mock.Close()

	c, cErr := Connect(mock.Addr())
	require.Nil(cErr)

	w := c.Walk("/root")
	w.cur = &item{
		path: "root/file1",
		err:  nil,
		entry: &Entry{
			name: "file1",
			size: 123,
			time: time.Now(),
		},
	}

	w.stack = []*item{
		{
			path: "file",
			err:  nil,
			entry: &Entry{
				name: "file",
				size: 123,
				time: time.Now(),
			},
		},
		{
			path: "root/file1",
			err:  nil,
			entry: &Entry{
				name: "file1",
				size: 123,
				time: time.Now(),
			},
		},
	}

	result := w.Next()
	assert.Equal(true, result, "Result should return true")

	result = w.Next()

	assert.Equal(true, result, "Result should return true")
	assert.Equal(0, len(w.stack))
	assert.Equal("file", w.cur.entry.Name())
}

func TestCurInit(t *testing.T) {
	mock, err := newFtpMock(t, "127.0.0.1")
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	c, cErr := Connect(mock.Addr())
	if cErr != nil {
		t.Fatal(err)
	}

	w := c.Walk("/root")

	result := w.Next()

	// mock fs has one file 'lo'

	assert.Equal(t, true, result, "Result should return false")
	assert.Equal(t, 0, len(w.stack))
	assert.Equal(t, "/root/lo", w.Path())
}
