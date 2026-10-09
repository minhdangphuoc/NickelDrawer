package main

import (
	"archive/zip"
	"io"
	"os"
	"strconv"
	"testing"
)

type testZip struct{ writer *zip.Writer }

func newZip(out io.Writer) testZip { return testZip{writer: zip.NewWriter(out)} }

func (archive testZip) add(t *testing.T, name, content string) {
	t.Helper()
	method := zip.Deflate
	if name == "mimetype" {
		method = zip.Store
	}
	entry, err := archive.writer.CreateHeader(&zip.FileHeader{Name: name, Method: method})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := entry.Write([]byte(content)); err != nil {
		t.Fatal(err)
	}
}

func (archive testZip) close(t *testing.T) {
	t.Helper()
	if err := archive.writer.Close(); err != nil {
		t.Fatal(err)
	}
}

// strconvPid is this test process's pid, which is alive, so a lock holding it is not stale.
func strconvPid() string { return strconv.Itoa(os.Getpid()) }
