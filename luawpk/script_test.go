package luawpk_test

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/schwarzlichtbezirk/wpk"
	lw "github.com/schwarzlichtbezirk/wpk/luawpk"
)

var scrdir = wpk.Envfmt("${GOPATH}/src/github.com/schwarzlichtbezirk/wpk/test/")
var mediadir = scrdir + "media/"

// Test package content on nested and external files equivalent.
func CheckPackage(t *testing.T, wpkname string) {
	var err error
	var pack wpk.Writer
	var fwpk *os.File

	// open temporary file for read/write
	if fwpk, err = os.Open(wpkname); err != nil {
		t.Fatal(err)
	}
	defer fwpk.Close()

	if err = pack.OpenFTT(fwpk); err != nil {
		t.Fatal(err)
	}

	pack.Enum(func(fkey string, ts *wpk.Tagset_t) bool {
		var ok bool
		var offset, _ = ts.FOffset()
		var size, _ = ts.FSize()
		var fid, _ = ts.FID()
		var fpath = ts.Path()

		if fid == 0 {
			var label, _ = ts.String(wpk.TIDlabel)
			t.Logf("package info: offset %d, size %d, label '%s'", offset, size, label)
			return true
		}

		if ok = ts.Has(wpk.TIDcreated); !ok {
			t.Logf("found packed data #%d '%s'", fid, fpath)
			return true // skip packed data
		}

		var link wpk.Tag_t
		if link, ok = ts.Get(wpk.TIDlink); !ok {
			t.Fatalf("found file without link #%d '%s'", fid, fpath)
		}

		var orig []byte
		if orig, err = os.ReadFile(mediadir + string(link)); err != nil {
			t.Fatal(err)
		}

		if size != wpk.FSize_t(len(orig)) {
			t.Errorf("size of file '%s' (%d) in package is defer from original (%d)",
				fpath, size, len(orig))
		}

		var extr = make([]byte, size)
		var n int
		if n, err = fwpk.ReadAt(extr, int64(offset)); err != nil {
			t.Fatal(err)
		}
		if n != len(extr) {
			t.Errorf("can not extract content of file '%s' completely", fpath)
		}
		if !bytes.Equal(orig, extr) {
			t.Errorf("content of file '%s' is defer from original", fpath)
		}

		if t.Failed() {
			return false
		}

		t.Logf("checkup #%d '%s' is ok", fid, fpath)
		return true
	})
}

// Test packdir script call.
func TestPackdir(t *testing.T) {
	var wpkname = filepath.Join(os.TempDir(), "packdir.wpk")
	defer os.Remove(wpkname)

	if err := lw.RunLuaVM(scrdir + "packdir.lua"); err != nil {
		t.Fatal(err)
	}

	// make package file check up
	CheckPackage(t, wpkname)
}

// Test append package ability by scripts.
func TestSteps(t *testing.T) {
	var wpkname = filepath.Join(os.TempDir(), "steps.wpk")
	defer os.Remove(wpkname)

	if err := lw.RunLuaVM(scrdir + "step1.lua"); err != nil {
		t.Fatal(err)
	}

	// make package file check up
	CheckPackage(t, wpkname)

	if err := lw.RunLuaVM(scrdir + "step2.lua"); err != nil {
		t.Fatal(err)
	}

	// make package file check up
	CheckPackage(t, wpkname)
}

// The End.
