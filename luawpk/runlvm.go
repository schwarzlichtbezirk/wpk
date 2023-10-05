package luawpk

import (
	"errors"
	"io/fs"
	"log"
	"os"
	"path"

	"github.com/schwarzlichtbezirk/wpk"
	lua "github.com/yuin/gopher-lua"
)

var (
	// compiled binary version, sets by compiler with command
	//    go build -ldflags="-X 'github.com/schwarzlichtbezirk/wpk/luawpk.BuildVers=%buildvers%'"
	BuildVers string

	// compiled binary build date, sets by compiler with command
	//    go build -ldflags="-X 'github.com/schwarzlichtbezirk/wpk/luawpk.BuildTime=%buildtime%'"
	BuildTime string
)

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
	if errors.Is(err, fs.ErrNotExist) {
		ls.Push(lua.LBool(false))
		return 1
	}
	ls.Push(lua.LBool(false))
	ls.Push(lua.LString(err.Error()))
	return 2
}

// RunLuaVM runs specified Lua-script with Lua WPK API.
func RunLuaVM(fpath string) (err error) {
	var ls = lua.NewState()
	defer ls.Close()

	OpenPath(ls)
	RegPack(ls)

	var bindir = path.Dir(wpk.ToSlash(os.Args[0]))
	var scrdir = path.Dir(wpk.ToSlash(fpath))

	ls.SetGlobal("buildvers", lua.LString(BuildVers))
	ls.SetGlobal("buildtime", lua.LString(BuildTime))
	ls.SetGlobal("bindir", lua.LString(bindir))
	ls.SetGlobal("scrdir", lua.LString(scrdir))
	ls.SetGlobal("tmpdir", lua.LString(wpk.TempPath(".")))
	ls.SetGlobal("log", ls.NewFunction(lualog))
	ls.SetGlobal("checkfile", ls.NewFunction(luacheckfile))

	if err = ls.DoFile(fpath); err != nil {
		return
	}
	return
}

// The End.
