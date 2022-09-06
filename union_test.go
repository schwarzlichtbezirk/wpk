package wpk_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/schwarzlichtbezirk/wpk"
	"github.com/schwarzlichtbezirk/wpk/bulk"
	"github.com/schwarzlichtbezirk/wpk/mmap"
)

var testpack1 = filepath.Join(os.TempDir(), "testpack1.wpk")
var testpack2 = filepath.Join(os.TempDir(), "testpack2.wpk")

// PackFiles makes package with given list of files.
func PackFiles(t *testing.T, wpkname string, list []string) {
	var err error
	var fwpk *os.File
	var tagsnum = 0
	var fidcount uint
	var pack = wpk.NewPackage(pts)

	// helper functions
	var putfile = func(name string) {
		var file *os.File
		if file, err = os.Open(mediadir + name); err != nil {
			t.Fatal(err)
		}
		defer file.Close()

		var ts *wpk.Tagset_t
		if ts, err = pack.PackFile(fwpk, file, name); err != nil {
			t.Fatal(err)
		}

		tagsnum++
		fidcount++
		ts.Put(wpk.TIDfid, wpk.TagUint(fidcount))
		var size = ts.Size()
		t.Logf("put file #%d '%s', %d bytes", fidcount, name, size)
	}

	// open temporary file for read/write
	if fwpk, err = os.OpenFile(wpkname, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644); err != nil {
		t.Fatal(err)
	}
	defer fwpk.Close()

	// starts new package
	if err = pack.Begin(fwpk); err != nil {
		t.Fatal(err)
	}
	// put content
	for _, fname := range list {
		putfile(fname)
	}
	// finalize
	if err = pack.Sync(fwpk, nil); err != nil {
		t.Fatal(err)
	}
}

// Test to make union of two different packages, and checks
// that union have valid files set.
func TestUnion(t *testing.T) {
	PackFiles(t, testpack1, []string{
		"bounty.jpg",
		"img1/claustral.jpg",
		"img2/marble.jpg",
	})
	PackFiles(t, testpack2, []string{
		"bounty.jpg",
		"img1/qarataslar.jpg",
		"img2/uzunji.jpg",
	})

	defer os.Remove(testpack1)
	defer os.Remove(testpack2)

	var err error
	var pack1, pack2 wpk.Packager
	if pack1, err = mmap.OpenPackage(testpack1); err != nil {
		t.Fatal(err)
	}
	if pack2, err = bulk.OpenPackage(testpack2); err != nil {
		t.Fatal(err)
	}

	var u wpk.Union
	u.List = []wpk.Packager{pack1, pack2}
	defer u.Close()

	var m = map[string]wpk.Void{
		"bounty.jpg":          {},
		"img1/claustral.jpg":  {},
		"img2/marble.jpg":     {},
		"img1/qarataslar.jpg": {},
		"img2/uzunji.jpg":     {},
	}

	var list = u.AllKeys()
	t.Log(list)
	if len(list) != len(m) {
		t.Fatalf("expected %d filenames in union, got %d", len(m), len(list))
	}
	for _, fname := range list {
		if _, ok := m[fname]; !ok {
			t.Fatalf("got filename '%s' from union that does not present at preset", fname)
		}
	}
}

// The End.
