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

type (
	TID_t   = uint16
	TSize_t = uint16
)

const (
	foffset = 8
	fsize   = 8
	fidsz   = 4
	tssize  = 2
)

// command line settings
var (
	srcpath string
	SrcList []string
	DstFile string
	PutMIME bool
	ShowLog bool
	Split   bool
)

func parseargs() {
	flag.StringVar(&srcpath, "src", "", "full path to folder with source files to be packaged, or list of folders divided by ';'")
	flag.StringVar(&DstFile, "dst", "", "full path to output package file")
	flag.BoolVar(&PutMIME, "mime", false, "put content MIME type defined by file extension")
	flag.BoolVar(&ShowLog, "sl", true, "show process log for each extracting file")
	flag.BoolVar(&Split, "split", false, "write package to splitted files")
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

var num, sum int64

func packdirclosure(r io.ReadSeeker, ts *wpk.Tagset_t[TID_t, TSize_t]) (err error) {
	var size = ts.Size()
	var fname, _ = ts.String(wpk.TIDpath)
	num++
	sum += size
	if ShowLog {
		log.Printf("#%-4d %7d bytes   %s", num, size, fname)
	}

	// adjust tags
	if PutMIME {
		const sniffLen = 512
		var ctype = mime.TypeByExtension(filepath.Ext(fname))
		if ctype == "" {
			// rewind to file start
			if _, err = r.Seek(0, io.SeekStart); err != nil {
				return err
			}
			// read a chunk to decide between utf-8 text and binary
			var buf [sniffLen]byte
			var n int64
			if n, err = io.CopyN(bytes.NewBuffer(buf[:]), r, sniffLen); err != nil && err != io.EOF {
				return err
			}
			ctype = http.DetectContentType(buf[:n])
		}
		if ctype != "" {
			ts.Put(wpk.TIDmime, wpk.TagString(ctype))
		}
	}
	return nil
}

func writepackage() (err error) {
	var fwpk, fwpd wpk.WriteSeekCloser
	var pkgfile, datfile = DstFile, DstFile
	var pack = wpk.NewPackage[TID_t, TSize_t](foffset, fsize, fidsz, tssize)
	if Split {
		pkgfile, datfile = wpk.MakeTagsPath(pkgfile), wpk.MakeDataPath(datfile)
	}

	// open package file to write
	if fwpk, err = os.OpenFile(pkgfile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644); err != nil {
		return
	}
	defer fwpk.Close()

	if Split {
		if fwpd, err = os.OpenFile(datfile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644); err != nil {
			return
		}
		defer fwpd.Close()

		log.Printf("destination tags part:  %s\n", pkgfile)
		log.Printf("destination files part: %s\n", datfile)
	} else {
		log.Printf("destination file: %s\n", pkgfile)
	}

	// starts new package
	if err = pack.Begin(fwpk); err != nil {
		return
	}

	// data writer
	var w = fwpk
	if fwpd != nil {
		w = fwpd
	}

	// write all source folders
	for i, fpath := range SrcList {
		log.Printf("source folder #%d: %s", i+1, fpath)
		num, sum = 0, 0
		if err = pack.PackDir(w, fpath, "", packdirclosure); err != nil {
			return
		}
		log.Printf("packed: %d files on %d bytes", num, sum)
	}

	// finalize
	log.Printf("write tags table")
	if err = pack.Sync(fwpk, fwpd); err != nil {
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
