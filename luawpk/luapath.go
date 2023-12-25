package luawpk

import (
	"os"
	"path"
	"path/filepath"

	"github.com/schwarzlichtbezirk/wpk"
	lua "github.com/yuin/gopher-lua"
)

// OpenPath registers "path" namespace into Lua virtual machine.
func OpenPath(ls *lua.LState) int {
	var mod = ls.RegisterModule("path", pathfuncs).(*lua.LTable)
	mod.RawSetString("sep", lua.LString("/"))
	ls.Push(mod)
	return 1
}

var pathfuncs = map[string]lua.LGFunction{
	"toslash": pathtoslash,
	"clean":   pathclean,
	"volume":  pathvolume,
	"dir":     pathdir,
	"base":    pathbase,
	"name":    pathname,
	"ext":     pathext,
	"split":   pathsplit,
	"match":   pathmatch,
	"join":    pathjoin,
	"glob":    pathglob,
	"enum":    pathenum,
	"envfmt":  pathenvfmt,
}

func pathtoslash(ls *lua.LState) int {
	var fpath = ls.CheckString(1)

	ls.Push(lua.LString(wpk.ToSlash(fpath)))
	return 1
}

func pathclean(ls *lua.LState) int {
	var fpath = ls.CheckString(1)

	var path = path.Clean(fpath)
	ls.Push(lua.LString(path))
	return 1
}

func pathvolume(ls *lua.LState) int {
	var fpath = ls.CheckString(1)

	var vol = filepath.VolumeName(fpath)
	ls.Push(lua.LString(vol))
	return 1
}

func pathdir(ls *lua.LState) int {
	var fpath = ls.CheckString(1)

	var dir = path.Dir(fpath)
	ls.Push(lua.LString(dir))
	return 1
}

func pathbase(ls *lua.LState) int {
	var fpath = ls.CheckString(1)

	var base = path.Base(fpath)
	ls.Push(lua.LString(base))
	return 1
}

func pathname(ls *lua.LState) int {
	var fpath = ls.CheckString(1)

	var name = wpk.PathName(fpath)
	ls.Push(lua.LString(name))
	return 1
}

func pathext(ls *lua.LState) int {
	var fpath = ls.CheckString(1)

	var ext = path.Ext(fpath)
	ls.Push(lua.LString(ext))
	return 1
}

func pathsplit(ls *lua.LState) int {
	var fpath = ls.CheckString(1)

	var dir, file = path.Split(fpath)
	ls.Push(lua.LString(dir))
	ls.Push(lua.LString(file))
	return 2
}

func pathmatch(ls *lua.LState) int {
	var name = ls.CheckString(1)
	var pattern = ls.CheckString(2)

	var matched, err = path.Match(name, pattern)
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

	var dir = path.Join(elem...)
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
		ls.Push(lua.LString(wpk.ToSlash(dir)))
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
	}
	ls.Push(tb)
	return 1
}

func pathenvfmt(ls *lua.LState) int {
	var fpath = ls.CheckString(1)
	ls.Push(lua.LString(wpk.Envfmt(fpath, nil)))
	return 1
}

// The End.
