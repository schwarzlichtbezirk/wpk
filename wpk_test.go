package wpk_test

import (
	"bytes"
	"io"
	"io/fs"
	"os"
	"testing"

	"github.com/schwarzlichtbezirk/wpk"
)

const mediadir = "testdata/media/"

var testpack = wpk.TempPath("testpack.wpk")
var testpkgt = wpk.TempPath("testpack.wpt")
var testpkgf = wpk.TempPath("testpack.wpf")

var memdata = map[string][]byte{
	"sample.txt": wpk.S2B("The quick brown fox jumps over the lazy dog"),
	"array.dat": {
		0, 1, 2, 3, 4, 5, 6, 7, 8, 9,
		100, 101, 102, 103, 104, 105, 106, 107, 108, 109,
		200, 201, 202, 203, 204, 205, 206, 207, 208, 209,
	},
}

// Test package content on nested and external files equivalent.
func CheckPackage(t *testing.T, fwpt, fwpf *os.File, tagsnum int) {
	var err error

	// Open package files tags table
	var pkg = wpk.NewPackage()
	if err = pkg.OpenStream(fwpt); err != nil {
		t.Fatal(err)
	}

	var label, _ = pkg.GetInfo().TagStr(wpk.TIDlabel)
	t.Logf("package info: data size %d, label '%s'", pkg.DataSize(), label)

	var realtagsnum int
	pkg.Enum(func(fkey string, ts wpk.TagsetRaw) bool {
		var link, haslink = ts.TagStr(wpk.TIDlink)
		var offset, size = ts.Pos()
		var fid, _ = ts.TagUint(wpk.TIDfid)
		realtagsnum++

		var orig []byte
		if haslink {
			if orig, err = os.ReadFile(link); err != nil {
				t.Fatal(err)
			}
		} else {
			var is bool
			if orig, is = memdata[fkey]; !is {
				t.Fatalf("memory block named as '%s' not found", fkey)
			}
		}

		if size != wpk.Uint(len(orig)) {
			t.Errorf("size of file '%s' (%d) in package is defer from original (%d)",
				fkey, size, len(orig))
		}

		var extr = make([]byte, size)
		var n int
		if n, err = fwpf.ReadAt(extr, int64(offset)); err != nil {
			t.Fatal(err)
		}
		if n != len(extr) {
			t.Errorf("can not extract content of file '%s' completely", fkey)
		}
		if !bytes.Equal(orig, extr) {
			t.Errorf("content of file '%s' is defer from original", fkey)
		}

		if t.Failed() {
			return false
		}

		if haslink {
			t.Logf("check file #%d '%s' is ok", fid, fkey)
		} else {
			t.Logf("check data #%d '%s' is ok", fid, fkey)
		}
		return true
	})
	if realtagsnum != tagsnum {
		t.Fatalf("expected %d entries in package, really got %d entries", tagsnum, realtagsnum)
	}
}

// Test package SetInfo function and GetPackageInfo.
func TestInfo(t *testing.T) {
	var err error
	var fwpk *os.File
	var pkg = wpk.NewPackage()

	const (
		label  = "empty-package"
		link   = "github.com/schwarzlichtbezirk/wpk"
		author = "schwarzlichtbezirk"
	)

	defer os.Remove(testpack)

	// open temporary file for read/write
	if fwpk, err = os.OpenFile(testpack, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644); err != nil {
		t.Fatal(err)
	}
	defer fwpk.Close()

	// starts new package
	if err = pkg.Begin(fwpk, nil); err != nil {
		t.Fatal(err)
	}
	// put package info somewhere at the code before finalize
	pkg.SetInfo(wpk.TagsetRaw{}.
		Put(wpk.TIDlabel, wpk.StrTag(label)).
		Put(wpk.TIDlink, wpk.StrTag(link)).
		Put(wpk.TIDauthor, wpk.StrTag(author)))
	// finalize
	if err = pkg.Sync(fwpk, nil); err != nil {
		t.Fatal(err)
	}

	// at the end checkup package info
	var ts wpk.TagsetRaw
	if _, ts, err = wpk.GetPackageInfo(fwpk); err != nil {
		t.Fatal(err)
	}
	var ok bool
	var str string
	if str, ok = ts.TagStr(wpk.TIDlabel); !ok {
		t.Fatal("label tag not found in package info")
	}
	if str != label {
		t.Fatal("label in package info is not equal to original")
	}
	if str, ok = ts.TagStr(wpk.TIDlink); !ok {
		t.Fatal("link tag not found in package info")
	}
	if str != link {
		t.Fatal("link in package info is not equal to original")
	}
	if str, ok = ts.TagStr(wpk.TIDauthor); !ok {
		t.Fatal("author tag not found in package info")
	}
	if str != author {
		t.Fatal("author in package info is not equal to original")
	}
}

