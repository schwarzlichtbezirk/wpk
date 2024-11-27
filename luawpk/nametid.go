package luawpk

import (
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"time"

	lua "github.com/yuin/gopher-lua"

	"github.com/schwarzlichtbezirk/wpk"
)

const (
	TTany = iota
	TTbin
	TTstr
	TTbool
	TTuint
	TTnum
	TTtime
)

const ISO8601 = "2006-01-02T15:04:05.999Z07:00"

// TidType helps to convert raw tags to Lua values.
var TidType = map[wpk.TID]int{
	wpk.TIDoffset: TTuint,
	wpk.TIDsize:   TTuint,
	wpk.TIDpath:   TTstr,
	wpk.TIDfid:    TTuint,
	wpk.TIDmtime:  TTtime,
	wpk.TIDatime:  TTtime,
	wpk.TIDctime:  TTtime,
	wpk.TIDbtime:  TTtime,
	wpk.TIDattr:   TTuint,
	wpk.TIDmime:   TTstr,

	wpk.TIDcrc32ieee: TTbin,
	wpk.TIDcrc32c:    TTbin,
	wpk.TIDcrc32k:    TTbin,
	wpk.TIDcrc64iso:  TTbin,

	wpk.TIDmd5:    TTbin,
	wpk.TIDsha1:   TTbin,
	wpk.TIDsha224: TTbin,
	wpk.TIDsha256: TTbin,
	wpk.TIDsha384: TTbin,
	wpk.TIDsha512: TTbin,

	wpk.TIDtmbjpeg:  TTbin,
	wpk.TIDtmbwebp:  TTbin,
	wpk.TIDlabel:    TTstr,
	wpk.TIDlink:     TTstr,
	wpk.TIDkeywords: TTstr,
	wpk.TIDcategory: TTstr,
	wpk.TIDversion:  TTstr,
	wpk.TIDauthor:   TTstr,
	wpk.TIDcomment:  TTstr,
}

// NameTid helps convert Lua-table string keys to associated TID values.
var NameTid = map[string]wpk.TID{
	"offset": wpk.TIDoffset,
	"size":   wpk.TIDsize,
	"path":   wpk.TIDpath,
	"fid":    wpk.TIDfid,
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

	"tmbjpeg":  wpk.TIDtmbjpeg,
	"tmbwebp":  wpk.TIDtmbwebp,
	"label":    wpk.TIDlabel,
	"link":     wpk.TIDlink,
	"keywords": wpk.TIDkeywords,
	"category": wpk.TIDcategory,
	"version":  wpk.TIDversion,
	"author":   wpk.TIDauthor,
	"comment":  wpk.TIDcomment,
}

