package main

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/schwarzlichtbezirk/wpk"
	"github.com/yuin/gopher-lua"
)

const PackMT = "wpk"

type LuaPackage struct {
	wpk.Package
	automime bool
	crc32    bool
	md5      bool
	sha256   bool
}

func RegPack(ls *lua.LState) {
	var mt = ls.NewTypeMetatable(PackMT)
	ls.SetGlobal(PackMT, mt)
	// static attributes
	ls.SetField(mt, "new", ls.NewFunction(NewPack))
	// methods
	ls.SetField(mt, "__index", ls.NewFunction(getter_pack))
	ls.SetField(mt, "__newindex", ls.NewFunction(setter_pack))
	ls.SetField(mt, "__tostring", ls.NewFunction(tostring_pack))
	for name, f := range methods_pack {
		ls.SetField(mt, name, ls.NewFunction(f))
	}
	for i, p := range properties_pack {
		ls.SetField(mt, p.name, lua.LNumber(i))
	}
}

func PushPack(ls *lua.LState, v *LuaPackage) {
	var ud = ls.NewUserData()
	ud.Value = v
	ls.SetMetatable(ud, ls.GetTypeMetatable(PackMT))
	ls.Push(ud)
}

// LuaPackage constructor.
func NewPack(ls *lua.LState) int {
	var pack LuaPackage
	pack.Init()
	PushPack(ls, &pack)
	return 1
}

// Checks whether the lua argument with given number is
// a *LUserData with *LuaPackage and returns this *LuaPackage.
func CheckPack(ls *lua.LState, arg int) *LuaPackage {
	var ud = ls.CheckUserData(arg)
	if v, ok := ud.Value.(*LuaPackage); ok {
		return v
	}
	ls.ArgError(arg, PackMT+" object required")
	return nil
}

func getter_pack(ls *lua.LState) int {
	var mt = ls.GetMetatable(ls.Get(1))
	var val = ls.GetField(mt, ls.CheckString(2))
	switch val := val.(type) {
	case *lua.LFunction:
		ls.Push(val)
		return 1
	case lua.LNumber:
		var l = &properties_pack[int(val)]
		if l.getter == nil {
			ls.RaiseError("no getter \"%s\" of class \"%s\" defined", l.name, PackMT)
			return 0
		}
		ls.Remove(2) // remove getter name
		return l.getter(ls)
	default:
		ls.Push(lua.LNil)
		return 1
	}
}

func setter_pack(ls *lua.LState) int {
	var mt = ls.GetMetatable(ls.Get(1))
	var val = ls.GetField(mt, ls.CheckString(2))
	switch val := val.(type) {
	case *lua.LFunction:
		ls.Push(val)
		return 1
	case lua.LNumber:
		var l = &properties_pack[int(val)]
		if l.setter == nil {
			ls.RaiseError("no setter \"%s\" of class \"%s\" defined", l.name, PackMT)
			return 0
		}
		ls.Remove(2) // remove setter name
		return l.setter(ls)
	default:
		ls.RaiseError("internal error, wrong pointer type at userdata metatable")
		return 0
	}
}

func tostring_pack(ls *lua.LState) int {
	var pack = CheckPack(ls, 1)

	var size int64 = 0
	var refs int64 = 0
	for _, rec := range pack.FAT {
		if rec.Size > 0 {
			size += rec.Size
		} else if rec.Size == wpk.DATAREF {
			refs++
		}
	}
	var s = fmt.Sprintf("records: %d, total size: %d, references: %d", len(pack.FAT), size, refs)
	ls.Push(lua.LString(s))
	return 1
}

var properties_pack = []struct {
	name   string
	getter lua.LGFunction // getters always must return 1 value
	setter lua.LGFunction // setters always must return no values
}{
	{"recnum", getrecnum, nil},
	{"tagnum", gettagnum, nil},
	{"automime", getautomime, setautomime},
	{"crc32", getcrc32, setcrc32},
	{"md5", getmd5, setmd5},
	{"sha256", getsha256, setsha256},
}

var methods_pack = map[string]lua.LGFunction{
	"rename":    rename,
	"putalias":  putalias,
	"deltagset": deltagset,
}

// properties section

