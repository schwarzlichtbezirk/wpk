package luawpk

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"strings"

	lua "github.com/yuin/gopher-lua"

	"github.com/schwarzlichtbezirk/wpk"
)

// Package writer errors.
var (
	ErrPackOpened = errors.New("package write stream already opened")
	ErrPackClosed = errors.New("package write stream does not opened")
	ErrDataClosed = errors.New("package data file is not opened")
)

// PackMT is "wpk" name of Lua metatable.
const PackMT = "wpk"

// LuaPackage is "wpk" userdata structure.
type LuaPackage struct {
	wpk.Package
	fidcount uint
	autofid  bool
	automime bool
	secret   []byte
	crc32    bool
	crc64    bool
	md5      bool
	sha1     bool
	sha224   bool
	sha256   bool
	sha384   bool
	sha512   bool

	pkgpath string
	datpath string
	wpt     wpk.WriteSeekCloser // package tags part
	wpf     wpk.WriteSeekCloser // package data part
}

// RegPack registers "wpk" userdata into Lua virtual machine.
func RegPack(ls *lua.LState) {
	var mt = ls.NewTypeMetatable(PackMT)
	ls.SetGlobal(PackMT, mt)
	// static attributes
	ls.SetField(mt, "new", ls.NewFunction(NewPack))
	// methods
	ls.SetField(mt, "__index", ls.NewFunction(getterPack))
	ls.SetField(mt, "__newindex", ls.NewFunction(setterPack))
	ls.SetField(mt, "__tostring", ls.NewFunction(tostringPack))
	for name, f := range methodsPack {
		ls.SetField(mt, name, ls.NewFunction(f))
	}
	for i, p := range propertiesPack {
		ls.SetField(mt, p.name, lua.LNumber(i))
	}
}

// PushPack push LuaPackage object into stack.
func PushPack(ls *lua.LState, v *LuaPackage) {
	var ud = ls.NewUserData()
	ud.Value = v
	ls.SetMetatable(ud, ls.GetTypeMetatable(PackMT))
	ls.Push(ud)
}

// NewPack is LuaPackage constructor.
func NewPack(ls *lua.LState) int {
	var pkg LuaPackage
	var ftt = &wpk.FTT{}
	ftt.Init(&wpk.Header{})
	pkg.FTT = ftt
	pkg.Workspace = "."
	PushPack(ls, &pkg)
	return 1
}

// CheckPack checks whether the lua argument with given number is
// a *LUserData with *LuaPackage and returns this *LuaPackage.
func CheckPack(ls *lua.LState, arg int) *LuaPackage {
	if v, ok := ls.CheckUserData(arg).Value.(*LuaPackage); ok {
		return v
	}
	ls.ArgError(arg, PackMT+" object required")
	return nil
}

