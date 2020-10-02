package main

import (
	"bytes"
	"encoding/binary"
	"flag"
	"io"
	"log"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	. "github.com/schwarzlichtbezirk/wpk"
)

// command line settings
var (
	srcpath string
	SrcList []string
	DstFile string
	PutMIME bool
)

var efre = regexp.MustCompile(`\$\(\w+\)`)

func envfmt(p string) string {
	return filepath.ToSlash(efre.ReplaceAllStringFunc(p, func(name string) string {
		return os.Getenv(name[2 : len(name)-1]) // strip $(...) and replace by env value
	}))
}

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
		path = envfmt(path)
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

	DstFile = envfmt(DstFile)
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
	if dst, err = os.OpenFile(DstFile, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0755); err != nil {
		return
	}
	defer dst.Close()

	var pack Package

	// reset header
	copy(pack.Signature[:], Prebuild)
	pack.TagOffset = PackHdrSize
	pack.RecNumber = 0
	pack.TagNumber = 0
	// setup empty tags table
	pack.Tags = map[string]Tagset{}
	// write prebuild header
	if err = binary.Write(dst, binary.LittleEndian, &pack.PackHdr); err != nil {
		return
	}

	log.Printf("destination file: %s", DstFile)

	// write all source folders
	for i, path := range SrcList {
		log.Printf("source folder #%d: %s", i+1, path)
		if err = pack.PackDir(dst, path, "", func(fi os.FileInfo, fname, fpath string) {
			log.Printf("#%-4d %7d bytes   %s", pack.RecNumber, fi.Size(), fname)
		}); err != nil {
			return
		}
	}

	// adjust tags
	if PutMIME {
		log.Printf("put mime type tags")
		const sniffLen = 512
		for fname, tags := range pack.Tags {
			var ctype = mime.TypeByExtension(filepath.Ext(fname))
			if ctype == "" {
				var offset, _ = tags.Uint64(TID_offset)
				var size, _ = tags.Uint64(TID_size)
				// rewind to file start
				if _, err = dst.Seek(int64(offset), io.SeekStart); err != nil {
					return
				}
				// read a chunk to decide between utf-8 text and binary
				var buf [sniffLen]byte
				var n int64
				if n, err = io.Copy(bytes.NewBuffer(buf[:]), io.LimitReader(dst, int64(size))); err != nil {
					return
				}
				ctype = http.DetectContentType(buf[:n])
			}
			tags[TID_mime] = TagString(ctype)
		}
	}

	// write files tags table
	log.Printf("write tags table")
	var tagoffset int64
	if tagoffset, err = dst.Seek(0, io.SeekEnd); err != nil {
		return
	}
	pack.TagOffset = OFFSET(tagoffset)
	pack.TagNumber = FID(len(pack.Tags))
	for _, tags := range pack.Tags {
		if _, err = tags.WriteTo(dst); err != nil {
			return
		}
	}

	// rewrite true header
	if _, err = dst.Seek(0, io.SeekStart); err != nil {
		return
	}
	copy(pack.Signature[:], Signature)
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
