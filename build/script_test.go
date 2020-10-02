package main

import (
	"bytes"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	. "github.com/schwarzlichtbezirk/wpk"
)

var scrdir = envfmt("$(GOPATH)/src/github.com/schwarzlichtbezirk/wpk/test/")
var mediadir = scrdir + "media/"

// Test package content on nested and external files equivalent.
func CheckPackage(t *testing.T, wpkname string) {
	var err error
	var pack Package
	var fwpk *os.File

	// open temporary file for read/write
	if fwpk, err = os.Open(wpkname); err != nil {
		t.Fatal(err)
	}
	defer fwpk.Close()

	if err = pack.Read(fwpk); err != nil {
		t.Fatal(err)
	}
	if int(pack.TagNumber) != len(pack.Tags) {
		t.Fatalf("stored at header %d entries, realy got %d entries", pack.TagNumber, len(pack.Tags))
	}

	for _, tags := range pack.Tags {
		var path, _ = tags.String(TID_path)
		var link, is = tags.String(TID_link)
		if !is {
			t.Logf("found packed data #%d '%s'", tags.FID(), path)
			continue // skip file without link
		}
		var offset, size = tags.Record()

		var orig []byte
		if orig, err = ioutil.ReadFile(mediadir + link); err != nil {
			t.Fatal(err)
		}

		if tags.Size() != int64(len(orig)) {
			t.Errorf("size of file '%s' (%d) in package is defer from original (%d)",
				path, tags.Size(), len(orig))
		}

		var extr = make([]byte, size, size)
		var n int
		if n, err = fwpk.ReadAt(extr, offset); err != nil {
			t.Fatal(err)
		}
		if n != len(extr) {
			t.Errorf("can not extract content of file '%s' completely", path)
		}

		if !bytes.Equal(orig, extr) {
			t.Errorf("content of file '%s' is defer from original", path)
		}

		if t.Failed() {
			break
		}

		t.Logf("checkup #%d '%s' is ok", tags.FID(), path)
	}
}

// Test packdir script call.
func TestPackdir(t *testing.T) {
	var wpkname = filepath.Join(os.TempDir(), "packdir.wpk")
	defer os.Remove(wpkname)

	if err := mainluavm(scrdir + "packdir.lua"); err != nil {
		t.Fatal(err)
	}

	// make package file check up
	CheckPackage(t, wpkname)
}

// Test append package ability by scripts.
func TestSteps(t *testing.T) {
	var wpkname = filepath.Join(os.TempDir(), "steps.wpk")
	defer os.Remove(wpkname)

	if err := mainluavm(scrdir + "step1.lua"); err != nil {
		t.Fatal(err)
	}

	// make package file check up
	CheckPackage(t, wpkname)

	if err := mainluavm(scrdir + "step2.lua"); err != nil {
		t.Fatal(err)
	}

	// make package file check up
	CheckPackage(t, wpkname)
}

// The End.
