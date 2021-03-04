package wpk_test

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/schwarzlichtbezirk/wpk"
)

var mediadir = wpk.Envfmt("${GOPATH}/src/github.com/schwarzlichtbezirk/wpk/test/media/")
var testpack = filepath.Join(os.TempDir(), "testpack.wpk")

var memdata = map[string][]byte{
	"sample.txt": []byte("The quick brown fox jumps over the lazy dog"),
	"array.dat": {
		0, 1, 2, 3, 4, 5, 6, 7, 8, 9,
		100, 101, 102, 103, 104, 105, 106, 107, 108, 109,
		200, 201, 202, 203, 204, 205, 206, 207, 208, 209,
	},
}

// Test package content on nested and external files equivalent.
func CheckPackage(t *testing.T, fwpk *os.File, tagsnum int) {
	var err error
	var pack wpk.Writer

	if err = pack.Read(fwpk); err != nil {
		t.Fatal(err)
	}
	if int(pack.TagNumber) != len(pack.Tags) {
		t.Fatalf("stored at header %d entries, realy got %d entries", pack.TagNumber, len(pack.Tags))
	}
	if len(pack.Tags) != tagsnum {
		t.Fatalf("expected %d entries in package, realy got %d entries", tagsnum, len(pack.Tags))
	}

	for _, tags := range pack.Tags {
		var _, isfile = tags[wpk.TIDcreated]
		var kpath = tags.Path()
		var link, is = tags[wpk.TIDlink]
		if isfile && !is {
			t.Fatalf("found file without link #%d '%s'", tags.FID(), kpath)
		}
		var offset, size = tags.Offset(), tags.Size()

		var orig []byte
		if isfile {
			if orig, err = os.ReadFile(mediadir + string(link)); err != nil {
				t.Fatal(err)
			}
		} else {
			var is bool
			if orig, is = memdata[kpath]; !is {
				t.Fatalf("memory block named as '%s' not found", kpath)
			}
		}

		if size != int64(len(orig)) {
			t.Errorf("size of file '%s' (%d) in package is defer from original (%d)",
				kpath, size, len(orig))
		}

		var extr = make([]byte, size, size)
		var n int
		if n, err = fwpk.ReadAt(extr, offset); err != nil {
			t.Fatal(err)
		}
		if n != len(extr) {
			t.Errorf("can not extract content of file '%s' completely", kpath)
		}
		if !bytes.Equal(orig, extr) {
			t.Errorf("content of file '%s' is defer from original", kpath)
		}

		if t.Failed() {
			break
		}

		if isfile {
			t.Logf("check file #%d '%s' is ok", tags.FID(), kpath)
		} else {
			t.Logf("check data #%d '%s' is ok", tags.FID(), kpath)
		}
	}
}

// Test PackDir function work.
func TestPackDir(t *testing.T) {
	var err error
	var pack wpk.Writer
	var fwpk *os.File
	var tagsnum = 0

	defer os.Remove(testpack)

	// open temporary file for read/write
	if fwpk, err = os.OpenFile(testpack, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644); err != nil {
		t.Fatal(err)
	}
	defer fwpk.Close()

	// starts new package
	if err = pack.Begin(fwpk); err != nil {
		t.Fatal(err)
	}
	// put media directory to file
	if err = pack.PackDir(fwpk, mediadir, "", func(fi os.FileInfo, fname, fpath string) bool {
		if !fi.IsDir() {
			tagsnum++
			t.Logf("put file #%d '%s', %d bytes", pack.RecNumber+1, fname, fi.Size())
		}
		return true
	}); err != nil {
		t.Fatal(err)
	}
	// finalize
	if err = pack.Complete(fwpk); err != nil {
		t.Fatal(err)
	}

	// make package file check up
	CheckPackage(t, fwpk, tagsnum)
}

