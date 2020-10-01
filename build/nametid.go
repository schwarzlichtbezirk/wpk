package main

import (
	"errors"
	"fmt"

	. "github.com/schwarzlichtbezirk/wpk"
	"github.com/yuin/gopher-lua"
)

// Helps convert Lua-table string keys to associated TID values.
var NameTid = map[string]TID{
	"fid":     TID_FID,
	"offset":  TID_offset,
	"size":    TID_size,
	"name":    TID_path,
	"path":    TID_path,
	"time":    TID_created,
	"created": TID_created,
	"crt":     TID_created,

	"crc32":     TID_CRC32C,
	"crc32ieee": TID_CRC32IEEE,
	"crc32c":    TID_CRC32C,
	"crc32k":    TID_CRC32K,
	"crc64":     TID_CRC64ISO,
	"crc64iso":  TID_CRC64ISO,

	"md5":    TID_MD5,
	"sha1":   TID_SHA1,
	"sha224": TID_SHA224,
	"sha256": TID_SHA256,
	"sha384": TID_SHA384,
	"sha512": TID_SHA512,

	"mime":     TID_mime,
	"link":     TID_link,
	"keywords": TID_keywords,
	"category": TID_category,
	"version":  TID_version,
	"author":   TID_author,
	"comment":  TID_comment,
}

// Errors on tag identifiers string presentation.
type ErrKeyUndef struct {
	TagKey string
}

func (e *ErrKeyUndef) Error() string {
	return fmt.Sprintf("tag key '%s' is undefined", e.TagKey)
}

var (
	ErrBadTagKey = errors.New("tag key type is not number or string")
	ErrBadTagVal = errors.New("tag value type is not string or boolean or 'tag' userdata")
)

// Convert LValue to uint16 tag identifier. Numbers converts explicitly,
// strings converts to uint16 values wich they presents.
// Error returns on any other case.
func ValueToAid(k lua.LValue) (tid TID, err error) {
	if n, ok := k.(lua.LNumber); ok {
		tid = TID(n)
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

// Convert LValue to Tag. Strings converts explicitly to byte sequence,
// boolen converts to 1 byte slice with 1 for 'true' and 0 for 'false'.
// Otherwise if it is not 'tag' uservalue with Tag, returns error.
func ValueToTag(v lua.LValue) (tag Tag, err error) {
	if val, ok := v.(lua.LString); ok {
		tag = TagString(string(val))
	} else if val, ok := v.(lua.LBool); ok {
		tag = TagBool(bool(val))
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

// Convert Lua-table to Tagset. Lua-table keys can be number identifiers
// or string names associated ID values. Lua-table values can be strings,
// boolean or "tag" userdata values. Numbers can not be passed to table
// to prevent ambiguous type representation.
func TableToTagset(lt *lua.LTable) (ts Tagset, err error) {
	ts = Tagset{}
	lt.ForEach(func(k lua.LValue, v lua.LValue) {
		var (
			tid TID
			tag Tag
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
