package luawpk_test

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/schwarzlichtbezirk/wpk"
	lw "github.com/schwarzlichtbezirk/wpk/luawpk"
)

type (
	TID_t    = uint16
	TSize_t  = uint16
	TSSize_t = uint16
)

var scrdir = wpk.Envfmt("${GOPATH}/src/github.com/schwarzlichtbezirk/wpk/test/")
var mediadir = scrdir + "media/"

// Test package content on nested and external files equivalent.
func CheckPackage(t *testing.T, wpkname string) {
	var err error
	var pack wpk.Package[TID_t, TSize_t, TSSize_t]
	var fwpk *os.File

	// open temporary file for read/write
	if fwpk, err = os.Open(wpkname); err != nil {
		t.Fatal(err)
	}
	defer fwpk.Close()

	if err = pack.OpenFTT(fwpk); err != nil {
		t.Fatal(err)
	}

	if ts, ok := pack.Tagset(""); ok {
		var offset, _ = wpk.UintTagset[TID_t, TSize_t, wpk.FOffset_t](ts, wpk.TIDoffset)
		var size, _ = wpk.UintTagset[TID_t, TSize_t, wpk.FSize_t](ts, wpk.TIDsize)
		var label, _ = ts.String(wpk.TIDlabel)
		t.Logf("package info: offset %d, size %d, label '%s'", offset, size, label)
	}
	var n = 0
	pack.Enum(func(fkey string, ts *wpk.Tagset_t[TID_t, TSize_t]) bool {
		var ok bool
		var offset, _ = wpk.UintTagset[TID_t, TSize_t, wpk.FOffset_t](ts, wpk.TIDoffset)
		var size, _ = wpk.UintTagset[TID_t, TSize_t, wpk.FSize_t](ts, wpk.TIDsize)
		var fpath = ts.Path()
		n++

		if ok = ts.Has(wpk.TIDmtime); !ok {
			t.Logf("found packed data #%d '%s'", n, fpath)
			return true // skip packed data
		}

		var link wpk.Tag_t
		if link, ok = ts.Get(wpk.TIDlink); !ok {
			t.Fatalf("found file without link #%d '%s'", n, fpath)
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
		var readed int
		if readed, err = fwpk.ReadAt(extr, int64(offset)); err != nil {
			t.Fatal(err)
		}
		if readed != len(extr) {
			t.Errorf("can not extract content of file '%s' completely", fpath)
		}
		if !bytes.Equal(orig, extr) {
			t.Errorf("content of file '%s' is defer from original", fpath)
		}

		if t.Failed() {
			return false
		}

		t.Logf("checkup #%d '%s' is ok", n, fpath)
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
