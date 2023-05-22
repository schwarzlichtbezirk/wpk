package main

import (
	"flag"
	"io"
	"log"
	"os"
	"path"
	"strings"

	"github.com/schwarzlichtbezirk/wpk"
	"github.com/schwarzlichtbezirk/wpk/bulk"
	"github.com/schwarzlichtbezirk/wpk/fsys"
	"github.com/schwarzlichtbezirk/wpk/mmap"
)

// command line settings
var (
	srcfile string
	SrcList []string
	DstPath string
	MkDst   bool
	OrgTime bool
	ShowLog bool
	PkgMode string
)

var pkg *wpk.Package

func parseargs() {
	flag.StringVar(&srcfile, "src", "", "package full file name, or list of files divided by ';'")
	flag.StringVar(&DstPath, "dst", "", "full destination path for output extracted files")
	flag.BoolVar(&MkDst, "md", false, "create destination path if it does not exist")
	flag.BoolVar(&OrgTime, "ft", false, "change the access and modification times of extracted files to original file times")
	flag.BoolVar(&ShowLog, "sl", true, "show process log for each extracting file")
	flag.StringVar(&PkgMode, "pm", "mmap", "package opening mode, can be \"bulk\", \"mmap\" and \"fsys\"")
	flag.Parse()
}

func checkargs() int {
	var ec = 0 // error counter

	for i, fpath := range strings.Split(srcfile, ";") {
		if fpath == "" {
			continue
		}
		fpath = wpk.ToSlash(wpk.Envfmt(fpath))
		if ok, _ := wpk.PathExists(fpath); !ok {
			log.Printf("source file #%d '%s' does not exist", i+1, fpath)
			ec++
			continue
		}
		SrcList = append(SrcList, fpath)
	}
	if len(srcfile) == 0 {
		log.Println("package file does not specified")
		ec++
	}

	DstPath = wpk.ToSlash(wpk.Envfmt(DstPath))
	if DstPath == "" {
		log.Println("destination path does not specified")
		ec++
	}
	if ok, _ := wpk.PathExists(DstPath); !ok {
		if MkDst {
			if err := os.MkdirAll(DstPath, os.ModePerm); err != nil {
				log.Println(err.Error())
				ec++
			}
		} else {
			log.Println("destination path does not exist")
			ec++
		}
	}

	if PkgMode != "bulk" && PkgMode != "mmap" && PkgMode != "fsys" {
		log.Println("given package opening type does not supported")
		ec++
	}

	return ec
}

func openpackage(pkgpath string) (err error) {
	if pkg, err = wpk.OpenPackage(pkgpath); err != nil {
		return
	}
	var fpath string
	if pkg.IsSplitted() {
		fpath = wpk.MakeDataPath(pkgpath)
	} else {
		fpath = pkgpath
	}
	switch PkgMode {
	case "bulk":
		if pkg.Tagger, err = bulk.MakeTagger(fpath); err != nil {
			return
		}
	case "mmap":
		if pkg.Tagger, err = mmap.MakeTagger(fpath); err != nil {
			return
		}
	case "fsys":
		if pkg.Tagger, err = fsys.MakeTagger(fpath); err != nil {
			return
		}
	default:
		panic("no way to here")
	}
	return
}

func readpackage() (err error) {
	log.Printf("destination path: %s", DstPath)

	for _, pkgpath := range SrcList {
		log.Printf("source package: %s", pkgpath)
		func() {
			if err = openpackage(pkgpath); err != nil {
				return
			}
			defer pkg.Close()

			var num, sum int64
			pkg.Enum(func(fkey string, ts *wpk.TagsetRaw) (next bool) {
				defer func() {
					next = err == nil
				}()
				var fpath = ts.Path()

				var fullpath = path.Join(DstPath, fpath)
				if err = os.MkdirAll(path.Dir(fullpath), os.ModePerm); err != nil {
					return
				}

				var src wpk.NestedFile
				if src, err = pkg.OpenTagset(ts); err != nil {
					return
				}
				defer src.Close()

				var dst io.WriteCloser
				if dst, err = os.OpenFile(fullpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0755); err != nil {
					return
				}
				if OrgTime {
					var atime, aok = ts.TagTime(wpk.TIDatime)
					var mtime, mok = ts.TagTime(wpk.TIDmtime)
					if aok && mok {
						defer os.Chtimes(fullpath, atime, mtime)
					}
				}
				defer dst.Close()

				var n int64
				if n, err = io.Copy(dst, src); err != nil {
					return
				}
				num++
				sum += n

				if ShowLog {
					log.Printf("#%-3d %6d bytes   %s", num, n, fpath)
				}
				return
			})
			log.Printf("unpacked: %d files on %d bytes", num, sum)
		}()
		if err != nil {
			return
		}
	}

	return
}

func main() {
	parseargs()
	if checkargs() > 0 {
		return
	}

	log.Println("starts")
	if err := readpackage(); err != nil {
		log.Println(err.Error())
		return
	}
	log.Println("done.")
}

// The End.