// TidName helps format Lua-tables with string keys associated to TID values.
var TidName = func() map[wpk.TID]string {
	var tn = map[wpk.TID]string{}
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

// ErrProtected is "protected tag" error.
type ErrProtected struct {
	tid wpk.TID
}

func (e *ErrProtected) Error() string {
	return fmt.Sprintf("tries to change protected tag '%s'", TidName[e.tid])
}

// Tags identifiers conversion errors.
var (
	ErrBadTagKey = errors.New("tag key type is not number or string")
	ErrBadTagVal = errors.New("tag value type is not string or boolean or 'tag' userdata")
)

// ValueToTID converts LValue to uint16 tag identifier.
// Numbers converts explicitly, strings converts to uint16
// values which they presents. Error returns on any other case.
func ValueToTID(k lua.LValue) (tid wpk.TID, err error) {
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

// ValueToTag converts LValue to TagRaw. Strings converts explicitly to byte sequence,
// boolen converts to 1 byte slice with 1 for 'true' and 0 for 'false'.
// Otherwise if it is not 'tag' uservalue with TagRaw, returns error.
func ValueToTag(tid wpk.TID, v lua.LValue) (tag wpk.TagRaw, err error) {
	switch TidType[tid] {
	case TTbin:
		if val, ok := v.(lua.LNumber); ok {
			tag = wpk.UintTag(uint(val))
		} else if val, ok := v.(lua.LString); ok {
			tag, err = hex.DecodeString(string(val))
		} else {
			err = ErrBadTagVal
			return
		}
	case TTstr:
		if val, ok := v.(lua.LNumber); ok {
			tag = wpk.StrTag(val.String())
		} else if val, ok := v.(lua.LString); ok {
			tag = wpk.StrTag(val.String())
		} else if val, ok := v.(lua.LBool); ok {
			tag = wpk.StrTag(val.String())
		} else {
			err = ErrBadTagVal
			return
		}
	case TTbool:
		if val, ok := v.(lua.LNumber); ok {
			tag = wpk.BoolTag(val != 0)
		} else if val, ok := v.(lua.LString); ok {
			tag = wpk.BoolTag(len(val) > 0 && val != "false")
		} else if val, ok := v.(lua.LBool); ok {
			tag = wpk.BoolTag(bool(val))
		} else {
			err = ErrBadTagVal
		}
	case TTuint:
		if val, ok := v.(lua.LNumber); ok {
			tag = wpk.UintTag(uint(val))
		} else if val, ok := v.(lua.LString); ok {
			var i int
			if i, err = strconv.Atoi(string(val)); err != nil {
				return
			}
			tag = wpk.UintTag(uint(i))
		} else if val, ok := v.(lua.LBool); ok {
			if val {
				tag = wpk.UintTag(1)
			} else {
				tag = wpk.UintTag(0)
			}
		} else {
			err = ErrBadTagVal
			return
		}
	case TTnum:
		if val, ok := v.(lua.LNumber); ok {
			tag = wpk.NumberTag(float64(val))
		} else if val, ok := v.(lua.LString); ok {
			var f float64
			if f, err = strconv.ParseFloat(string(val), 64); err != nil {
				return
			}
			tag = wpk.NumberTag(f)
		} else if val, ok := v.(lua.LBool); ok {
			if val {
				tag = wpk.NumberTag(1)
			} else {
				tag = wpk.NumberTag(0)
			}
		} else {
			err = ErrBadTagVal
			return
		}
	case TTtime:
		if val, ok := v.(lua.LNumber); ok {
			var milli = int64(val)
			var t = time.Unix(milli/1000, (milli%1000)*1000000)
			tag = wpk.UnixmsTag(t)
		} else if val, ok := v.(lua.LString); ok {
			var t time.Time
			if t, err = time.Parse(ISO8601, string(val)); err != nil {
				return
			}
			tag = wpk.TimeTag(t)
		} else {
			err = ErrBadTagVal
			return
		}
	default:
		if val, ok := v.(lua.LNumber); ok {
			if val < 0 || float64(val) != float64(int64(val)) {
				tag = wpk.NumberTag(float64(val))
			} else {
				tag = wpk.UintTag(uint(val))
			}
		} else if val, ok := v.(lua.LString); ok {
			tag = wpk.StrTag(string(val))
		} else if val, ok := v.(lua.LBool); ok {
			tag = wpk.BoolTag(bool(val))
		} else {
			err = ErrBadTagVal
			return
		}
	}
	return
}

func TagToValue(tid wpk.TID, tag wpk.TagRaw) (v lua.LValue, err error) {
	switch TidType[tid] {
	default: // TTany, TTstr
		var val, _ = tag.TagStr()
		v = lua.LString(val)
	case TTbin:
		v = lua.LString(hex.EncodeToString(tag))
	case TTbool:
		var val, ok = tag.TagUint()
		if !ok {
			err = ErrBadTagVal
			return
		}
		v = lua.LBool(val != 0)
	case TTuint:
		var val, ok = tag.TagUint()
		if !ok {
			err = ErrBadTagVal
			return
		}
		v = lua.LNumber(val)
	case TTnum:
		var val, ok = tag.TagNumber()
		if !ok {
			err = ErrBadTagVal
			return
		}
		v = lua.LNumber(val)
	case TTtime:
		var val, ok = tag.TagTime()
		if !ok {
			err = ErrBadTagVal
			return
		}
		v = lua.LString(val.UTC().Format(ISO8601))
	}
	return
}

// TableToTagset converts Lua-table to TagsetRaw. Lua-table keys can be number identifiers
// or string names associated ID values. Lua-table values can be strings, numbers,
// or boolean values.
func TableToTagset(lt *lua.LTable, ts wpk.TagsetRaw) (wpk.TagsetRaw, error) {
	var err error
	var errs []error
	lt.ForEach(func(k lua.LValue, v lua.LValue) {
		var (
			errk error
			errv error
			tid  wpk.TID
			tag  wpk.TagRaw
		)

		if tid, errk = ValueToTID(k); errk != nil {
			errs = append(errs, errk)
		} else if tid == wpk.TIDoffset || tid == wpk.TIDsize || tid == wpk.TIDpath {
			errk = &ErrProtected{tid}
			errs = append(errs, errk)
		}
		if tag, errv = ValueToTag(tid, v); err != nil {
			errs = append(errs, errv)
		}

		if errk == nil && errv == nil {
			ts = ts.Set(tid, tag)
		}
	})
	return ts, errors.Join(errs...)
}

// The End.
