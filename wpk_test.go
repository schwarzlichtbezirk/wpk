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
		if _, ok := tags.String(TID_link); ok {
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
	pack.Complete(file)

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
		var key1 = ToKey(oldname)
		var key2 = ToKey(newname)
		var tags1, ok = pack.Tags[key1]
		if !ok {
			t.Fatalf("file '%s' is not found in package", oldname)
		}
		if _, ok = pack.Tags[key2]; ok {
			t.Fatalf("file '%s' already present in package", newname)
		}
		var tags2 = Tagset{}
		for k, v := range tags1 {
			tags2[k] = v
		}
		tags2[TID_path] = TagString(newname)
		if _, ok := tags2.String(TID_link); !ok {
			tags2[TID_link] = TagString(oldname)
		}
		pack.Tags[key2] = tags2
		pack.TagNumber = FID(len(pack.Tags))
		t.Logf("alias '%s' to '%s'", oldname, newname)
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
	pack.Complete(file)

	CheckPackage(t, file)

	t.Fail()
}

// The End.