func getrecnum(ls *lua.LState) int {
	var pack = CheckPack(ls, 1)
	ls.Push(lua.LNumber(len(pack.FAT)))
	return 1
}

func gettagnum(ls *lua.LState) int {
	var pack = CheckPack(ls, 1)
	ls.Push(lua.LNumber(len(pack.Tags)))
	return 1
}

func getautomime(ls *lua.LState) int {
	var pack = CheckPack(ls, 1)
	ls.Push(lua.LBool(pack.automime))
	return 1
}

func setautomime(ls *lua.LState) int {
	var pack = CheckPack(ls, 1)
	var val = ls.CheckBool(2)

	pack.automime = val
	return 0
}

func getcrc32(ls *lua.LState) int {
	var pack = CheckPack(ls, 1)
	ls.Push(lua.LBool(pack.crc32))
	return 1
}

func setcrc32(ls *lua.LState) int {
	var pack = CheckPack(ls, 1)
	var val = ls.CheckBool(2)

	pack.crc32 = val
	return 0
}

func getmd5(ls *lua.LState) int {
	var pack = CheckPack(ls, 1)
	ls.Push(lua.LBool(pack.md5))
	return 1
}

func setmd5(ls *lua.LState) int {
	var pack = CheckPack(ls, 1)
	var val = ls.CheckBool(2)

	pack.md5 = val
	return 0
}

func getsha256(ls *lua.LState) int {
	var pack = CheckPack(ls, 1)
	ls.Push(lua.LBool(pack.sha256))
	return 1
}

func setsha256(ls *lua.LState) int {
	var pack = CheckPack(ls, 1)
	var val = ls.CheckBool(2)

	pack.sha256 = val
	return 0
}

// methods section

// Renames tag set with file name fname1 to fname2.
// rename(fname1, fname2)
//   fname1 - old file name
//   fname2 - new file name
func rename(ls *lua.LState) int {
	var pack = CheckPack(ls, 1)
	var fname1 = ls.CheckString(2)
	var fname2 = ls.CheckString(3)

	var key1 = strings.ToLower(filepath.ToSlash(fname1))
	var key2 = strings.ToLower(filepath.ToSlash(fname2))
	var tags, ok = pack.Tags[key1]
	if !ok {
		ls.ArgError(2, fmt.Sprintf("file with name '%s' does not present", fname1))
		return 0
	}
	if _, ok = pack.Tags[key2]; ok {
		ls.ArgError(3, fmt.Sprintf("can not create alias with name '%s' because file with this name already present", fname2))
		return 0
	}

	tags.SetString(wpk.AID_name, fname2)
	delete(pack.Tags, key1) // delete at first if fname1 == fname2
	pack.Tags[key2] = tags
	return 0
}

// Creates copy of tag set with new file name.
// putalias(fname1, fname2)
//   fname1 - file name of packaged file
//   fname2 - new file name that will be reference to fname1 file data
func putalias(ls *lua.LState) int {
	var pack = CheckPack(ls, 1)
	var fname1 = ls.CheckString(2)
	var fname2 = ls.CheckString(3)

	var key1 = strings.ToLower(filepath.ToSlash(fname1))
	var key2 = strings.ToLower(filepath.ToSlash(fname2))
	var tags1, ok = pack.Tags[key1]
	if !ok {
		ls.ArgError(2, fmt.Sprintf("file with name '%s' does not present", fname1))
		return 0
	}
	if _, ok = pack.Tags[key2]; ok {
		ls.ArgError(3, fmt.Sprintf("can not create alias with name '%s' because file with this name already present", fname2))
		return 0
	}

	var tags2 = wpk.TagSet{}
	for k, v := range tags1 {
		tags2[k] = v
	}
	tags2.SetString(wpk.AID_name, fname2)
	pack.Tags[key2] = tags2
	return 0
}

// Deletes tag set with given file name. Data block will still persist.
func deltagset(ls *lua.LState) int {
	var pack = CheckPack(ls, 1)
	var fname = ls.CheckString(2)

	var key = strings.ToLower(filepath.ToSlash(fname))
	var _, ok = pack.Tags[key]
	if ok {
		delete(pack.Tags, key)
	}
	ls.Push(lua.LBool(ok))
	return 1
}

// The End.
