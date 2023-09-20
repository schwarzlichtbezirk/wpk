package wpk_test

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/schwarzlichtbezirk/wpk"
)

func ExampleFTT_GetInfo() {
	var err error

	// Open package files tags table
	var pkg = wpk.NewPackage()
	if err = pkg.OpenFile("example.wpk"); err != nil {
		log.Fatal(err)
	}

	// Format package information
	var items = []string{
		fmt.Sprintf("records: %d", pkg.TagsetNum()),
		fmt.Sprintf("datasize: %d", pkg.DataSize()),
	}
	if str, ok := pkg.GetInfo().TagStr(wpk.TIDlabel); ok {
		items = append(items, fmt.Sprintf("label: %s", str))
	}
	if str, ok := pkg.GetInfo().TagStr(wpk.TIDlink); ok {
		items = append(items, fmt.Sprintf("link: %s", str))
	}
	log.Println(strings.Join(items, ", "))
}

func ExampleFTT_SetInfo() {
	var err error
	var f *os.File
	var pkg = wpk.NewPackage()

	const (
		label  = "empty-package"
		link   = "github.com/schwarzlichtbezirk/wpk"
		author = "schwarzlichtbezirk"
	)

	// open temporary file for read/write
	if f, err = os.OpenFile("example.wpk", os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644); err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	// starts new package
	if err = pkg.Begin(f, nil); err != nil {
		log.Fatal(err)
	}

	// put package info somewhere at the code before finalize
	pkg.SetInfo(wpk.TagsetRaw{}.
		Put(wpk.TIDlabel, wpk.StrTag(label)).
		Put(wpk.TIDlink, wpk.StrTag(link)).
		Put(wpk.TIDauthor, wpk.StrTag(author)))

	// finalize at the end
	if err = pkg.Sync(f, nil); err != nil {
		log.Fatal(err)
	}
}

func ExamplePackage_Enum() {
	var err error

	// Open package files tags table
	var pkg = wpk.NewPackage()
	if err = pkg.OpenFile("example.wpk"); err != nil {
		log.Fatal(err)
	}

	// How many unique records in package
	var m = map[wpk.Uint]wpk.Void{}
	pkg.Enum(func(fkey string, ts wpk.TagsetRaw) bool {
		if offset, ok := ts.TagUint(wpk.TIDoffset); ok {
			m[offset] = wpk.Void{} // count unique offsets
		}
		return true
	})
	log.Printf("total %d unique files and %d aliases in package", len(m), pkg.TagsetNum()-len(m))
}

func ExamplePackage_Glob() {
	var err error

	// Open package files tags table
	var pkg = wpk.NewPackage()
	if err = pkg.OpenFile("example.wpk"); err != nil {
		log.Fatal(err)
	}

	// Get all JPEG-files in subdirectories
	var res []string
	if res, err = pkg.Glob("*/*.jpg"); err != nil {
		log.Fatal(err)
	}
	// and print them
	for _, fname := range res {
		log.Println(fname)
	}
}

func ExampleGetPackageInfo() {
	// list of packages to get info
	var list = []string{
		"example1.wpk", "example2.wpk", "example3.wpk",
	}
	for _, fname := range list {
		func() {
			var err error

			// open package file
			var f *os.File
			if f, err = os.Open(fname); err != nil {
				log.Printf("can not open package %s, %s", fname, err.Error())
				return
			}
			defer f.Close()

			// get quick package info
			var hdr wpk.Header
			var info wpk.TagsetRaw
			if hdr, info, err = wpk.GetPackageInfo(f); err != nil {
				log.Printf("can not get package info %s, %s", fname, err.Error())
				return
			}

			// print package info
			var label, _ = info.TagStr(wpk.TIDlabel)
			log.Printf("package %s info: %d records, %d data size, %d info notes, label '%s'",
				fname, hdr.Count(), hdr.DataSize(), info.Num(), label)
		}()
	}
}

// The End.
