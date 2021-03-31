package main

import (
	"bytes"
	"flag"
	"io"
	"log"
	"mime"
	"net/http"
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
	ShowLog bool
)

func parseargs() {
	flag.StringVar(&srcpath, "src", "", "full path to folder with source files to be packaged, or list of folders divided by ';'")
	flag.StringVar(&DstFile, "dst", "", "full path to output package file")
	flag.BoolVar(&PutMIME, "mime", false, "put content MIME type defined by file extension")
	flag.BoolVar(&ShowLog, "sl", true, "show process log for each extracting file")
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
		path = wpk.Envfmt(path)
		if !strings.HasSuffix(path, "/") {
			path += "/"
		}
		if ok, _ := wpk.PathExists(path); !ok {
			log.Printf("source path #%d '%s' does not exist", i+1, path)
			ec++
			continue
		}
		SrcList = append(SrcList, path)
	}

	DstFile = wpk.Envfmt(DstFile)
	if DstFile == "" {
		log.Println("destination file does not specified")
		ec++
	} else if ok, _ := wpk.PathExists(filepath.Dir(DstFile)); !ok {
		log.Println("destination path does not exist")
		ec++
	}

	return ec
}

func writepackage() (err error) {
	var pack wpk.Writer
	var fwpk *os.File

	// open package file to write
	if fwpk, err = os.OpenFile(DstFile, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644); err != nil {
		return
	}
	defer fwpk.Close()
	log.Printf("destination file: %s", DstFile)

	// starts new package
	if err = pack.Begin(fwpk); err != nil {
		return
	}

	// write all source folders
	for i, path := range SrcList {
		log.Printf("source folder #%d: %s", i+1, path)
		var sum int64
		if err = pack.PackDir(fwpk, path, "", func(fi os.FileInfo, fname, fpath string) bool {
			var size = fi.Size()
			if ShowLog && !fi.IsDir() {
				log.Printf("#%-4d %7d bytes   %s", pack.RecNumber()+1, size, fname)
			}
			sum += size
			return true
		}); err != nil {
			return
		}
		log.Printf("packed: %d files on %d bytes", pack.RecNumber(), sum)
	}

	// adjust tags
	if PutMIME {
		log.Printf("put mime type tags")
		const sniffLen = 512
		for fname, tags := range pack.Tags {
			var ctype = mime.TypeByExtension(filepath.Ext(fname))
			if ctype == "" {
				var offset, size = tags.Offset(), tags.Size()
				// rewind to file start
				if _, err = fwpk.Seek(offset, io.SeekStart); err != nil {
					return
				}
				// read a chunk to decide between utf-8 text and binary
				var buf [sniffLen]byte
				var n int64
				if n, err = io.CopyN(bytes.NewBuffer(buf[:]), io.LimitReader(fwpk, size), sniffLen); err != nil && err != io.EOF {
					return
				}
				ctype = http.DetectContentType(buf[:n])
			}
			tags[wpk.TIDmime] = wpk.TagString(ctype)
		}
	}

	// finalize
	log.Printf("write tags table")
	if err = pack.Finalize(fwpk); err != nil {
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
