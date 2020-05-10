package main

import (
	"flag"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/schwarzlichtbezirk/wpk"
)

// command line settings
var (
	srcfile string
	SrcList []string
	DstPath string
	MkDst   bool
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
	flag.StringVar(&srcfile, "src", "", "package full file name, or list of files divided by ';'")
	flag.StringVar(&DstPath, "dst", "", "full destination path for output extracted files")
	flag.BoolVar(&MkDst, "md", false, "create destination path if it does not exist")
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
		if ok, _ := pathexists(file); !ok {
			log.Printf("source file #%d '%s' does not exist", i+1, file)
			ec++
			continue
		}
		SrcList = append(SrcList, file)
	}

	DstPath = filepath.ToSlash(DstPath)
	if DstPath == "" {
		log.Println("destination path does not specified")
		ec++
	} else if !strings.HasSuffix(DstPath, "/") {
		DstPath += "/"
	}
	if ok, _ := pathexists(DstPath); !ok {
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

	return ec
}

func readpackage() (err error) {
	log.Printf("destination path: %s", DstPath)

	for _, pkgpath := range SrcList {
		log.Printf("source package: %s", pkgpath)
		if func() {
			var pack wpk.Package

			var src *os.File
			if src, err = os.Open(pkgpath); err != nil {
				return
			}
			defer src.Close()

			if err = pack.Open(src); err != nil {
				return
			}

			for fname, tags := range pack.Tags {
				var fid, _ = tags.Uint32(wpk.TID_FID)
				var rec = &pack.FAT[fid]
				log.Printf("#%-4d %7d bytes   %s", fid, rec.Size, fname)

				if func() {
					var fpath = DstPath+fname
					if err = os.MkdirAll(filepath.Dir(fpath), os.ModePerm); err != nil {
						return
					}

					var dst *os.File
					if dst, err = os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0755); err != nil {
						return
					}
					defer dst.Close()

					if _, err = src.Seek(int64(rec.Offset), io.SeekStart); err != nil {
						return
					}
					if _, err = io.CopyN(dst, src, int64(rec.Size)); err != nil {
						return
					}
				}(); err != nil {
					return
				}
			}
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