// Test ability of files sequence packing, and make alias.
func TestPutFiles(t *testing.T) {
	var err error
	var pack wpk.Writer
	var fwpk *os.File
	var tagsnum = 0

	defer os.Remove(testpack)

	// helper functions
	var putfile = func(name string) {
		var file *os.File
		if file, err = os.Open(mediadir + name); err != nil {
			t.Fatal(err)
		}
		defer file.Close()

		var tags wpk.Tagset
		if tags, err = pack.PackFile(fwpk, file, name); err != nil {
			t.Fatal(err)
		}

		tagsnum++
		t.Logf("put file #%d '%s', %d bytes", pack.RecNumber, name, tags.Size())
	}
	var putdata = func(name string, data []byte) {
		var r = bytes.NewReader(data)

		var tags wpk.Tagset
		if tags, err = pack.PackData(fwpk, r, name); err != nil {
			t.Fatal(err)
		}

		tagsnum++
		t.Logf("put data #%d '%s', %d bytes", pack.RecNumber, name, tags.Size())
	}
	var putalias = func(oldname, newname string) {
		if err = pack.PutAlias(oldname, newname); err != nil {
			t.Fatal(err)
		}
		tagsnum++
		t.Logf("put alias '%s' to '%s'", newname, oldname)
	}
	var delalias = func(name string) {
		if ok := pack.DelAlias(name); !ok {
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
	if err = pack.Begin(fwpk); err != nil {
		t.Fatal(err)
	}
	// put content
	putfile("bounty.jpg")
	putfile("img1/claustral.jpg")
	putfile("img1/qarataslar.jpg")
	putfile("img2/marble.jpg")
	putfile("img2/uzunji.jpg")
	putalias("img1/claustral.jpg", "basaltbay.jpg")
	for name, data := range memdata {
		putdata(name, data)
	}
	putalias("img1/claustral.jpg", "jasper.jpg")
	delalias("basaltbay.jpg")
	// finalize
	if err = pack.Complete(fwpk); err != nil {
		t.Fatal(err)
	}

	// make package file check up
	CheckPackage(t, fwpk, tagsnum)

	// check alias existence
	if _, ok := pack.Tags["jasper.jpg"]; !ok {
		t.Fatal("'jasper.jpg' alias not found")
	}
	if _, ok := pack.Tags["basaltbay.jpg"]; ok {
		t.Fatal("'basaltbay.jpg' alias not deleted")
	}
}

// Test to make package in two steps on single package open:
// creates package file, make package, do some job,
// then append new files to existing package.
func TestAppendContinues(t *testing.T) {
	var err error
	var pack wpk.Writer
	var fwpk *os.File
	var tagsnum = 0

	defer os.Remove(testpack)

	// helper functions
	var putfile = func(name string) {
		var file *os.File
		if file, err = os.Open(mediadir + name); err != nil {
			t.Fatal(err)
		}
		defer file.Close()

		var tags wpk.Tagset
		if tags, err = pack.PackFile(fwpk, file, name); err != nil {
			t.Fatal(err)
		}

		tagsnum++
		t.Logf("put file #%d '%s', %d bytes", pack.RecNumber, name, tags.Size())
	}

	// open temporary file for read/write
	if fwpk, err = os.OpenFile(testpack, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644); err != nil {
		t.Fatal(err)
	}
	defer fwpk.Close()

	// starts new package
	if err = pack.Begin(fwpk); err != nil {
		t.Fatal(err)
	}
	// put content
	putfile("bounty.jpg")
	putfile("img1/claustral.jpg")
	putfile("img1/qarataslar.jpg")
	// finalize
	if err = pack.Complete(fwpk); err != nil {
		t.Fatal(err)
	}

	// make package file check up
	CheckPackage(t, fwpk, tagsnum)

	//
	// here can be any job using package
	//

	// starts append to existing package
	if err = pack.Append(fwpk); err != nil {
		t.Fatal(err)
	}
	// put content
	putfile("img2/marble.jpg")
	putfile("img2/uzunji.jpg")
	// finalize
	if err = pack.Complete(fwpk); err != nil {
		t.Fatal(err)
	}

	// make package file check up
	CheckPackage(t, fwpk, tagsnum)
}

// Test to make package in two steps on twice package opens:
// creates package file, make package, close file,
// then open package file again and append new files.
func TestAppendDiscrete(t *testing.T) {
	var err error
	var pack wpk.Writer
	var fwpk *os.File
	var tagsnum = 0

	defer os.Remove(testpack)

	// helper functions
	var putfile = func(name string) {
		var file *os.File
		if file, err = os.Open(mediadir + name); err != nil {
			t.Fatal(err)
		}
		defer file.Close()

		var tags wpk.Tagset
		if tags, err = pack.PackFile(fwpk, file, name); err != nil {
			t.Fatal(err)
		}

		tagsnum++
		t.Logf("put file #%d '%s', %d bytes", pack.RecNumber, name, tags.Size())
	}

	t.Run("step1", func(t *testing.T) {
		// open temporary file for read/write
		if fwpk, err = os.OpenFile(testpack, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644); err != nil {
			t.Fatal(err)
		}
		defer fwpk.Close()

		// starts new package
		if err = pack.Begin(fwpk); err != nil {
			t.Fatal(err)
		}
		// put content
		putfile("bounty.jpg")
		putfile("img1/claustral.jpg")
		putfile("img1/qarataslar.jpg")
		// finalize
		if err = pack.Complete(fwpk); err != nil {
			t.Fatal(err)
		}

		// make package file check up
		CheckPackage(t, fwpk, tagsnum)
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
		// pack value already contains data from previous step
		// and this call can be skipped,
		// but we want to test here read functionality
		if err = pack.Read(fwpk); err != nil {
			t.Fatal(err)
		}

		// starts append to existing package
		if err = pack.Append(fwpk); err != nil {
			t.Fatal(err)
		}
		// put content
		putfile("img2/marble.jpg")
		putfile("img2/uzunji.jpg")
		// finalize
		if err = pack.Complete(fwpk); err != nil {
			t.Fatal(err)
		}

		// make package file check up
		CheckPackage(t, fwpk, tagsnum)
	})
}

// The End.