func getterPack(ls *lua.LState) int {
	var mt = ls.GetMetatable(ls.Get(1))
	var val = ls.GetField(mt, ls.CheckString(2))
	switch val := val.(type) {
	case *lua.LFunction:
		ls.Push(val)
		return 1
	case lua.LNumber:
		var l = &propertiesPack[int(val)]
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

func setterPack(ls *lua.LState) int {
	var mt = ls.GetMetatable(ls.Get(1))
	var val = ls.GetField(mt, ls.CheckString(2))
	switch val := val.(type) {
	case *lua.LFunction:
		ls.Push(val)
		return 1
	case lua.LNumber:
		var l = &propertiesPack[int(val)]
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

func tostringPack(ls *lua.LState) int {
	var pkg = CheckPack(ls, 1)

	var m = map[uint]wpk.Void{}
	pkg.Enum(func(fkey string, ts wpk.TagsetRaw) bool {
		if offset, ok := ts.TagUint(wpk.TIDoffset); ok {
			m[offset] = wpk.Void{} // count unique offsets
		}
		return true
	})
	var items = []string{
		fmt.Sprintf("files: %d", len(m)),
		fmt.Sprintf("aliases: %d", pkg.TagsetNum()-len(m)),
		fmt.Sprintf("datasize: %d", pkg.DataSize()),
	}
	if str, ok := pkg.GetInfo().TagStr(wpk.TIDlabel); ok {
		items = append(items, fmt.Sprintf("label: %s", str))
	}
	if str, ok := pkg.GetInfo().TagStr(wpk.TIDlink); ok {
		items = append(items, fmt.Sprintf("link: %s", str))
	}
	if str, ok := pkg.GetInfo().TagStr(wpk.TIDversion); ok {
		items = append(items, fmt.Sprintf("version: %s", str))
	}
	if str, ok := pkg.GetInfo().TagStr(wpk.TIDauthor); ok {
		items = append(items, fmt.Sprintf("author: %s", str))
	}
	if str, ok := pkg.GetInfo().TagStr(wpk.TIDcomment); ok {
		items = append(items, fmt.Sprintf("comment: %s", str))
	}
	var s = strings.Join(items, ", ")

	ls.Push(lua.LString(s))
	return 1
}

var propertiesPack = []struct {
	name   string
	getter lua.LGFunction // getters always must return 1 value
	setter lua.LGFunction // setters always must return no values
}{
	{"label", getlabel, setlabel},
	{"pkgpath", getpkgpath, nil},
	{"datpath", getdatpath, nil},
	{"recnum", getrecnum, nil},
	{"tagnum", gettagnum, nil},
	{"fftsize", getfftsize, nil},
	{"datasize", getdatasize, nil},
	{"autofid", getautofid, setautofid},
	{"automime", getautomime, setautomime},
	{"secret", getsecret, setsecret},
	{"crc32", getcrc32, setcrc32},
	{"crc64", getcrc64, setcrc64},
	{"md5", getmd5, setmd5},
	{"sha1", getsha1, setsha1},
	{"sha224", getsha224, setsha224},
	{"sha256", getsha256, setsha256},
	{"sha384", getsha384, setsha384},
	{"sha512", getsha512, setsha512},
}

var methodsPack = map[string]lua.LGFunction{
	"load":      wpkload,
	"begin":     wpkbegin,
	"append":    wpkappend,
	"finalize":  wpkfinalize,
	"flush":     wpkflush,
	"sumsize":   wpksumsize,
	"glob":      wpkglob,
	"hasfile":   wpkhasfile,
	"filesize":  wpkfilesize,
	"putdata":   wpkputdata,
	"putfile":   wpkputfile,
	"rename":    wpkrename,
	"renamedir": wpkrenamedir,
	"putalias":  wpkputalias,
	"delalias":  wpkdelalias,
	"hastag":    wpkhastag,
	"gettag":    wpkgettag,
	"settag":    wpksettag,
	"addtag":    wpkaddtag,
	"deltag":    wpkdeltag,
	"gettags":   wpkgettags,
	"settags":   wpksettags,
	"addtags":   wpkaddtags,
	"deltags":   wpkdeltags,
	"getinfo":   wpkgetinfo,
	"setupinfo": wpksetupinfo,
}

// properties section

func getlabel(ls *lua.LState) int {
	var pkg = CheckPack(ls, 1)

	if str, ok := pkg.GetInfo().TagStr(wpk.TIDlabel); ok {
		ls.Push(lua.LString(str))
		return 1
	}
	ls.Push(lua.LNil)
	return 1
}

func setlabel(ls *lua.LState) int {
	var pkg = CheckPack(ls, 1)
	var label = ls.CheckString(2)

	pkg.SetInfo(wpk.CopyTagset(pkg.GetInfo()).
		Set(wpk.TIDlabel, wpk.StrTag(label)))
	return 0
}

func getpkgpath(ls *lua.LState) int {
	var pkg = CheckPack(ls, 1)
	if pkg.wpt == nil {
		ls.Push(lua.LNil)
		return 1
	}
	ls.Push(lua.LString(pkg.pkgpath))
	return 1
}

func getdatpath(ls *lua.LState) int {
	var pkg = CheckPack(ls, 1)
	if pkg.wpf == nil {
		ls.Push(lua.LNil)
		return 1
	}
	ls.Push(lua.LString(pkg.datpath))
	return 1
}

func getrecnum(ls *lua.LState) int {
	var pkg = CheckPack(ls, 1)
	var m = map[uint]wpk.Void{}
	pkg.Enum(func(fkey string, ts wpk.TagsetRaw) bool {
		if offset, ok := ts.TagUint(wpk.TIDoffset); ok {
			m[offset] = wpk.Void{}
		}
		return true
	})
	ls.Push(lua.LNumber(len(m)))
	return 1
}

func gettagnum(ls *lua.LState) int {
	var pkg = CheckPack(ls, 1)
	ls.Push(lua.LNumber(pkg.TagsetNum()))
	return 1
}

func getfftsize(ls *lua.LState) int {
	var pkg = CheckPack(ls, 1)
	var size int
	pkg.Enum(func(fkey string, ts wpk.TagsetRaw) bool {
		size += len(ts) + wpk.PTStssize
		return true
	})
	ls.Push(lua.LNumber(size))
	return 1
}

func getdatasize(ls *lua.LState) int {
	var pkg = CheckPack(ls, 1)
	ls.Push(lua.LNumber(pkg.DataSize()))
	return 1
}

func getautofid(ls *lua.LState) int {
	var pkg = CheckPack(ls, 1)
	ls.Push(lua.LBool(pkg.autofid))
	return 1
}

func setautofid(ls *lua.LState) int {
	var pkg = CheckPack(ls, 1)
	var val = ls.CheckBool(2)

	pkg.autofid = val
	return 0
}

func getautomime(ls *lua.LState) int {
	var pkg = CheckPack(ls, 1)
	ls.Push(lua.LBool(pkg.automime))
	return 1
}

func setautomime(ls *lua.LState) int {
	var pkg = CheckPack(ls, 1)
	var val = ls.CheckBool(2)

	pkg.automime = val
	return 0
}

func getsecret(ls *lua.LState) int {
	var pkg = CheckPack(ls, 1)
	ls.Push(lua.LString(pkg.secret))
	return 1
}

func setsecret(ls *lua.LState) int {
	var pkg = CheckPack(ls, 1)
	var val = ls.CheckString(2)

	pkg.secret = []byte(val)
	return 0
}

func getcrc32(ls *lua.LState) int {
	var pkg = CheckPack(ls, 1)
	ls.Push(lua.LBool(pkg.crc32))
	return 1
}

func setcrc32(ls *lua.LState) int {
	var pkg = CheckPack(ls, 1)
	var val = ls.CheckBool(2)

	pkg.crc32 = val
	return 0
}

func getcrc64(ls *lua.LState) int {
	var pkg = CheckPack(ls, 1)
	ls.Push(lua.LBool(pkg.crc64))
	return 1
}

func setcrc64(ls *lua.LState) int {
	var pkg = CheckPack(ls, 1)
	var val = ls.CheckBool(2)

	pkg.crc64 = val
	return 0
}

func getmd5(ls *lua.LState) int {
	var pkg = CheckPack(ls, 1)
	ls.Push(lua.LBool(pkg.md5))
	return 1
}

func setmd5(ls *lua.LState) int {
	var pkg = CheckPack(ls, 1)
	var val = ls.CheckBool(2)

	pkg.md5 = val
	return 0
}

func getsha1(ls *lua.LState) int {
	var pkg = CheckPack(ls, 1)
	ls.Push(lua.LBool(pkg.sha1))
	return 1
}

func setsha1(ls *lua.LState) int {
	var pkg = CheckPack(ls, 1)
	var val = ls.CheckBool(2)

	pkg.sha1 = val
	return 0
}

func getsha224(ls *lua.LState) int {
	var pkg = CheckPack(ls, 1)
	ls.Push(lua.LBool(pkg.sha224))
	return 1
}

func setsha224(ls *lua.LState) int {
	var pkg = CheckPack(ls, 1)
	var val = ls.CheckBool(2)

	pkg.sha224 = val
	return 0
}

func getsha256(ls *lua.LState) int {
	var pkg = CheckPack(ls, 1)
	ls.Push(lua.LBool(pkg.sha256))
	return 1
}

func setsha256(ls *lua.LState) int {
	var pkg = CheckPack(ls, 1)
	var val = ls.CheckBool(2)

	pkg.sha256 = val
	return 0
}

func getsha384(ls *lua.LState) int {
	var pkg = CheckPack(ls, 1)
	ls.Push(lua.LBool(pkg.sha384))
	return 1
}

func setsha384(ls *lua.LState) int {
	var pkg = CheckPack(ls, 1)
	var val = ls.CheckBool(2)

	pkg.sha384 = val
	return 0
}

func getsha512(ls *lua.LState) int {
	var pkg = CheckPack(ls, 1)
	ls.Push(lua.LBool(pkg.sha512))
	return 1
}

func setsha512(ls *lua.LState) int {
	var pkg = CheckPack(ls, 1)
	var val = ls.CheckBool(2)

	pkg.sha512 = val
	return 0
}

// methods section

func wpkload(ls *lua.LState) int {
	var err error
	defer func() {
		if err != nil {
			ls.RaiseError(err.Error())
		}
	}()
	var pkg = CheckPack(ls, 1)
	var pkgpath = ls.CheckString(2)
	var datpath = ls.OptString(3, "")

	if pkg.wpt != nil {
		err = ErrPackOpened
		return 0
	}

	// open package file
	if err = pkg.OpenFile(pkgpath); err != nil {
		return 0
	}
	pkg.pkgpath, pkg.datpath = pkgpath, datpath

	return 0
}

func wpkbegin(ls *lua.LState) int {
	var err error
	defer func() {
		if err != nil {
			ls.RaiseError(err.Error())
		}
	}()
	var pkg = CheckPack(ls, 1)
	var pkgpath = ls.CheckString(2)
	var datpath = ls.OptString(3, "")

	if pkg.wpt != nil {
		err = ErrPackOpened
		return 0
	}

	// create package file
	if pkg.wpt, err = os.OpenFile(pkgpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0755); err != nil {
		return 0
	}
	if datpath != "" {
		if pkg.wpf, err = os.OpenFile(datpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0755); err != nil {
			pkg.wpt.Close()
			pkg.wpt = nil
			return 0
		}
	}
	// setup file representation
	pkg.pkgpath, pkg.datpath = pkgpath, datpath
	// starts new package
	if err = pkg.Begin(pkg.wpt, pkg.wpf); err != nil {
		return 0
	}

	return 0
}

func wpkappend(ls *lua.LState) int {
	var err error
	defer func() {
		if err != nil {
			ls.RaiseError(err.Error())
		}
	}()
	var pkg = CheckPack(ls, 1)

	if pkg.wpt != nil {
		err = ErrPackOpened
		return 0
	}

	// open package file
	if pkg.wpt, err = os.OpenFile(pkg.pkgpath, os.O_WRONLY, 0755); err != nil {
		return 0
	}
	if pkg.datpath != "" {
		if pkg.wpf, err = os.OpenFile(pkg.datpath, os.O_WRONLY, 0755); err != nil {
			pkg.wpt.Close()
			pkg.wpt = nil
			return 0
		}
	}
	// starts to append files
	if err = pkg.Append(pkg.wpt, pkg.wpf); err != nil {
		return 0
	}

	return 0
}

func wpkfinalize(ls *lua.LState) int {
	var err error
	defer func() {
		if err != nil {
			ls.RaiseError(err.Error())
		}
	}()
	var pkg = CheckPack(ls, 1)

	if pkg.wpt == nil {
		err = ErrPackClosed
		return 0
	}

	// sync
	if err = pkg.Sync(pkg.wpt, pkg.wpf); err != nil {
		return 0
	}
	// close package file
	if err = pkg.wpt.Close(); err != nil {
		return 0
	}
	pkg.wpt = nil
	if pkg.wpf != nil {
		if err = pkg.wpf.Close(); err != nil {
			return 0
		}
		pkg.wpf = nil
	}

	return 0
}

func wpkflush(ls *lua.LState) int {
	var err error
	defer func() {
		if err != nil {
			ls.RaiseError(err.Error())
		}
	}()
	var pkg = CheckPack(ls, 1)

	if pkg.wpt == nil {
		err = ErrPackClosed
		return 0
	}
	if pkg.wpf == nil { // can work only for splitted package
		err = ErrDataClosed
		return 0
	}

	// sync
	if err = pkg.Sync(pkg.wpt, pkg.wpf); err != nil {
		return 0
	}

	return 0
}

func wpksumsize(ls *lua.LState) int {
	var pkg = CheckPack(ls, 1)

	var sum int64
	pkg.Enum(func(fkey string, ts wpk.TagsetRaw) bool {
		var size = ts.Size()
		sum += size
		return true
	})

	ls.Push(lua.LNumber(sum))
	return 1
}

func wpkglob(ls *lua.LState) int {
	var err error
	defer func() {
		if err != nil {
			ls.RaiseError(err.Error())
		}
	}()
	var pkg = CheckPack(ls, 1)
	var pattern = ls.CheckString(2)

	var n int
	var matched bool
	if _, err = path.Match(pattern, ""); err != nil {
		return 0
	}
	pkg.Enum(func(fkey string, ts wpk.TagsetRaw) bool {
		if matched, _ = path.Match(pattern, fkey); matched {
			ls.Push(lua.LString(fkey))
			n++
		}
		return true
	})
	return n
}

func wpkhasfile(ls *lua.LState) int {
	var pkg = CheckPack(ls, 1)
	var fkey = ls.CheckString(2)

	var ok = pkg.HasTagset(fkey)

	ls.Push(lua.LBool(ok))
	return 1
}

func wpkfilesize(ls *lua.LState) int {
	var err error
	defer func() {
		if err != nil {
			ls.RaiseError(err.Error())
		}
	}()
	var pkg = CheckPack(ls, 1)
	var fkey = ls.CheckString(2)

	var ts wpk.TagsetRaw
	var ok bool
	if ts, ok = pkg.GetTagset(fkey); !ok {
		err = &fs.PathError{Op: "filesize", Path: fkey, Err: fs.ErrNotExist}
		return 0
	}

	var size = ts.Size()
	ls.Push(lua.LNumber(size))
	return 1
}

func wpkputdata(ls *lua.LState) int {
	var err error
	defer func() {
		if err != nil {
			ls.RaiseError(err.Error())
		}
	}()
	var pkg = CheckPack(ls, 1)
	var fkey = ls.CheckString(2)
	var data = ls.CheckString(3)
	var tags = ls.OptTable(4, ls.CreateTable(0, 0))

	if pkg.wpt == nil {
		err = ErrPackClosed
		return 0
	}

	var r = strings.NewReader(data)

	var w = pkg.wpt
	if pkg.wpf != nil {
		w = pkg.wpf
	}
	var ts wpk.TagsetRaw
	if ts, err = pkg.PackData(w, r, fkey); err != nil {
		return 0
	}

	if ts, err = pkg.adjusttagset(r, ts); err != nil {
		return 0
	}

	if ts, err = TableToTagset(tags, ts); err != nil {
		return 0
	}

	pkg.SetupTagset(ts)

	return 0
}

func wpkputfile(ls *lua.LState) int {
	var err error
	defer func() {
		if err != nil {
			ls.RaiseError(err.Error())
		}
	}()
	var pkg = CheckPack(ls, 1)
	var fkey = ls.CheckString(2)
	var fpath = ls.CheckString(3)
	var tags = ls.OptTable(4, ls.CreateTable(0, 0))

	if pkg.wpt == nil {
		err = ErrPackClosed
		return 0
	}

	var file wpk.RFile
	if file, err = os.Open(fpath); err != nil {
		return 0
	}
	defer file.Close()

	var w = pkg.wpt
	if pkg.wpf != nil {
		w = pkg.wpf
	}
	var ts wpk.TagsetRaw
	if ts, err = pkg.PackFile(w, file, fkey); err != nil {
		return 0
	}

	if ts, err = pkg.adjusttagset(file, ts); err != nil {
		return 0
	}

	if ts, err = TableToTagset(tags, ts); err != nil {
		return 0
	}

	pkg.SetupTagset(ts)

	return 0
}

// Renames tagset with file name fkey1 to fkey2.
// rename(fkey1, fkey2)
//
//	fkey1 - old file name
//	fkey2 - new file name
func wpkrename(ls *lua.LState) int {
	var err error
	defer func() {
		if err != nil {
			ls.RaiseError(err.Error())
		}
	}()
	var pkg = CheckPack(ls, 1)
	var fkey1 = ls.CheckString(2)
	var fkey2 = ls.CheckString(3)

	if err = pkg.Rename(fkey1, fkey2); err != nil {
		return 0
	}
	return 0
}

// Renames all files in package with 'dir1' path to 'dir2' path.
// rename(dir1, dir2)
//
//	dir1 - old directory
//	dir2 - new directory
//	skipexist - 'true' to skip files with existing new names
func wpkrenamedir(ls *lua.LState) int {
	var err error
	defer func() {
		if err != nil {
			ls.RaiseError(err.Error())
		}
	}()
	var pkg = CheckPack(ls, 1)
	var dir1 = ls.CheckString(2)
	var dir2 = ls.CheckString(3)
	var skipexist = ls.OptBool(4, true)

	var count int
	if count, err = pkg.RenameDir(dir1, dir2, skipexist); err != nil {
		return 0
	}
	ls.Push(lua.LNumber(count))
	return 1
}

// Creates copy of tagset with new file name.
// putalias(fkey1, fkey2)
//
//	fkey1 - file name of packaged file
//	fkey2 - new file name that will be reference to fkey1 file data
func wpkputalias(ls *lua.LState) int {
	var err error
	defer func() {
		if err != nil {
			ls.RaiseError(err.Error())
		}
	}()
	var pkg = CheckPack(ls, 1)
	var fkey1 = ls.CheckString(2)
	var fkey2 = ls.CheckString(3)

	if err = pkg.PutAlias(fkey1, fkey2); err != nil {
		return 0
	}
	return 0
}

// Deletes tagset with given file name. Data block will still persist.
func wpkdelalias(ls *lua.LState) int {
	var pkg = CheckPack(ls, 1)
	var fkey = ls.CheckString(2)

	var _, ok = pkg.DelTagset(fkey)
	ls.Push(lua.LBool(ok))
	return 1
}

// Returns true if tags for given file name have tag with given identifier
// (in numeric or string representation).
func wpkhastag(ls *lua.LState) int {
	var err error
	defer func() {
		if err != nil {
			ls.RaiseError(err.Error())
		}
	}()
	var pkg = CheckPack(ls, 1)
	var fkey = ls.CheckString(2)
	var k = ls.Get(3)

	var tid wpk.TID
	if tid, err = ValueToTID(k); err != nil {
		return 0
	}

	var ts wpk.TagsetRaw
	var ok bool
	if ts, ok = pkg.GetTagset(fkey); !ok {
		err = &fs.PathError{Op: "hastag", Path: fkey, Err: fs.ErrNotExist}
		return 0
	}
	ok = ts.Has(tid)

	ls.Push(lua.LBool(ok))
	return 1
}

// Returns single tag with specified identifier from tagset of given file.
func wpkgettag(ls *lua.LState) int {
	var err error
	defer func() {
		if err != nil {
			ls.RaiseError(err.Error())
		}
	}()
	var pkg = CheckPack(ls, 1)
	var fkey = ls.CheckString(2)
	var k = ls.Get(3)

	var tid wpk.TID
	if tid, err = ValueToTID(k); err != nil {
		return 0
	}

	var ts wpk.TagsetRaw
	var ok bool
	if ts, ok = pkg.GetTagset(fkey); !ok {
		err = &fs.PathError{Op: "gettag", Path: fkey, Err: fs.ErrNotExist}
		return 0
	}

	var tag wpk.TagRaw
	if tag, ok = ts.Get(tid); !ok {
		return 0
	}

	var val lua.LValue
	if val, err = TagToValue(tid, tag); err != nil {
		return 0
	}
	ls.Push(val)
	return 1
}

// Set tag with given identifier to tagset of specified file.
func wpksettag(ls *lua.LState) int {
	var err error
	defer func() {
		if err != nil {
			ls.RaiseError(err.Error())
		}
	}()
	var pkg = CheckPack(ls, 1)
	var fkey = ls.CheckString(2)
	var k = ls.Get(3)
	var v = ls.Get(4)

	var tid wpk.TID
	if tid, err = ValueToTID(k); err != nil {
		return 0
	}
	if tid == wpk.TIDoffset || tid == wpk.TIDsize || tid == wpk.TIDpath {
		err = &ErrProtected{tid}
		return 0
	}

	var tag wpk.TagRaw
	if tag, err = ValueToTag(tid, v); err != nil {
		return 0
	}

	var ts, ok = pkg.GetTagset(fkey)
	if !ok {
		err = &fs.PathError{Op: "settag", Path: fkey, Err: fs.ErrNotExist}
		return 0
	}
	ts = wpk.CopyTagset(ts)
	ts, ok = ts.SetOk(tid, tag)
	pkg.SetupTagset(ts)

	ls.Push(lua.LBool(ok))
	return 1
}

// Add tag with given identifier to tagset of specified file.
func wpkaddtag(ls *lua.LState) int {
	var err error
	defer func() {
		if err != nil {
			ls.RaiseError(err.Error())
		}
	}()
	var pkg = CheckPack(ls, 1)
	var fkey = ls.CheckString(2)
	var k = ls.Get(3)
	var v = ls.Get(4)

	var tid wpk.TID
	if tid, err = ValueToTID(k); err != nil {
		return 0
	}
	if tid == wpk.TIDoffset || tid == wpk.TIDsize || tid == wpk.TIDpath {
		err = &ErrProtected{tid}
		return 0
	}

	var tag wpk.TagRaw
	if tag, err = ValueToTag(tid, v); err != nil {
		return 0
	}

	var ts, ok = pkg.GetTagset(fkey)
	if !ok {
		err = &fs.PathError{Op: "addtag", Path: fkey, Err: fs.ErrNotExist}
		return 0
	}
	ts = wpk.CopyTagset(ts)
	if ts, ok = ts.AddOk(tid, tag); ok {
		pkg.SetupTagset(ts)
	}

	ls.Push(lua.LBool(ok))
	return 1
}

// Delete tag with given identifier from tagset of specified file.
func wpkdeltag(ls *lua.LState) int {
	var err error
	defer func() {
		if err != nil {
			ls.RaiseError(err.Error())
		}
	}()
	var pkg = CheckPack(ls, 1)
	var fkey = ls.CheckString(2)
	var k = ls.Get(3)

	var tid wpk.TID
	if tid, err = ValueToTID(k); err != nil {
		return 0
	}
	if tid == wpk.TIDoffset || tid == wpk.TIDsize || tid == wpk.TIDpath {
		err = &ErrProtected{tid}
		return 0
	}

	var ts, ok = pkg.GetTagset(fkey)
	if !ok {
		err = &fs.PathError{Op: "deltag", Path: fkey, Err: fs.ErrNotExist}
		return 0
	}
	ts = wpk.CopyTagset(ts)
	if ts, ok = ts.DelOk(tid); ok {
		pkg.SetupTagset(ts)
	}

	ls.Push(lua.LBool(ok))
	return 1
}

func wpkgettags(ls *lua.LState) int {
	var err error
	defer func() {
		if err != nil {
			ls.RaiseError(err.Error())
		}
	}()
	var pkg = CheckPack(ls, 1)
	var fkey = ls.CheckString(2)

	var ts, ok = pkg.GetTagset(fkey)
	if !ok {
		err = &fs.PathError{Op: "gettags", Path: fkey, Err: fs.ErrNotExist}
		return 0
	}

	var tb = ls.CreateTable(0, 0)
	var tsi = ts.Iterator()
	for tsi.Next() {
		var tid, tag = tsi.TID(), tsi.Tag()
		var val lua.LValue
		if val, err = TagToValue(tid, tag); err != nil {
			return 0
		}
		if name, ok := TidName[tid]; ok {
			tb.RawSet(lua.LString(name), val)
		} else {
			tb.RawSet(lua.LNumber(tid), val)
		}
	}
	ls.Push(tb)
	return 1
}

// Sets or replaces tags for given file with new tags values.
func wpksettags(ls *lua.LState) int {
	var err error
	defer func() {
		if err != nil {
			ls.RaiseError(err.Error())
		}
	}()
	var pkg = CheckPack(ls, 1)
	var fkey = ls.CheckString(2)
	var lt = ls.CheckTable(3)

	var opts wpk.TagsetRaw
	if opts, err = TableToTagset(lt, opts); err != nil {
		return 0
	}

	var ts, ok = pkg.GetTagset(fkey)
	if !ok {
		err = &fs.PathError{Op: "settags", Path: fkey, Err: fs.ErrNotExist}
		return 0
	}

	ts = wpk.CopyTagset(ts)
	var optsi = opts.Iterator()
	for optsi.Next() {
		ts = ts.Set(optsi.TID(), optsi.Tag())
	}
	pkg.SetupTagset(ts)

	return 0
}

// Adds new tags for given file if there is no old values.
// Returns number of added tags.
func wpkaddtags(ls *lua.LState) int {
	var err error
	defer func() {
		if err != nil {
			ls.RaiseError(err.Error())
		}
	}()
	var pkg = CheckPack(ls, 1)
	var fkey = ls.CheckString(2)
	var lt = ls.CheckTable(3)

	var opts wpk.TagsetRaw
	if opts, err = TableToTagset(lt, opts); err != nil {
		return 0
	}

	var ts, ok = pkg.GetTagset(fkey)
	if !ok {
		err = &fs.PathError{Op: "addtags", Path: fkey, Err: fs.ErrNotExist}
		return 0
	}

	ts = wpk.CopyTagset(ts)
	var optsi = opts.Iterator()
	var n = 0
	for optsi.Next() {
		if ts, ok = ts.AddOk(optsi.TID(), optsi.Tag()); ok {
			n++
		}
	}
	if n > 0 {
		pkg.SetupTagset(ts)
	}

	ls.Push(lua.LNumber(n))
	return 1
}

// Removes tags with given identifiers for given file. Specified values of
// tags table ignored. Returns number of deleted tags.
func wpkdeltags(ls *lua.LState) int {
	var err error
	defer func() {
		if err != nil {
			ls.RaiseError(err.Error())
		}
	}()
	var pkg = CheckPack(ls, 1)
	var fkey = ls.CheckString(2)
	var lt = ls.CheckTable(3)

	var opts wpk.TagsetRaw
	if opts, err = TableToTagset(lt, opts); err != nil {
		return 0
	}

	var ts, ok = pkg.GetTagset(fkey)
	if !ok {
		err = &fs.PathError{Op: "deltags", Path: fkey, Err: fs.ErrNotExist}
		return 0
	}

	ts = wpk.CopyTagset(ts)
	var optsi = opts.Iterator()
	var n = 0
	for optsi.Next() {
		if ts, ok = ts.DelOk(optsi.TID()); ok {
			n++
		}
	}
	if n > 0 {
		pkg.SetupTagset(ts)
	}

	ls.Push(lua.LNumber(n))
	return 1
}

func wpkgetinfo(ls *lua.LState) int {
	var err error
	defer func() {
		if err != nil {
			ls.RaiseError(err.Error())
		}
	}()
	var pkg = CheckPack(ls, 1)

	var tb = ls.CreateTable(0, 0)
	var tsi = pkg.GetInfo().Iterator()
	for tsi.Next() {
		var tid, tag = tsi.TID(), tsi.Tag()
		if tid == wpk.TIDfid || tid == wpk.TIDpath {
			continue
		}
		var val lua.LValue
		if val, err = TagToValue(tid, tag); err != nil {
			return 0
		}
		if name, ok := TidName[tid]; ok {
			tb.RawSet(lua.LString(name), val)
		} else {
			tb.RawSet(lua.LNumber(tid), val)
		}
	}
	ls.Push(tb)
	return 1
}

func wpksetupinfo(ls *lua.LState) int {
	var err error
	defer func() {
		if err != nil {
			ls.RaiseError(err.Error())
		}
	}()
	var pkg = CheckPack(ls, 1)
	var lt = ls.CheckTable(2)

	var opts wpk.TagsetRaw
	if opts, err = TableToTagset(lt, opts); err != nil {
		return 0
	}
	pkg.SetInfo(opts)
	return 0
}

// The End.
