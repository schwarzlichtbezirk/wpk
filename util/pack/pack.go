package main

import (
	"bytes"
	"flag"
	"io"
	"io/fs"
	"log"
	"mime"
	"net/http"
	"os"
	"path"
	"strings"

	"github.com/schwarzlichtbezirk/wpk"
)

// command line settings
var (
	srcpath string
	SrcList []string
	DstFile string
	PutMIME bool
	PutLink bool
	ShowLog bool
	Split   bool
)

func parseargs() {
	flag.StringVar(&srcpath, "src", "", "full path to folder with source files to be packaged, or list of folders divided by ';'")
	flag.StringVar(&DstFile, "dst", "", "full path to output package file")
	flag.BoolVar(&PutMIME, "mime", false, "put content MIME type defined by file extension to each file tagset")
	flag.BoolVar(&PutLink, "link", false, "put full path to the original file to each file tagset")
	flag.BoolVar(&ShowLog, "log", true, "show process log for each extracting file")
	flag.BoolVar(&Split, "split", false, "write package to splitted files")
	flag.Parse()
}

func checkargs() (ec int) { // returns error counter
	for i, fpath := range strings.Split(srcpath, ";") {
		if fpath == "" {
			continue
		}
		fpath = wpk.ToSlash(wpk.Envfmt(fpath, nil))
		if !strings.HasSuffix(fpath, "/") {
			fpath += "/"
		}
		if ok, _ := wpk.DirExists(fpath); !ok {
			log.Printf("source path #%d '%s' does not exist", i+1, fpath)
			ec++
			continue
		}
		SrcList = append(SrcList, fpath)
	}
	if len(SrcList) == 0 {
		log.Println("source path does not specified")
		ec++
	}

	DstFile = wpk.ToSlash(wpk.Envfmt(DstFile, nil))
	if DstFile == "" {
		log.Println("destination file does not specified")
		ec++
	} else if ok, _ := wpk.DirExists(path.Dir(DstFile)); !ok {
		log.Println("destination path does not exist")
		ec++
	}

	return
}

func writepackage() (err error) {
	var fwpk, fwpf wpk.WriteSeekCloser
	var pkgfile, datfile = DstFile, DstFile
	var pkg = wpk.NewPackage()
	if Split {
		pkgfile, datfile = wpk.MakeTagsPath(pkgfile), wpk.MakeDataPath(datfile)
	}

	// open package file to write
	if fwpk, err = os.OpenFile(pkgfile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644); err != nil {
		return
	}
	defer fwpk.Close()

	if Split {
		if fwpf, err = os.OpenFile(datfile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644); err != nil {
			return
		}
		defer fwpf.Close()

		log.Printf("destination tags part:  %s\n", pkgfile)
		log.Printf("destination files part: %s\n", datfile)
	} else {
		log.Printf("destination file: %s\n", pkgfile)
	}

	// starts new package
	if err = pkg.Begin(fwpk, fwpf); err != nil {
		return
	}

	// data writer
	var w = fwpk
	if fwpf != nil {
		w = fwpf
	}

	// write all source folders
	for i, srcpath := range SrcList {
		log.Printf("source folder #%d: %s", i+1, srcpath)
		var num, sum int64
		fs.WalkDir(os.DirFS(srcpath), ".", func(fkey string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				return nil // file is directory
			}

			var fpath = wpk.JoinPath(srcpath, fkey)
			var file wpk.RFile
			var ts wpk.TagsetRaw
			if file, err = os.Open(fpath); err != nil {
				return err
			}
			defer file.Close()

			if ts, err = pkg.PackFile(w, file, fkey); err != nil {
				return err
			}

			var size = ts.Size()
			num++
			sum += size
			if ShowLog {
				log.Printf("#%-4d %7d bytes   %s", num, size, fkey)
			}

			// adjust tags
			if PutMIME {
				const sniffLen = 512
				var ctype = mime.TypeByExtension(path.Ext(fkey))
				if ctype == "" {
					// rewind to file start
					if _, err = file.Seek(0, io.SeekStart); err != nil {
						return err
					}
					// read a chunk to decide between utf-8 text and binary
					var buf [sniffLen]byte
					var n int64
					if n, err = io.CopyN(bytes.NewBuffer(buf[:]), file, sniffLen); err != nil && err != io.EOF {
						return err
					}
					ctype = http.DetectContentType(buf[:n])
				}
				if ctype != "" {
					ts = ts.Put(wpk.TIDmime, wpk.StrTag(ctype))
				}
			}
			if PutLink {
				ts = ts.Put(wpk.TIDlink, wpk.StrTag(fpath))
			}
			pkg.SetTagset(fkey, ts)
			return nil
		})
		log.Printf("packed: %d files on %d bytes", num, sum)
	}

	// finalize
	log.Printf("write tags table")
	if err = pkg.Sync(fwpk, fwpf); err != nil {
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
