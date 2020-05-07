package main

import (
	"errors"
	"fmt"

	"github.com/schwarzlichtbezirk/wpk"
	"github.com/yuin/gopher-lua"
)

// Helps convert Lua-table string keys to associated uint16 ID values.
var NameAid = map[string]uint16{
	"fid":     wpk.AID_FID,
	"name":    wpk.AID_name,
	"created": wpk.AID_created,
	"crt":     wpk.AID_created,

	"crc32":     wpk.AID_CRC32C,
	"crc32ieee": wpk.AID_CRC32IEEE,
	"crc32c":    wpk.AID_CRC32C,
	"crc32k":    wpk.AID_CRC32K,
	"crc64":     wpk.AID_CRC64ISO,
	"crc64iso":  wpk.AID_CRC64ISO,

	"md5":    wpk.AID_MD5,
	"sha1":   wpk.AID_SHA1,
	"sha224": wpk.AID_SHA224,
	"sha256": wpk.AID_SHA256,
	"sha384": wpk.AID_SHA384,
	"sha512": wpk.AID_SHA512,

	"mime":     wpk.AID_mime,
	"keywords": wpk.AID_keywords,
	"category": wpk.AID_category,
	"version":  wpk.AID_version,
	"author":   wpk.AID_author,
	"comment":  wpk.AID_comment,
}

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
func ValueToAid(k lua.LValue) (aid uint16, err error) {
	if n, ok := k.(lua.LNumber); ok {
		aid = uint16(n)
	} else if name, ok := k.(lua.LString); ok {
		if n, ok := NameAid[string(name)]; ok {
			aid = n
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

// Convert Lua-table to Tagset. Lua-table keys can be number identifiers
// or string names associated ID values. Lua-table values can be strings,
// boolean or "tag" userdata values. Numbers can not be passed to table
// to prevent ambiguous type representation.
func TableToTagset(lt *lua.LTable) (ts wpk.Tagset, err error) {
	ts = wpk.Tagset{}
	lt.ForEach(func(k lua.LValue, v lua.LValue) {
		var (
			aid uint16
			tag wpk.Tag
		)

		if aid, err = ValueToAid(k); err != nil {
			return
		}
		if tag, err = ValueToTag(v); err != nil {
			return
		}

		ts[aid] = tag
	})
	return
}

// The End.
