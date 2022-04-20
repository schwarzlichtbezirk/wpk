package main

import (
	"flag"
	"io"
	"log"
	"os"
	"path"
	"path/filepath"
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
	ShowLog bool
	PkgMode string
)

var (
	pack wpk.Packager
)

func parseargs() {
	flag.StringVar(&srcfile, "src", "", "package full file name, or list of files divided by ';'")
	flag.StringVar(&DstPath, "dst", "", "full destination path for output extracted files")
	flag.BoolVar(&MkDst, "md", false, "create destination path if it does not exist")
	flag.BoolVar(&ShowLog, "sl", true, "show process log for each extracting file")
	flag.StringVar(&PkgMode, "pm", "mmap", "package opening mode, can be \"bulk\", \"mmap\" and \"fsys\"")
	flag.Parse()
}

func checkargs() int {
	var ec = 0 // error counter

	srcfile = filepath.ToSlash(strings.Trim(srcfile, ";"))
	if srcfile == "" {
		log.Println("package file does not specified")
		ec++
	}
	for i, file := range strings.Split(srcfile, ";") {
		if file == "" {
			continue
		}
		file = wpk.Envfmt(file)
		if ok, _ := wpk.PathExists(file); !ok {
			log.Printf("source file #%d '%s' does not exist", i+1, file)
			ec++
			continue
		}
		SrcList = append(SrcList, file)
	}

	DstPath = wpk.Envfmt(DstPath)
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

func readpackage() (err error) {
	log.Printf("destination path: %s", DstPath)

	for _, pkgpath := range SrcList {
		log.Printf("source package: %s", pkgpath)
		if func() {
			switch PkgMode {
			case "bulk":
				if pack, err = bulk.OpenImage(pkgpath); err != nil {
					return
				}
			case "mmap":
				if pack, err = mmap.OpenImage(pkgpath); err != nil {
					return
				}
			case "fsys":
				if pack, err = fsys.OpenImage(pkgpath); err != nil {
					return
				}
			default:
				panic("no way to here")
			}
			defer pack.Close()

			var num, sum int64
			pack.Enum(func(fkey string, ts *wpk.Tagset_t) (next bool) {
				defer func() {
					next = err == nil
				}()
				var fpath = ts.Path()

				var fullpath = path.Join(DstPath, fpath)
				if err = os.MkdirAll(filepath.Dir(fullpath), os.ModePerm); err != nil {
					return
				}

				var src wpk.NestedFile
				if src, err = pack.OpenTags(*ts); err != nil {
					return
				}
				defer src.Close()

				var dst *os.File
				if dst, err = os.OpenFile(fullpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0755); err != nil {
					return
				}
				defer dst.Close()

				var n int64
				if n, err = io.Copy(dst, src); err != nil {
					return
				}
				num++
				sum += n

				if ShowLog {
					var fid, _ = ts.FID()
					log.Printf("#%-3d %6d bytes   %s", fid, n, fpath)
				}
				return
			})
			log.Printf("unpacked: %d files on %d bytes", num, sum)
		}(); err != nil {
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
