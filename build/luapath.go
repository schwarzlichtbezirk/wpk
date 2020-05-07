package main

import (
	"os"
	"path/filepath"

	"github.com/yuin/gopher-lua"
)

func OpenPath(ls *lua.LState) int {
	var mod = ls.RegisterModule("path", pathfuncs).(*lua.LTable)
	mod.RawSetString("sep", lua.LString(filepath.Separator))
	ls.Push(mod)
	return 1
}

var pathfuncs = map[string]lua.LGFunction{
	"toslash": pathtoslash,
	"volume":  pathvolume,
	"dir":     pathdir,
	"base":    pathbase,
	"ext":     pathext,
	"split":   pathsplit,
	"match":   pathmatch,
	"join":    pathjoin,
	"glob":    pathglob,
	"enum":    pathenum,
}

func pathtoslash(ls *lua.LState) int {
	var filename = ls.CheckString(1)

	var name = filepath.ToSlash(filename)
	ls.Push(lua.LString(name))
	return 1
}

func pathvolume(ls *lua.LState) int {
	var filename = ls.CheckString(1)

	var vol = filepath.VolumeName(filename)
	ls.Push(lua.LString(vol))
	return 1
}

func pathdir(ls *lua.LState) int {
	var filename = ls.CheckString(1)

	var dir = filepath.Dir(filename)
	ls.Push(lua.LString(dir))
	return 1
}

func pathbase(ls *lua.LState) int {
	var filename = ls.CheckString(1)

	var base = filepath.Base(filename)
	ls.Push(lua.LString(base))
	return 1
}

func pathext(ls *lua.LState) int {
	var filename = ls.CheckString(1)

	var ext = filepath.Ext(filename)
	ls.Push(lua.LString(ext))
	return 1
}

func pathsplit(ls *lua.LState) int {
	var filename = ls.CheckString(1)

	var dir, file = filepath.Split(filename)
	ls.Push(lua.LString(dir))
	ls.Push(lua.LString(file))
	return 2
}

func pathmatch(ls *lua.LState) int {
	var name = ls.CheckString(1)
	var pattern = ls.CheckString(2)

	var matched, err = filepath.Match(name, pattern)
	if err != nil {
		ls.RaiseError(err.Error())
		return 0
	}
	ls.Push(lua.LBool(matched))
	return 1
}

func pathjoin(ls *lua.LState) int {
	var elem = make([]string, ls.GetTop())
	for i := range elem {
		elem[i] = ls.CheckString(i + 1)
	}

	var dir = filepath.Join(elem...)
	ls.Push(lua.LString(dir))
	return 1
}

func pathglob(ls *lua.LState) int {
	var pattern = ls.CheckString(1)

	var matches, err = filepath.Glob(pattern)
	if err != nil {
		ls.RaiseError(err.Error())
		return 0
	}
	for _, dir := range matches {
		ls.Push(lua.LString(dir))
	}
	return len(matches)
}

func pathenum(ls *lua.LState) int {
	var dirname = ls.CheckString(1)
	var n = ls.OptInt(2, -1)

	var dir, err = os.Open(dirname)
	if err != nil {
		ls.RaiseError(err.Error())
		return 0
	}
	defer dir.Close()

	var names []string
	if names, err = dir.Readdirnames(n); err != nil {
		ls.RaiseError(err.Error())
		return 0
	}

	var tb = ls.CreateTable(len(names), 0)
	for i, name := range names {
		tb.RawSetInt(i+1, lua.LString(name))
		i++
	}
	ls.Push(tb)
	return 1
}

// The End.
