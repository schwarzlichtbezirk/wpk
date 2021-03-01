package main

import (
	"flag"
	"io"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/schwarzlichtbezirk/wpk/mmap"
)

// command line settings
var (
	srcfile string
	SrcList []string
	DstPath string
	MkDst   bool
)

var efre = regexp.MustCompile(`\$\{\w+\}`)

func envfmt(p string) string {
	return filepath.ToSlash(efre.ReplaceAllStringFunc(p, func(name string) string {
		return os.Getenv(name[2 : len(name)-1]) // strip ${...} and replace by env value
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
		file = envfmt(file)
		if ok, _ := pathexists(file); !ok {
			log.Printf("source file #%d '%s' does not exist", i+1, file)
			ec++
			continue
		}
		SrcList = append(SrcList, file)
	}

	DstPath = envfmt(DstPath)
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
			var pack mmap.PackDir
			var sum int64
			if err = pack.OpenWPK(pkgpath); err != nil {
				return
			}

			var tat = pack.Enum()
			for key := range tat {
				if func() {
					var ts, _ = pack.NamedTags(key)
					var kpath = ts.Path()

					var fpath = DstPath + kpath
					if err = os.MkdirAll(filepath.Dir(fpath), os.ModePerm); err != nil {
						return
					}

					var src, err = pack.OpenFile(ts)
					if err != nil {
						return
					}
					defer src.Close()

					var dst *os.File
					if dst, err = os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0755); err != nil {
						return
					}
					defer dst.Close()

					var n int64
					if n, err = io.Copy(dst, src); err != nil {
						return
					}
					sum += n

					log.Printf("#%-3d %6d bytes   %s", ts.FID(), n, kpath)
				}(); err != nil {
					return
				}
			}
			log.Printf("unpacked: %d files on %d bytes", len(tat), sum)
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
