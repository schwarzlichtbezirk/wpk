package wpk_test

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/schwarzlichtbezirk/wpk"
)

func ExamplePackage_OpenFTT() {
	var err error

	// Open package file for reading
	var f *os.File
	if f, err = os.Open("example.wpk"); err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	// Open package files tags table
	var pack = wpk.NewPackage[TID_t, TSize_t](tssize)
	if err = pack.OpenFTT(f); err != nil {
		log.Fatal(err)
	}

	// How many records in package
	var m = map[wpk.FOffset_t]struct{}{}
	var n = 0
	pack.Enum(func(fkey string, ts *wpk.Tagset_t[TID_t, TSize_t]) bool {
		if n < 5 { // print not more than 5 file names from package
			log.Println(fkey)
		}
		if offset, ok := wpk.UintTagset[TID_t, TSize_t, wpk.FOffset_t](ts, wpk.TIDoffset); ok {
			m[offset] = struct{}{}
		}
		n++
		return true
	})

	// Format package information
	var items []string
	items = append(items, fmt.Sprintf("records: %d", len(m)))
	items = append(items, fmt.Sprintf("aliases: %d", n))
	if ts, ok := pack.Tagset(""); ok { // get package info if it present
		if size, ok := wpk.UintTagset[TID_t, TSize_t, wpk.FSize_t](ts, wpk.TIDsize); ok {
			items = append(items, fmt.Sprintf("datasize: %d", size))
		}
		if str, ok := ts.String(wpk.TIDlabel); ok {
			items = append(items, fmt.Sprintf("label: %s", str))
		}
	}
	log.Println(strings.Join(items, ", "))
}

func ExamplePackage_Glob() {
	var err error

	// Open package file for reading
	var f *os.File
	if f, err = os.Open("example.wpk"); err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	// Open package files tags table
	var pack = wpk.NewPackage[TID_t, TSize_t](tssize)
	if err = pack.OpenFTT(f); err != nil {
		log.Fatal(err)
	}

	// Get all JPEG-files in subdirectories
	var res []string
	if res, err = pack.Glob("*/*.jpg"); err != nil {
		log.Fatal(err)
	}
	// and print them
	for _, fname := range res {
		log.Println(fname)
	}
}

// The End.
