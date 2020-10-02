package wpk

import (
	"bytes"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"testing"
)

var efre = regexp.MustCompile(`\$\(\w+\)`)

func envfmt(p string) string {
	return filepath.ToSlash(efre.ReplaceAllStringFunc(p, func(name string) string {
		return os.Getenv(name[2 : len(name)-1]) // strip $(...) and replace by env value
	}))
}

var mediadir = envfmt("$(GOPATH)/src/github.com/schwarzlichtbezirk/wpk/test/media/")
var testpack = filepath.Join(os.TempDir(), "testpack.wpk")

// Test package content on nested and external files equivalent.
func CheckPackage(t *testing.T, fwpk *os.File, tagsnum int) {
	var err error
	var pack Package

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

// Test PackDir function work.
func TestPackDir(t *testing.T) {
	var err error
	var pack Package
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
	if err = pack.PackDir(fwpk, mediadir, "", func(fi os.FileInfo, fname, fpath string) {
		tagsnum++
		t.Logf("put file #%d '%s', %d bytes", pack.RecNumber, fname, fi.Size())
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
	var pack Package
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

		var tags Tagset
		if tags, err = pack.PackFile(fwpk, file, name); err != nil {
			t.Fatal(err)
		}

		tagsnum++
		t.Logf("put file #%d '%s', %d bytes", pack.RecNumber, name, tags.Size())
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
	var pack Package
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

		var tags Tagset
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
	var pack Package
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

		var tags Tagset
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
