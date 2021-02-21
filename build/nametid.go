package main

import (
	"errors"
	"fmt"

	"github.com/schwarzlichtbezirk/wpk"
	lua "github.com/yuin/gopher-lua"
)

// NameTid helps convert Lua-table string keys to associated TID values.
var NameTid = map[string]wpk.TID{
	"fid":     wpk.TIDfid,
	"offset":  wpk.TIDoffset,
	"size":    wpk.TIDsize,
	"name":    wpk.TIDpath,
	"path":    wpk.TIDpath,
	"time":    wpk.TIDcreated,
	"created": wpk.TIDcreated,
	"crt":     wpk.TIDcreated,

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

	"mime":     wpk.TIDmime,
	"link":     wpk.TIDlink,
	"keywords": wpk.TIDkeywords,
	"category": wpk.TIDcategory,
	"version":  wpk.TIDversion,
	"author":   wpk.TIDauthor,
	"comment":  wpk.TIDcomment,
}

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

// ValueToAid converts LValue to uint16 tag identifier. Numbers converts explicitly,
// strings converts to uint16 values wich they presents.
// Error returns on any other case.
func ValueToAid(k lua.LValue) (tid wpk.TID, err error) {
	if n, ok := k.(lua.LNumber); ok {
		tid = wpk.TID(n)
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

// ValueToTag converts LValue to Tag. Strings converts explicitly to byte sequence,
// boolen converts to 1 byte slice with 1 for 'true' and 0 for 'false'.
// Otherwise if it is not 'tag' uservalue with Tag, returns error.
func ValueToTag(v lua.LValue) (tag wpk.Tag, err error) {
	if val, ok := v.(lua.LString); ok {
		tag = wpk.TagString(string(val))
	} else if val, ok := v.(lua.LBool); ok {
		tag = wpk.TagBool(bool(val))
	} else if ud, ok := v.(*lua.LUserData); ok {
		if val, ok := ud.Value.(*LuaTag); ok {
			tag = val.Tag
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

// TableToTagset converts Lua-table to Tagset. Lua-table keys can be number identifiers
// or string names associated ID values. Lua-table values can be strings,
// boolean or "tag" userdata values. Numbers can not be passed to table
// to prevent ambiguous type representation.
func TableToTagset(lt *lua.LTable) (ts wpk.Tagset, err error) {
	ts = wpk.Tagset{}
	lt.ForEach(func(k lua.LValue, v lua.LValue) {
		var (
			tid wpk.TID
			tag wpk.Tag
		)

		if tid, err = ValueToAid(k); err != nil {
			return
		}
		if tag, err = ValueToTag(v); err != nil {
			return
		}

		ts[tid] = tag
	})
	return
}

// The End.
