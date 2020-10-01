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

func CheckPackage(t *testing.T, file *os.File) {
	var err error
	var pack Package

	if err = pack.Read(file); err != nil {
		t.Fatal(err)
	}

	for _, tags := range pack.Tags {
		var path, _ = tags.String(TID_path)
		var offset, size = tags.Record()
		if link, ok := tags.String(TID_link); ok {
			t.Logf("found alias '%s' to '%s'", path, link)
			continue // skip aliases
		}

		var orig []byte
		if orig, err = ioutil.ReadFile(mediadir + path); err != nil {
			t.Fatal(err)
		}

		if tags.Size() != int64(len(orig)) {
			t.Errorf("size of file '%s' (%d) in package is defer from original (%d)",
				path, tags.Size(), len(orig))
		}

		var extr = make([]byte, size, size)
		var n int
		if n, err = file.ReadAt(extr, offset); err != nil {
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

		t.Logf("#%d checkup of '%s' is ok", tags.FID(), path)
	}

	//t.Fail()
}

func TestPackDir(t *testing.T) {
	var err error
	var pack Package
	var file *os.File

	if file, err = os.OpenFile(testpack, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644); err != nil {
		t.Fatal(err)
	}
	defer file.Close()

	// starts new package
	if err = pack.Begin(file); err != nil {
		t.Fatal(err)
	}
	// put media directory to file
	if err = pack.PackDir(file, mediadir, "", func(fi os.FileInfo, fname, fpath string) {
		t.Logf("#%-2d %6d bytes   %s", pack.RecNumber, fi.Size(), fname)
	}); err != nil {
		t.Fatal(err)
	}
	// finalize
	if err = pack.Complete(file); err != nil {
		t.Fatal(err)
	}

	CheckPackage(t, file)
}

func TestPutFiles(t *testing.T) {
	var err error
	var pack Package
	var file *os.File

	var putfile = func(name string) {
		var tags Tagset
		if tags, err = pack.PackFile(file, name, mediadir+name); err != nil {
			t.Fatal(err)
		}
		t.Logf("#%-2d %6d bytes   %s", pack.RecNumber, tags.Size(), name)
	}
	var makealias = func(oldname, newname string) {
		if err = pack.PutAlias(oldname, newname); err != nil {
			t.Fatal(err)
		}
		t.Logf("maked alias '%s' to '%s'", newname, oldname)
	}

	if file, err = os.OpenFile(testpack, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644); err != nil {
		t.Fatal(err)
	}
	defer file.Close()

	// starts new package
	if err = pack.Begin(file); err != nil {
		t.Fatal(err)
	}
	// put content
	putfile("bounty.jpg")
	putfile("img1/claustral.jpg")
	putfile("img1/qarataslar.jpg")
	putfile("img2/marble.jpg")
	putfile("img2/uzunji.jpg")
	makealias("img1/claustral.jpg", "jasper.jpg")
	// finalize
	if err = pack.Complete(file); err != nil {
		t.Fatal(err)
	}

	CheckPackage(t, file)
}

// The End.
