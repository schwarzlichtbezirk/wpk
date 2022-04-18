package wpk_test

import (
	"log"
	"os"

	"github.com/schwarzlichtbezirk/wpk"
)

func ExamplePackage_Read() {
	var err error

	// Open package file for reading
	var f *os.File
	if f, err = os.Open("example.wpk"); err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	// Read package files tags table
	var pack wpk.Package
	if err = pack.Read(f); err != nil {
		log.Fatal(err)
	}
	// How many records at package
	var n = 0
	pack.Enum(func(fkey string, ts *wpk.Tagset_t) bool {
		if n < 5 {
			// Print not more than 5 file names from package
			log.Println(fkey)
		}
		n++
		return true
	})
	log.Printf("files: %d, datasize: %d\n", n, pack.DataSize())
}

func ExamplePackage_Glob() {
	var err error

	// Open package file for reading
	var f *os.File
	if f, err = os.Open("example.wpk"); err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	// Read package files tags table
	var pack wpk.Package
	if err = pack.Read(f); err != nil {
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
