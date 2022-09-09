package wpk_test

import (
	"io/fs"
	"os"
	"path"
	"testing"

	"github.com/schwarzlichtbezirk/wpk"
	"github.com/schwarzlichtbezirk/wpk/bulk"
	"github.com/schwarzlichtbezirk/wpk/mmap"
)

var testpack1 = wpk.TempPath("testpack1.wpk")
var testpack2 = wpk.TempPath("testpack2.wpk")

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

		var ts *wpk.TagsetRaw
		if ts, err = pack.PackFile(fwpk, file, name); err != nil {
			t.Fatal(err)
		}

		tagsnum++
		fidcount++
		ts.Put(wpk.TIDfid, wpk.UintTag(fidcount))
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
// that union have valid files set. Glob test.
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

	var (
		folder fs.File
		list   []string
		check  = func(testname string, m map[string]wpk.Void) {
			if len(list) != len(m) {
				t.Fatalf("%s test: expected %d filenames in union, got %d", testname, len(m), len(list))
			}
			for _, fname := range list {
				if _, ok := m[fname]; !ok {
					t.Fatalf("%s test: got filename '%s' from union that does not present at preset", testname, fname)
				}
			}
		}
		checkfs = func(ufs fs.FS, fname string, m map[string]wpk.Void) {
			if folder, err = ufs.Open(fname); err != nil {
				t.Fatal(err)
			}
			if df, ok := folder.(fs.ReadDirFile); ok {
				var delist, _ = df.ReadDir(-1)
				list = nil
				for _, de := range delist {
					list = append(list, path.Join(fname, de.Name()))
				}
			} else {
				t.Fatalf("cannot cast '%s' directory property to fs.ReadDirFile", fname)
			}
			check(fname+" folder", m)
		}
	)

	//
	// AllKeys test
	//

	list = u.AllKeys()
	check("all keys", map[string]wpk.Void{
		"bounty.jpg":          {},
		"img1/claustral.jpg":  {},
		"img2/marble.jpg":     {},
		"img1/qarataslar.jpg": {},
		"img2/uzunji.jpg":     {},
	})

	//
	// Glob test
	//

	if list, err = u.Glob("*"); err != nil {
		t.Fatal(err)
	}
	check("root files", map[string]wpk.Void{
		"bounty.jpg": {},
	})

	if list, err = u.Glob("img2/*"); err != nil {
		t.Fatal(err)
	}
	check("img2/*", map[string]wpk.Void{
		"img2/marble.jpg": {},
		"img2/uzunji.jpg": {},
	})

	if list, err = u.Glob("*/?ar*.jpg"); err != nil {
		t.Fatal(err)
	}
	check("*/?ar*.jpg", map[string]wpk.Void{
		"img1/qarataslar.jpg": {},
		"img2/marble.jpg":     {},
	})

	//
	// File system test
	//

	checkfs(&u, ".", map[string]wpk.Void{
		"bounty.jpg": {},
		"img1":       {},
		"img2":       {},
	})

	checkfs(&u, "img1", map[string]wpk.Void{
		"img1/claustral.jpg":  {},
		"img1/qarataslar.jpg": {},
	})

	checkfs(&u, "img2", map[string]wpk.Void{
		"img2/marble.jpg": {},
		"img2/uzunji.jpg": {},
	})

	//
	// Subdirectory test
	//

	var u1 fs.FS
	if u1, err = u.Sub("img1"); err != nil {
		t.Fatal(err)
	}
	checkfs(u1, ".", map[string]wpk.Void{
		"claustral.jpg":  {},
		"qarataslar.jpg": {},
	})

	var u2 fs.FS
	if u2, err = u.Sub("img2"); err != nil {
		t.Fatal(err)
	}
	checkfs(u2, ".", map[string]wpk.Void{
		"marble.jpg": {},
		"uzunji.jpg": {},
	})
}

// The End.
