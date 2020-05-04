package main

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/schwarzlichtbezirk/wpk"
	"github.com/yuin/gopher-lua"
)

const PackMT = "wpk"

type LuaPackage struct {
	wpk.Package
	secret   string
	automime bool
	crc32    bool
	md5      bool
	sha1     bool
	sha224   bool
	sha256   bool

	w *os.File
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
	for _, rec := range pack.FAT {
		if rec.Size > 0 {
			size += rec.Size
		}
	}
	var s = fmt.Sprintf("records: %d, tags: %d, total size: %d", len(pack.FAT), len(pack.Tags), size)
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
	{"secret", getsecret, setsecret},
	{"crc32", getcrc32, setcrc32},
	{"md5", getmd5, setmd5},
	{"sha1", getsha1, setsha1},
	{"sha224", getsha224, setsha224},
	{"sha256", getsha256, setsha256},
}

var methods_pack = map[string]lua.LGFunction{
	"begin":     begin,
	"complete":  complete,
	"putfile":   putfile,
	"putstring": putstring,
	"datasize":  datasize,
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

func getsecret(ls *lua.LState) int {
	var pack = CheckPack(ls, 1)
	ls.Push(lua.LString(pack.secret))
	return 1
}

func setsecret(ls *lua.LState) int {
	var pack = CheckPack(ls, 1)
	var val = ls.CheckString(2)

	pack.secret = val
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

func getsha1(ls *lua.LState) int {
	var pack = CheckPack(ls, 1)
	ls.Push(lua.LBool(pack.sha1))
	return 1
}

func setsha1(ls *lua.LState) int {
	var pack = CheckPack(ls, 1)
	var val = ls.CheckBool(2)

	pack.sha1 = val
	return 0
}

func getsha224(ls *lua.LState) int {
	var pack = CheckPack(ls, 1)
	ls.Push(lua.LBool(pack.sha224))
	return 1
}

func setsha224(ls *lua.LState) int {
	var pack = CheckPack(ls, 1)
	var val = ls.CheckBool(2)

	pack.sha224 = val
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

func begin(ls *lua.LState) int {
	var pack = CheckPack(ls, 1)
	var pkgname = ls.CheckString(2)

	if pack.w != nil {
		ls.RaiseError("package write stream already opened")
		return 0
	}

	var err error

	// create package file
	var dst *os.File
	if dst, err = os.OpenFile(pkgname, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0755); err != nil {
		ls.RaiseError(err.Error())
		return 0
	}

	pack.Init()
	pack.w = dst

	// write prebuild header
	if err = binary.Write(pack.w, binary.LittleEndian, &pack.PackHdr); err != nil {
		ls.RaiseError(err.Error())
		return 0
	}

	return 0
}

func complete(ls *lua.LState) int {
	var pack = CheckPack(ls, 1)

	if pack.w == nil {
		ls.RaiseError("package write stream does not opened")
		return 0
	}

	var err error

	// write records table
	if pack.RecOffset, err = pack.w.Seek(0, io.SeekEnd); err != nil {
		ls.RaiseError(err.Error())
		return 0
	}
	pack.RecNumber = int64(len(pack.FAT))
	if err = binary.Write(pack.w, binary.LittleEndian, &pack.FAT); err != nil {
		ls.RaiseError(err.Error())
		return 0
	}

	// write files tags table
	if pack.TagOffset, err = pack.w.Seek(0, io.SeekCurrent); err != nil {
		ls.RaiseError(err.Error())
		return 0
	}
	pack.TagNumber = int64(len(pack.Tags))
	for _, tags := range pack.Tags {
		if err = tags.Write(pack.w); err != nil {
			ls.RaiseError(err.Error())
			return 0
		}
	}

	// rewrite true header
	if _, err = pack.w.Seek(0, io.SeekStart); err != nil {
		ls.RaiseError(err.Error())
		return 0
	}
	copy(pack.Signature[:], wpk.Signature)
	if err = binary.Write(pack.w, binary.LittleEndian, &pack.PackHdr); err != nil {
		ls.RaiseError(err.Error())
		return 0
	}

	// close package file
	if err = pack.w.Close(); err != nil {
		ls.RaiseError(err.Error())
		return 0
	}
	pack.w = nil

	return 0
}

func putfile(ls *lua.LState) int {
	var pack = CheckPack(ls, 1)
	var fname = ls.CheckString(2)
	var fpath = ls.CheckString(3)

	var key = strings.ToLower(filepath.ToSlash(fname))
	if _, is := pack.Tags[key]; is {
		ls.RaiseError("file with name '%s' already present", fname)
		return 0
	}

	var err error
	if func() {
		var file *os.File
		if file, err = os.Open(fpath); err != nil {
			return
		}
		defer func() {
			err = file.Close()
		}()

		var fi os.FileInfo
		if fi, err = file.Stat(); err != nil {
			return
		}
		var tags wpk.TagSet
		if tags, err = pack.PackData(pack.w, file, fname, fi.ModTime().Unix()); err != nil {
			return
		}
		if err = pack.adjusttagset(file, tags); err != nil {
			return
		}
	}(); err != nil {
		ls.RaiseError(err.Error())
		return 0
	}

	return 0
}

func putstring(ls *lua.LState) int {
	var pack = CheckPack(ls, 1)
	var fname = ls.CheckString(2)
	var data = ls.CheckString(3)

	var key = strings.ToLower(filepath.ToSlash(fname))
	if _, is := pack.Tags[key]; is {
		ls.RaiseError("file with name '%s' already present", fname)
		return 0
	}

	var err error
	var r = strings.NewReader(data)

	var tags wpk.TagSet
	if tags, err = pack.PackData(pack.w, r, fname, time.Now().Unix()); err != nil {
		ls.RaiseError(err.Error())
		return 0
	}

	if err = pack.adjusttagset(r, tags); err != nil {
		ls.RaiseError(err.Error())
		return 0
	}

	return 0
}

func datasize(ls *lua.LState) int {
	var pack = CheckPack(ls, 1)
	var fname = ls.CheckString(2)

	var rec, err = pack.NamedRecord(fname)
	if err != nil {
		ls.RaiseError(err.Error())
		return 0
	}

	ls.Push(lua.LNumber(rec.Size))
	return 1
}

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
		ls.RaiseError("file with name '%s' does not present", fname1)
		return 0
	}
	if _, ok = pack.Tags[key2]; ok {
		ls.RaiseError("file with name '%s' already present", fname2)
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
		ls.RaiseError("file with name '%s' does not present", fname1)
		return 0
	}
	if _, ok = pack.Tags[key2]; ok {
		ls.RaiseError("file with name '%s' already present", fname2)
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
