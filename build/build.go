package main

import (
	"log"
	"os"
	"path/filepath"
	"regexp"

	"github.com/yuin/gopher-lua"
)

var efre = regexp.MustCompile(`\$\(\w+\)`)

func envfmt(p string) string {
	return filepath.ToSlash(efre.ReplaceAllStringFunc(p, func(name string) string {
		return os.Getenv(name[2 : len(name)-1]) // strip $(...) and replace by env value
	}))
}

func lualog(ls *lua.LState) int {
	var s = ls.CheckString(1)

	log.Println(s)
	return 0
}

func luacheckfile(ls *lua.LState) int {
	var fpath = ls.CheckString(1)

	var err error
	var fi os.FileInfo
	if fi, err = os.Stat(fpath); err == nil {
		ls.Push(lua.LBool(true))
		ls.Push(lua.LBool(fi.IsDir()))
		return 2
	}
	if os.IsNotExist(err) {
		ls.Push(lua.LBool(false))
		return 1
	}
	ls.Push(lua.LBool(false))
	ls.Push(lua.LString(err.Error()))
	return 2
}

func mainluavm(fpath string) (err error) {
	var ls = lua.NewState()
	defer ls.Close()

	OpenPath(ls)
	RegTag(ls)
	RegPack(ls)

	var bindir = filepath.ToSlash(filepath.Dir(os.Args[0])) + "/"
	var scrdir = filepath.ToSlash(filepath.Dir(fpath)) + "/"
	ls.SetGlobal("bindir", lua.LString(bindir))
	ls.SetGlobal("scrdir", lua.LString(scrdir))
	ls.SetGlobal("tmpdir", lua.LString(filepath.ToSlash(os.TempDir())))
	ls.SetGlobal("log", ls.NewFunction(lualog))
	ls.SetGlobal("checkfile", ls.NewFunction(luacheckfile))

	if err = ls.DoFile(fpath); err != nil {
		return
	}
	return
}

func main() {
	for _, path := range os.Args[1:] {
		if err := mainluavm(envfmt(path)); err != nil {
			log.Println(err.Error())
			return
		}
	}
}

// The End.
