package main

import (
	"encoding/binary"
	"flag"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/schwarzlichtbezirk/wpk"
)

// command line settings
var (
	srcpath string
	SrcList []string
	DstFile string
	PutMIME bool
)

func pathexists(path string) (bool, error) {
	var err error
	if _, err = os.Stat(path); err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return true, err
}

func parseargs() {
	flag.StringVar(&srcpath, "src", "", "full path to folder with source files to be packaged, or list of folders divided by ';'")
	flag.StringVar(&DstFile, "dst", "", "full path to output package file")
	flag.BoolVar(&PutMIME, "mime", false, "put content MIME type defined by file extension")
	flag.Parse()
}

func checkargs() int {
	var ec = 0 // error counter

	srcpath = filepath.ToSlash(strings.Trim(srcpath, ";"))
	if srcpath == "" {
		log.Println("source path does not specified")
		ec++
	}
	for i, path := range strings.Split(srcpath, ";") {
		if path == "" {
			continue
		}
		if !strings.HasSuffix(path, "/") {
			path += "/"
		}
		if ok, _ := pathexists(path); !ok {
			log.Printf("source path #%d '%s' does not exist", i+1, path)
			ec++
			continue
		}
		SrcList = append(SrcList, path)
	}

	DstFile = filepath.ToSlash(DstFile)
	if DstFile == "" {
		log.Println("destination file does not specified")
		ec++
	} else if ok, _ := pathexists(filepath.Dir(DstFile)); !ok {
		log.Println("destination path does not exist")
		ec++
	}

	return ec
}

func writepackage() (err error) {
	var dst *os.File
	if dst, err = os.OpenFile(DstFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0755); err != nil {
		return
	}
	defer dst.Close()

	log.Printf("destination file: %s", DstFile)

	var pack wpk.Package
	pack.Init()

	// write prebuild header
	if err = binary.Write(dst, binary.LittleEndian, &pack.PackHdr); err != nil {
		return
	}

	// write all source folders
	for i, path := range SrcList {
		log.Printf("source folder #%d: %s", i+1, path)
		if err = pack.PackDir(dst, path, "", func(fi os.FileInfo, fname, fpath string) {
			log.Printf("#%-4d %7d bytes   %s", len(pack.FAT), fi.Size(), fname)
		}); err != nil {
			return
		}
	}

	// adjust tags
	if PutMIME {
		log.Printf("put mime type tags")
		for fname, tags := range pack.Tags {
			if ct, ok := mimeext[filepath.Ext(fname)]; ok {
				tags.SetString(wpk.AID_mime, ct)
			}
		}
	}

	// write records table
	log.Printf("write file allocation table")
	if pack.RecOffset, err = dst.Seek(0, os.SEEK_END); err != nil {
		return
	}
	pack.RecNumber = int64(len(pack.FAT))
	if err = binary.Write(dst, binary.LittleEndian, &pack.FAT); err != nil {
		return
	}

	// write files tags table
	log.Printf("write tags table")
	if pack.TagOffset, err = dst.Seek(0, os.SEEK_CUR); err != nil {
		return
	}
	pack.TagNumber = int64(len(pack.Tags))
	for _, tags := range pack.Tags {
		if err = tags.Write(dst); err != nil {
			return
		}
	}

	// rewrite true header
	if _, err = dst.Seek(0, os.SEEK_SET); err != nil {
		return
	}
	copy(pack.Signature[:], wpk.Signature)
	if err = binary.Write(dst, binary.LittleEndian, &pack.PackHdr); err != nil {
		return
	}

	return
}

func main() {
	parseargs()
	if checkargs() > 0 {
		return
	}

	log.Println("starts")
	if err := writepackage(); err != nil {
		log.Println(err.Error())
		return
	}
	log.Println("done.")
}

// The End.
