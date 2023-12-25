package wpk_test

import (
	"bytes"
	"io/fs"
	"os"
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
	var pkg = wpk.NewPackage()

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
	if fwpk, err = os.OpenFile(wpkname, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644); err != nil {
		t.Fatal(err)
	}
	defer fwpk.Close()

	// starts new package
	if err = pkg.Begin(fwpk, nil); err != nil {
		t.Fatal(err)
	}
	// put content
	for _, fname := range list {
		putfile(fname)
	}
	// finalize
	if err = pkg.Sync(fwpk, nil); err != nil {
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
		"img1/Qarataşlar.jpg",
		"img2/Uzuncı.jpg",
	})

	defer os.Remove(testpack1)
	defer os.Remove(testpack2)

	var err error
	var pack1 = wpk.NewPackage()
	if err = pack1.OpenFile(testpack1); err != nil {
		t.Fatal(err)
	}
	if pack1.Tagger, err = mmap.MakeTagger(testpack1); err != nil {
		t.Fatal(err)
	}
	var pack2 = wpk.NewPackage()
	if err = pack2.OpenFile(testpack2); err != nil {
		t.Fatal(err)
	}
	if pack2.Tagger, err = bulk.MakeTagger(testpack2); err != nil {
		t.Fatal(err)
	}

	var u wpk.Union
	u.List = []*wpk.Package{pack1, pack2}
	defer u.Close()

	var (
		folder fs.File
		list   []string
		check  = func(testname string, m map[string]wpk.Void) {
			if len(list) != len(m) {
				t.Fatalf("%s test: expected %d filenames in union, got %d", testname, len(m), len(list))
			}
			for _, fpath := range list {
				if _, ok := m[fpath]; !ok {
					t.Fatalf("%s test: got filename '%s' from union that does not present at preset", testname, fpath)
				}
			}
		}
		checkfs = func(ufs fs.FS, fpath string, m map[string]wpk.Void) {
			if folder, err = ufs.Open(fpath); err != nil {
				t.Fatal(err)
			}
			if df, ok := folder.(fs.ReadDirFile); ok {
				var delist, _ = df.ReadDir(-1)
				list = make([]string, len(delist))
				for i, de := range delist {
					list[i] = wpk.JoinPath(fpath, de.Name())
				}
			} else {
				t.Fatalf("cannot cast '%s' directory property to fs.ReadDirFile", fpath)
			}
			check(fpath+" folder", m)
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
		"img1/Qarataşlar.jpg": {},
		"img2/Uzuncı.jpg":     {},
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
		"img2/Uzuncı.jpg": {},
	})

	if list, err = u.Glob("*/?ar*.jpg"); err != nil {
		t.Fatal(err)
	}
	check("*/?ar*.jpg", map[string]wpk.Void{
		"img1/Qarataşlar.jpg": {},
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
		"img1/Qarataşlar.jpg": {},
	})

	checkfs(&u, "img2", map[string]wpk.Void{
		"img2/marble.jpg": {},
		"img2/Uzuncı.jpg": {},
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
		"Qarataşlar.jpg": {},
	})

	var u2 fs.FS
	if u2, err = u.Sub("img2"); err != nil {
		t.Fatal(err)
	}
	checkfs(u2, ".", map[string]wpk.Void{
		"marble.jpg": {},
		"Uzuncı.jpg": {},
	})

	//
	// ReadFile test
	//

	var imgfpath = wpk.JoinPath(mediadir, "img1/Qarataşlar.jpg")
	var imgb, pkgb []byte
	if imgb, err = os.ReadFile(imgfpath); err != nil {
		t.Fatal(err)
	}
	if pkgb, err = u.ReadFile("img1/Qarataşlar.jpg"); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(imgb, pkgb) {
		t.Fatal("content of 'img1/Qarataşlar.jpg' is not equal to original")
	}

	//
	// File open test
	//

	var imgf fs.File
	if imgf, err = u1.Open("Qarataşlar.jpg"); err != nil {
		t.Fatal(err)
	}
	var imgfi fs.FileInfo
	if imgfi, err = imgf.Stat(); err != nil {
		t.Fatal(err)
	}
	pkgb = make([]byte, imgfi.Size())
	if _, err = imgf.Read(pkgb); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(imgb, pkgb) {
		t.Fatal("content of 'Qarataşlar.jpg' is not equal to original")
	}
}

// The End.
