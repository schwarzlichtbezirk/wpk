package luawpk

import (
	"errors"
	"fmt"

	"github.com/schwarzlichtbezirk/wpk"
	lua "github.com/yuin/gopher-lua"
)

// NameTid helps convert Lua-table string keys to associated uint values.
var NameTid = map[string]uint{
	"offset": wpk.TIDoffset,
	"size":   wpk.TIDsize,
	"fid":    wpk.TIDfid,
	"path":   wpk.TIDpath,
	"mtime":  wpk.TIDmtime,
	"atime":  wpk.TIDatime,
	"ctime":  wpk.TIDctime,
	"btime":  wpk.TIDbtime,
	"attr":   wpk.TIDattr,
	"mime":   wpk.TIDmime,

	"crc32":     wpk.TIDcrc32c,
	"crc32ieee": wpk.TIDcrc32ieee,
	"crc32c":    wpk.TIDcrc32c,
	"crc32k":    wpk.TIDcrc32k,
	"crc64":     wpk.TIDcrc64iso,
	"crc64iso":  wpk.TIDcrc64iso,

	"md5":    wpk.TIDmd5,
	"sha1":   wpk.TIDsha1,
	"sha224": wpk.TIDsha224,
	"sha256": wpk.TIDsha256,
	"sha384": wpk.TIDsha384,
	"sha512": wpk.TIDsha512,

	"tmbimg":   wpk.TIDtmbimg,
	"tmbmime":  wpk.TIDtmbmime,
	"label":    wpk.TIDlabel,
	"link":     wpk.TIDlink,
	"keywords": wpk.TIDkeywords,
	"category": wpk.TIDcategory,
	"version":  wpk.TIDversion,
	"author":   wpk.TIDauthor,
	"comment":  wpk.TIDcomment,
}

// TidName helps format Lua-tables with string keys associated to uint values.
var TidName = func() map[uint]string {
	var tn = map[uint]string{}
	for name, tid := range NameTid {
		tn[tid] = name
	}
	return tn
}()

// ErrKeyUndef represents error on tag identifiers string presentation.
type ErrKeyUndef struct {
	TagKey string
}

func (e *ErrKeyUndef) Error() string {
	return fmt.Sprintf("tag key '%s' is undefined", e.TagKey)
}

// Tags identifiers conversion errors.
var (
	ErrBadTagKey = errors.New("tag key type is not number or string")
	ErrBadTagVal = errors.New("tag value type is not string or boolean or 'tag' userdata")
)

// ValueToTID converts LValue to uint16 tag identifier.
// Numbers converts explicitly, strings converts to uint16
// values which they presents. Error returns on any other case.
func ValueToTID(k lua.LValue) (tid uint, err error) {
	if n, ok := k.(lua.LNumber); ok {
		tid = uint(n)
	} else if name, ok := k.(lua.LString); ok {
		if n, ok := NameTid[string(name)]; ok {
			tid = n
		} else {
			err = &ErrKeyUndef{string(name)}
			return
		}
	} else {
		err = ErrBadTagKey
		return
	}
	return
}

// ValueToTag converts LValue to Tag_t. Strings converts explicitly to byte sequence,
// boolen converts to 1 byte slice with 1 for 'true' and 0 for 'false'.
// Otherwise if it is not 'tag' uservalue with Tag_t, returns error.
func ValueToTag(v lua.LValue) (tag wpk.Tag_t, err error) {
	if val, ok := v.(lua.LNumber); ok {
		var u = uint(val)
		if val < 0 || val > lua.LNumber(uint(1<<64-1)) || val-lua.LNumber(u) != 0 {
			tag = wpk.NumberTag(float64(val))
		} else {
			tag = wpk.UintTag(u)
		}
	} else if val, ok := v.(lua.LString); ok {
		tag = wpk.StrTag(string(val))
	} else if val, ok := v.(lua.LBool); ok {
		tag = wpk.BoolTag(bool(val))
	} else if ud, ok := v.(*lua.LUserData); ok {
		if val, ok := ud.Value.(*LuaTag); ok {
			tag = val.Tag_t
		} else {
			err = ErrBadTagVal
			return
		}
	} else {
		err = ErrBadTagVal
		return
	}
	return
}

// TableToTagset converts Lua-table to Tagset_t. Lua-table keys can be number identifiers
// or string names associated ID values. Lua-table values can be strings,
// boolean or "tag" userdata values. Numbers can not be passed to table
// to prevent ambiguous type representation.
func TableToTagset(lt *lua.LTable, ts *wpk.Tagset_t) (err error) {
	lt.ForEach(func(k lua.LValue, v lua.LValue) {
		var (
			tid uint
			tag wpk.Tag_t
		)

		if tid, err = ValueToTID(k); err != nil {
			return
		}
		if tag, err = ValueToTag(v); err != nil {
			return
		}

		ts.Put(tid, tag)
	})
	return
}

// The End.
