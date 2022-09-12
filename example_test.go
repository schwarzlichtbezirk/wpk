package wpk_test

import (
	"fmt"
	"log"
	"strings"

	"github.com/schwarzlichtbezirk/wpk"
)

func ExampleFTT_Info() {
	var err error

	// Open package files tags table
	var pkg *wpk.Package
	if pkg, err = wpk.OpenPackage("example.wpk"); err != nil {
		log.Fatal(err)
	}

	// How many records in package
	var m = map[uint]wpk.Void{}
	var n = 0
	pkg.Enum(func(fkey string, ts *wpk.TagsetRaw) bool {
		if offset, ok := ts.TagUint(wpk.TIDoffset); ok {
			m[offset] = wpk.Void{}
		}
		n++
		return true
	})

	// Format package information
	var items []string
	items = append(items, fmt.Sprintf("files: %d", len(m)))
	items = append(items, fmt.Sprintf("aliases: %d", n))
	if ts, ok := pkg.Info(); ok { // get package info if it present
		if size, ok := ts.TagUint(wpk.TIDsize); ok {
			items = append(items, fmt.Sprintf("datasize: %d", size))
		}
		if str, ok := ts.TagStr(wpk.TIDlabel); ok {
			items = append(items, fmt.Sprintf("label: %s", str))
		}
	}
	log.Println(strings.Join(items, ", "))
}

func ExamplePackage_Enum() {
	var err error

	// Open package files tags table
	var pkg *wpk.Package
	if pkg, err = wpk.OpenPackage("example.wpk"); err != nil {
		log.Fatal(err)
	}

	// How many records in package
	var n = 0
	pkg.Enum(func(fkey string, ts *wpk.TagsetRaw) bool {
		if n < 5 { // print not more than 5 file names from package
			log.Println(fkey)
		}
		n++
		return true
	})
	log.Printf("total %d records in package files tags table", n)
}

func ExamplePackage_Glob() {
	var err error

	// Open package files tags table
	var pkg *wpk.Package
	if pkg, err = wpk.OpenPackage("example.wpk"); err != nil {
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

// The End.