// Test PackDir function work.
func TestPackDir(t *testing.T) {
	var err error
	var fwpk *os.File
	var tagsnum = 0
	var fidcount wpk.Uint
	var pkg = wpk.NewPackage()

	defer os.Remove(testpack)

	// open temporary file for read/write
	if fwpk, err = os.OpenFile(testpack, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644); err != nil {
		t.Fatal(err)
	}
	defer fwpk.Close()

	// starts new package
	if err = pkg.Begin(fwpk, nil); err != nil {
		t.Fatal(err)
	}
	// put package info somewhere before finalize
	pkg.SetInfo(wpk.TagsetRaw{}.
		Put(wpk.TIDlabel, wpk.StrTag("packed-dir")))
	// put media directory to file
	if err = pkg.PackDir(fwpk, mediadir, "", func(pkg *wpk.Package, r io.ReadSeeker, ts wpk.TagsetRaw) error {
		tagsnum++
		fidcount++
		var fpath = mediadir + ts.Path()
		pkg.SetupTagset(ts.
			Put(wpk.TIDfid, wpk.UintTag(fidcount)).
			Put(wpk.TIDlink, wpk.StrTag(fpath)))
		t.Logf("put file #%d '%s', %d bytes", fidcount, ts.Path(), ts.Size())
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	// finalize
	if err = pkg.Sync(fwpk, nil); err != nil {
		t.Fatal(err)
	}

	// make package file check up
	CheckPackage(t, fwpk, fwpk, tagsnum)
}

// Test to read abnormal closed package in database mode.
func TestBrokenDB(t *testing.T) {
	var err error
	var fwpt, fwpf *os.File
	var pkg = wpk.NewPackage()

	defer os.Remove(testpkgt)
	defer os.Remove(testpkgf)

	// open temporary header file for read/write
	if fwpt, err = os.OpenFile(testpkgt, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644); err != nil {
		t.Fatal(err)
	}
	defer fwpt.Close()

	// open temporary data file for read/write
	if fwpf, err = os.OpenFile(testpkgf, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644); err != nil {
		t.Fatal(err)
	}
	defer fwpf.Close()

	// starts new package
	if err = pkg.Begin(fwpt, fwpf); err != nil {
		t.Fatal(err)
	}

	// try to read files tags table in empty package
	if err = pkg.OpenStream(fwpt); err != nil {
		t.Fatal(err)
	}

	// put package info
	pkg.SetInfo(wpk.TagsetRaw{}.
		Put(wpk.TIDlabel, wpk.StrTag("broken-pkg")))

	// put somewhat
	for name, data := range memdata {
		if _, err = pkg.PackData(fwpf, bytes.NewReader(data), name); err != nil {
			t.Fatal(err)
		}
	}

	// finalize
	if err = pkg.Sync(fwpt, fwpf); err != nil {
		t.Fatal(err)
	}

	// try to read files tags table with some data
	if err = pkg.OpenStream(fwpt); err != nil {
		t.Fatal(err)
	}
}

// Test package writing to splitted header and data files.
func TestPackDirSplit(t *testing.T) {
	var err error
	var fwpt, fwpf *os.File
	var tagsnum = 0
	var fidcount wpk.Uint
	var pkg = wpk.NewPackage()

	defer os.Remove(testpkgt)
	defer os.Remove(testpkgf)

	// open temporary header file for read/write
	if fwpt, err = os.OpenFile(testpkgt, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644); err != nil {
		t.Fatal(err)
	}
	defer fwpt.Close()

	// open temporary data file for read/write
	if fwpf, err = os.OpenFile(testpkgf, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644); err != nil {
		t.Fatal(err)
	}
	defer fwpf.Close()

	// starts new package
	if err = pkg.Begin(fwpt, fwpf); err != nil {
		t.Fatal(err)
	}
	// put package info somewhere before finalize
	pkg.SetInfo(wpk.TagsetRaw{}.
		Put(wpk.TIDlabel, wpk.StrTag("splitted-pkg")))
	// put media directory to file
	if err = pkg.PackDir(fwpf, mediadir, "", func(pkg *wpk.Package, r io.ReadSeeker, ts wpk.TagsetRaw) error {
		tagsnum++
		fidcount++
		var fpath = mediadir + ts.Path()
		pkg.SetupTagset(ts.
			Put(wpk.TIDfid, wpk.UintTag(fidcount)).
			Put(wpk.TIDlink, wpk.StrTag(fpath)))
		t.Logf("put file #%d '%s', %d bytes", fidcount, ts.Path(), ts.Size())
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	// finalize
	if err = pkg.Sync(fwpt, fwpf); err != nil {
		t.Fatal(err)
	}

	// make package file check up
	CheckPackage(t, fwpt, fwpf, tagsnum)
}

// Test ability of files sequence packing, and make alias.
func TestPutFiles(t *testing.T) {
	var err error
	var fwpk *os.File
	var tagsnum = 0
	var fidcount wpk.Uint
	var pkg = wpk.NewPackage()

	defer os.Remove(testpack)

	// helper functions
	var putfile = func(name string) {
		var fpath = mediadir + name
		var file fs.File
		if file, err = os.Open(fpath); err != nil {
			t.Fatal(err)
		}
		defer file.Close()

		var ts wpk.TagsetRaw
		if ts, err = pkg.PackFile(fwpk, file, name); err != nil {
			t.Fatal(err)
		}

		tagsnum++
		fidcount++
		pkg.SetupTagset(ts.
			Put(wpk.TIDfid, wpk.UintTag(fidcount)).
			Put(wpk.TIDlink, wpk.StrTag(fpath)))
		var size = ts.Size()
		t.Logf("put file #%d '%s', %d bytes", fidcount, name, size)
	}
	var putdata = func(name string, data []byte) {
		var r = bytes.NewReader(data)

		var ts wpk.TagsetRaw
		if ts, err = pkg.PackData(fwpk, r, name); err != nil {
			t.Fatal(err)
		}

		tagsnum++
		fidcount++
		pkg.SetupTagset(ts.Put(wpk.TIDfid, wpk.UintTag(fidcount)))
		var size = ts.Size()
		t.Logf("put data #%d '%s', %d bytes", fidcount, name, size)
	}
	var putalias = func(oldname, newname string) {
		if err = pkg.PutAlias(oldname, newname); err != nil {
			t.Fatal(err)
		}
		tagsnum++
		t.Logf("put alias '%s' to '%s'", newname, oldname)
	}
	var delalias = func(name string) {
		if _, ok := pkg.DelTagset(name); !ok {
			t.Fatalf("alias '%s' not deleted", name)
		}
		tagsnum--
		t.Logf("del alias '%s'", name)
	}

	// open temporary file for read/write
	if fwpk, err = os.OpenFile(testpack, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644); err != nil {
		t.Fatal(err)
	}
	defer fwpk.Close()

	// starts new package
	if err = pkg.Begin(fwpk, nil); err != nil {
		t.Fatal(err)
	}
	// put package info somewhere before finalize
	pkg.SetInfo(wpk.TagsetRaw{}.
		Put(wpk.TIDlabel, wpk.StrTag("put-files")))
	// put content
	putfile("bounty.jpg")
	putfile("img1/claustral.jpg")
	putfile("img1/Qarataşlar.jpg")
	putfile("img2/marble.jpg")
	putfile("img2/Uzuncı.jpg")
	putalias("img1/claustral.jpg", "basaltbay.jpg")
	for name, data := range memdata {
		putdata(name, data)
	}
	putalias("img1/claustral.jpg", "jasper.jpg")
	delalias("basaltbay.jpg")
	// finalize
	if err = pkg.Sync(fwpk, nil); err != nil {
		t.Fatal(err)
	}

	// make package file check up
	CheckPackage(t, fwpk, fwpk, tagsnum)

	// check alias existence
	if _, ok := pkg.GetTagset("jasper.jpg"); !ok {
		t.Fatal("'jasper.jpg' alias not found")
	}
	if _, ok := pkg.GetTagset("basaltbay.jpg"); ok {
		t.Fatal("'basaltbay.jpg' alias not deleted")
	}

	// check renamedir call
	var count int
	if count, err = pkg.RenameDir("img2", "img3", false); err != nil {
		t.Fatal(err)
	}
	if count != 2 {
		t.Fatalf("expected 2 files to rename directory from 'img2' to 'img3', got %d", count)
	}
	if _, ok := pkg.GetTagset("img3/marble.jpg"); !ok {
		t.Fatal("'img3/marble.jpg' not found")
	}
	if _, ok := pkg.GetTagset("img3/Uzuncı.jpg"); !ok {
		t.Fatal("'img3/Uzuncı.jpg' not found")
	}
}

// Test to make package in two steps on single package open:
// creates package file, make package, do some job,
// then append new files to existing package.
func TestAppendContinues(t *testing.T) {
	var err error
	var fwpk *os.File
	var tagsnum = 0
	var fidcount wpk.Uint
	var pkg = wpk.NewPackage()

	defer os.Remove(testpack)

	// helper functions
	var putfile = func(name string) {
		var fpath = mediadir + name
		var file fs.File
		if file, err = os.Open(fpath); err != nil {
			t.Fatal(err)
		}
		defer file.Close()

		var ts wpk.TagsetRaw
		if ts, err = pkg.PackFile(fwpk, file, name); err != nil {
			t.Fatal(err)
		}

		tagsnum++
		fidcount++
		pkg.SetupTagset(ts.
			Put(wpk.TIDfid, wpk.UintTag(fidcount)).
			Put(wpk.TIDlink, wpk.StrTag(fpath)))
		var size = ts.Size()
		t.Logf("put file #%d '%s', %d bytes", fidcount, name, size)
	}

	// open temporary file for read/write
	if fwpk, err = os.OpenFile(testpack, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644); err != nil {
		t.Fatal(err)
	}
	defer fwpk.Close()

	// starts new package
	if err = pkg.Begin(fwpk, nil); err != nil {
		t.Fatal(err)
	}
	// put package info somewhere before finalize
	pkg.SetInfo(wpk.TagsetRaw{}.
		Put(wpk.TIDlabel, wpk.StrTag("append-continues")))
	// put content
	putfile("bounty.jpg")
	putfile("img1/claustral.jpg")
	putfile("img1/Qarataşlar.jpg")
	// finalize
	if err = pkg.Sync(fwpk, nil); err != nil {
		t.Fatal(err)
	}

	// make package file check up
	CheckPackage(t, fwpk, fwpk, tagsnum)

	//
	// here can be any job using package
	//

	// starts append to existing package
	if err = pkg.Append(fwpk, nil); err != nil {
		t.Fatal(err)
	}
	// put content
	putfile("img2/marble.jpg")
	putfile("img2/Uzuncı.jpg")
	// finalize
	if err = pkg.Sync(fwpk, nil); err != nil {
		t.Fatal(err)
	}

	// make package file check up
	CheckPackage(t, fwpk, fwpk, tagsnum)
}

// Test to make package in two steps on twice package opens:
// creates package file, make package, close file,
// then open package file again and append new files.
func TestAppendDiscrete(t *testing.T) {
	var err error
	var fwpk *os.File
	var tagsnum = 0
	var fidcount wpk.Uint
	var pkg = wpk.NewPackage()

	defer os.Remove(testpack)

	// helper functions
	var putfile = func(name string) {
		var fpath = mediadir + name
		var file fs.File
		if file, err = os.Open(fpath); err != nil {
			t.Fatal(err)
		}
		defer file.Close()

		var ts wpk.TagsetRaw
		if ts, err = pkg.PackFile(fwpk, file, name); err != nil {
			t.Fatal(err)
		}

		tagsnum++
		fidcount++
		pkg.SetupTagset(ts.
			Put(wpk.TIDfid, wpk.UintTag(fidcount)).
			Put(wpk.TIDlink, wpk.StrTag(fpath)))
		var size = ts.Size()
		t.Logf("put file #%d '%s', %d bytes", fidcount, name, size)
	}

	t.Run("step1", func(t *testing.T) {
		// open temporary file for read/write
		if fwpk, err = os.OpenFile(testpack, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644); err != nil {
			t.Fatal(err)
		}
		defer fwpk.Close()

		// starts new package
		if err = pkg.Begin(fwpk, nil); err != nil {
			t.Fatal(err)
		}
		// put package info somewhere before finalize
		var ts = append(wpk.TagsetRaw{}, pkg.GetInfo()...) // append prevents rewriting data at solid slice with FTT
		ts, _ = ts.Set(wpk.TIDlabel, wpk.StrTag("discrete-step#1"))
		pkg.SetInfo(ts)
		// put content
		putfile("bounty.jpg")
		putfile("img1/claustral.jpg")
		putfile("img1/Qarataşlar.jpg")
		// finalize
		if err = pkg.Sync(fwpk, nil); err != nil {
			t.Fatal(err)
		}

		// make package file check up
		CheckPackage(t, fwpk, fwpk, tagsnum)
	})

	//
	// here can be any job using package
	//

	t.Run("step2", func(t *testing.T) {
		// open temporary file for read/write
		if fwpk, err = os.OpenFile(testpack, os.O_RDWR, 0644); err != nil {
			t.Fatal(err)
		}
		defer fwpk.Close()

		// read package content again.
		// pkg value already contains data from previous step
		// and this call can be skipped,
		// but we want to test here read functionality
		if err = pkg.OpenStream(fwpk); err != nil {
			t.Fatal(err)
		}

		// starts append to existing package
		if err = pkg.Append(fwpk, nil); err != nil {
			t.Fatal(err)
		}
		// put package info somewhere before finalize
		var ts = append(wpk.TagsetRaw{}, pkg.GetInfo()...) // append prevents rewriting data at solid slice with FTT
		ts, _ = ts.Set(wpk.TIDlabel, wpk.StrTag("discrete-step#2"))
		pkg.SetInfo(ts)
		// put content
		putfile("img2/marble.jpg")
		putfile("img2/Uzuncı.jpg")
		// finalize
		if err = pkg.Sync(fwpk, nil); err != nil {
			t.Fatal(err)
		}

		// make package file check up
		CheckPackage(t, fwpk, fwpk, tagsnum)
	})
}

// The End.
