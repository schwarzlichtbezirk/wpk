package wpk

import (
	"errors"
	"io/fs"
	"path"
	"time"

	"github.com/schwarzlichtbezirk/wpk/util"
)

var (
	ErrBadIntLen = errors.New("unacceptable integer length")
	ErrTsiConst  = errors.New("content changes are disabled for iterator")
)

// TagRaw - file description item.
type TagRaw []byte

// TagStr tag converter.
func (t TagRaw) TagStr() (string, bool) {
	return util.B2S(t), true
}

// StrTag is string tag constructor.
func StrTag(val string) TagRaw {
	return util.S2B(val)
}

// TagBool is boolean tag converter.
func (t TagRaw) TagBool() (bool, bool) {
	if len(t) == 1 {
		return t[0] > 0, true
	}
	return false, false
}

// BoolTag is boolean tag constructor.
func BoolTag(val bool) TagRaw {
	var buf [1]byte
	if val {
		buf[0] = 1
	}
	return buf[:]
}

// TagByte tag converter.
func (t TagRaw) TagByte() (byte, bool) {
	if len(t) == 1 {
		return t[0], true
	}
	return 0, false
}

// ByteTag is byte tag constructor.
func ByteTag(val byte) TagRaw {
	var buf = [1]byte{val}
	return buf[:]
}

// TagUint16 is 16-bit unsigned int tag converter.
func (t TagRaw) TagUint16() (uint16, bool) {
	if len(t) == 2 {
		return util.GetU16(t), true
	}
	return 0, false
}

// Uint16Tag is 16-bit unsigned int tag constructor.
func Uint16Tag(val uint16) TagRaw {
	var buf [2]byte
	util.SetU16(buf[:], val)
	return buf[:]
}

// TagUint32 is 32-bit unsigned int tag converter.
func (t TagRaw) TagUint32() (uint32, bool) {
	if len(t) == 4 {
		return util.GetU32(t), true
	}
	return 0, false
}

// Uint32Tag is 32-bit unsigned int tag constructor.
func Uint32Tag(val uint32) TagRaw {
	var buf [4]byte
	util.SetU32(buf[:], val)
	return buf[:]
}

// TagUint64 is 64-bit unsigned int tag converter.
func (t TagRaw) TagUint64() (uint64, bool) {
	if len(t) == 8 {
		return util.GetU64(t), true
	}
	return 0, false
}

// Uint64Tag is 64-bit unsigned int tag constructor.
func Uint64Tag(val uint64) TagRaw {
	var buf [8]byte
	util.SetU64(buf[:], val)
	return buf[:]
}

// TagUint is unspecified size unsigned int tag converter.
func (t TagRaw) TagUint() (uint, bool) {
	switch len(t) {
	case 8:
		return uint(util.GetU64(t)), true
	case 4:
		return uint(util.GetU32(t)), true
	case 2:
		return uint(util.GetU16(t)), true
	case 1:
		return uint(t[0]), true
	}
	return 0, false
}

// UintTag is unspecified size unsigned int tag constructor.
func UintTag(val uint) TagRaw {
	switch {
	case val > 0xffffffff:
		var buf [8]byte
		util.SetU64(buf[:], uint64(val))
		return buf[:]
	case val > 0xffff:
		var buf [4]byte
		util.SetU32(buf[:], uint32(val))
		return buf[:]
	case val > 0xff:
		var buf [2]byte
		util.SetU16(buf[:], uint16(val))
		return buf[:]
	default:
		var buf [1]byte
		buf[0] = byte(val)
		return buf[:]
	}
}

// UintLenTag is unsigned int tag constructor with specified length in bytes.
func UintLenTag(val uint, l int) TagRaw {
	switch l {
	case 8:
		var buf [8]byte
		util.SetU64(buf[:], uint64(val))
		return buf[:]
	case 4:
		var buf [4]byte
		util.SetU32(buf[:], uint32(val))
		return buf[:]
	case 2:
		var buf [2]byte
		util.SetU16(buf[:], uint16(val))
		return buf[:]
	case 1:
		var buf [1]byte
		buf[0] = byte(val)
		return buf[:]
	default:
		panic(ErrBadIntLen)
	}
}

// TagNumber is 64-bit float tag converter.
func (t TagRaw) TagNumber() (float64, bool) {
	if len(t) == 8 {
		return util.GetF64(t), true
	}
	return 0, false
}

// NumberTag is 64-bit float tag constructor.
func NumberTag(val float64) TagRaw {
	var buf [8]byte
	util.SetF64(buf[:], val)
	return buf[:]
}

// TagTime is 8/12-bytes time tag converter.
func (t TagRaw) TagTime() (time.Time, bool) {
	switch len(t) {
	case 8:
		var milli = int64(util.GetU64(t))
		return time.Unix(milli/1000, (milli%1000)*1000000), true
	case 12:
		var sec = int64(util.GetU64(t[:8]))
		var nsec = int64(util.GetU32(t[8:]))
		return time.Unix(sec, nsec), true
	}
	return time.Time{}, false
}

// TagUnixms is 8/12-bytes tag converter to UNIX time in milliseconds.
func (t TagRaw) TagUnixms() (int64, bool) {
	switch len(t) {
	case 8:
		return int64(util.GetU64(t)), true
	case 12:
		var sec = int64(util.GetU64(t[:8]))
		var nsec = int64(util.GetU32(t[8:]))
		return time.Unix(sec, nsec).UnixMilli(), true
	}
	return 0, false
}

// UnixmsTag is 8-bytes UNIX time in milliseconds tag constructor.
func UnixmsTag(val time.Time) TagRaw {
	var buf [8]byte
	util.SetU64(buf[:], uint64(val.UnixMilli()))
	return buf[:]
}

// TimeTag is 12-bytes UNIX time tag constructor.
func TimeTag(val time.Time) TagRaw {
	var buf [12]byte
	util.SetU64(buf[:8], uint64(val.Unix()))
	util.SetU32(buf[8:], uint32(val.Nanosecond()))
	return buf[:]
}

// TagsetRaw is slice of bytes with tags set. Each tag should be with unique ID.
// fs.FileInfo interface implementation.
type TagsetRaw []byte

// CopyTagset makes copy of given tagset to new space.
// It prevents rewriting data at solid slice with FTT when tagset modify.
func CopyTagset(ts TagsetRaw) TagsetRaw {
	return append(TagsetRaw{}, ts...) // make copy with some extra space
}

// Num returns number of tags in tagset.
func (ts TagsetRaw) Num() (n int) {
	var tsi = ts.Iterator()
	for tsi.Next() {
		n++
	}
	return
}

// Has checks existence of tag with given ID.
func (ts TagsetRaw) Has(tid TID) bool {
	var tsi = ts.Iterator()
	for tsi.Next() && tsi.tid != tid {
	}
	return tsi.tid == tid
}

// GetTag returns TagRaw with given identifier.
// If tag is not found, slice content is broken,
// returns false.
func (ts TagsetRaw) Get(tid TID) (TagRaw, bool) {
	var tsi = ts.Iterator()
	for tsi.Next() && tsi.tid != tid {
	}
	if tsi.Failed() {
		return nil, false // io.ErrUnexpectedEOF
	}
	if tsi.tid != tid {
		return nil, false // ErrNoTag
	}
	return TagRaw(tsi.TagsetRaw[tsi.tag:tsi.pos]), true
}

const taghdrsz = PTStidsz + PTStagsz

// Put appends new tag to tagset.
// Can be used in chain calls at initialization.
func (ts TagsetRaw) Put(tid TID, tag TagRaw) TagsetRaw {
	var buf = make([]byte, taghdrsz+len(tag))
	util.SetU16(buf, tid)
	util.SetU16(buf[PTStidsz:], uint16(len(tag)))
	copy(buf[taghdrsz:], tag)
	ts = append(ts, buf...)
	return ts
}

// AddOk appends tag with given ID only if tagset does not have same yet.
// Signature helps to build the call chain.
func (ts TagsetRaw) Add(tid TID, tag TagRaw) TagsetRaw {
	ts, _ = ts.AddOk(tid, tag)
	return ts
}

// AddOk appends tag with given ID only if tagset does not have same yet.
// Returns true if it added.
func (ts TagsetRaw) AddOk(tid TID, tag TagRaw) (TagsetRaw, bool) {
	var tsi = ts.Iterator()
	for tsi.Next() && tsi.tid != tid {
	}
	if tsi.tid == tid {
		return ts, false
	}
	return ts.Put(tid, tag), true
}

// Set replaces tag with given ID and equal size, or
// appends it to tagset. Signature helps to build the call chain.
func (ts TagsetRaw) Set(tid TID, tag TagRaw) TagsetRaw {
	ts, _ = ts.SetOk(tid, tag)
	return ts
}

// SetOk replaces tag with given ID and equal size, or
// appends it to tagset. Returns true if new one added.
func (ts TagsetRaw) SetOk(tid TID, tag TagRaw) (TagsetRaw, bool) {
	var tsi = ts.Iterator()
	for tsi.Next() && tsi.tid != tid {
	}
	if tsi.tid != tid {
		return ts.Put(tid, tag), true
	}

	var tl = uint16(len(tag))
	if tl == uint16(tsi.pos-tsi.tag) {
		copy(ts[tsi.tag:tsi.pos], tag)
	} else {
		util.SetU16(ts[tsi.tag-PTStagsz:tsi.tag], tl) // set tag length
		var suff = ts[tsi.pos:]
		ts = append(ts[:tsi.tag], tag...)
		ts = append(ts, suff...)
	}
	return ts, false
}

// Del deletes tag with given ID. Signature helps to build the call chain.
func (ts TagsetRaw) Del(tid TID) TagsetRaw {
	ts, _ = ts.DelOk(tid)
	return ts
}

// DelOk deletes tag with given ID. Returns true if tagset was modified.
func (ts TagsetRaw) DelOk(tid TID) (TagsetRaw, bool) {
	var tsi = ts.Iterator()
	for tsi.Next() && tsi.tid != tid {
	}
	if tsi.tid != tid {
		return ts, false // ErrNoTag
	}
	ts = append(ts[:tsi.tag-PTStagsz-PTStidsz], ts[tsi.pos:]...)
	return ts, true
}

// TagStr tag getter.
func (ts TagsetRaw) TagStr(tid TID) (string, bool) {
	if data, ok := ts.Get(tid); ok {
		return data.TagStr()
	}
	return "", false
}

// TagBool is boolean tag getter.
func (ts TagsetRaw) TagBool(tid TID) (bool, bool) {
	if data, ok := ts.Get(tid); ok {
		return data.TagBool()
	}
	return false, false
}

// TagByte tag getter.
func (ts TagsetRaw) TagByte(tid TID) (byte, bool) {
	if data, ok := ts.Get(tid); ok {
		return data.TagByte()
	}
	return 0, false
}

// TagUint16 is 16-bit unsigned int tag getter.
// Conversion can be used to get signed 16-bit integers.
func (ts TagsetRaw) TagUint16(tid TID) (uint16, bool) {
	if data, ok := ts.Get(tid); ok {
		return data.TagUint16()
	}
	return 0, false
}

// TagUint32 is 32-bit unsigned int tag getter.
// Conversion can be used to get signed 32-bit integers.
func (ts TagsetRaw) TagUint32(tid TID) (uint32, bool) {
	if data, ok := ts.Get(tid); ok {
		return data.TagUint32()
	}
	return 0, false
}

// TagUint64 is 64-bit unsigned int tag getter.
// Conversion can be used to get signed 64-bit integers.
func (ts TagsetRaw) TagUint64(tid TID) (uint64, bool) {
	if data, ok := ts.Get(tid); ok {
		return data.TagUint64()
	}
	return 0, false
}

// TagUint is unspecified size unsigned int tag getter.
func (ts TagsetRaw) TagUint(tid TID) (uint, bool) {
	if data, ok := ts.Get(tid); ok {
		return data.TagUint()
	}
	return 0, false
}

// TagNumber is 64-bit float tag getter.
func (ts TagsetRaw) TagNumber(tid TID) (float64, bool) {
	if data, ok := ts.Get(tid); ok {
		return data.TagNumber()
	}
	return 0, false
}

// TagTime is time tag getter.
func (ts TagsetRaw) TagTime(tid TID) (time.Time, bool) {
	if data, ok := ts.Get(tid); ok {
		return data.TagTime()
	}
	return time.Time{}, false
}

// TagUnixms is UNIX time in milliseconds tag getter.
func (ts TagsetRaw) TagUnixms(tid TID) (int64, bool) {
	if data, ok := ts.Get(tid); ok {
		return data.TagUnixms()
	}
	return 0, false
}

// Pos returns file offset and file size in package.
// Those values required to be present in any tagset.
func (ts TagsetRaw) Pos() (offset, size uint) {
	var tsi, n = ts.Iterator(), 2
	for tsi.Next() && n > 0 {
		switch tsi.tid {
		case TIDoffset:
			offset, _ = TagRaw(tsi.TagsetRaw[tsi.tag:tsi.pos]).TagUint()
			n--
		case TIDsize:
			size, _ = TagRaw(tsi.TagsetRaw[tsi.tag:tsi.pos]).TagUint()
			n--
		}
	}
	return
}

// Path returns path of nested into package file.
// Path required to be present in any tagset.
func (ts TagsetRaw) Path() string {
	var fpath, _ = ts.TagStr(TIDpath)
	return fpath
}

// Name returns base name of nested into package file.
// fs.FileInfo implementation.
func (ts TagsetRaw) Name() string {
	var fpath, _ = ts.TagStr(TIDpath)
	return path.Base(fpath) // path should be here with true slashes
}

// Size returns size of nested into package file.
// fs.FileInfo implementation.
func (ts TagsetRaw) Size() int64 {
	var size, _ = ts.TagUint(TIDsize)
	return int64(size)
}

// Mode is for fs.FileInfo interface compatibility.
func (ts TagsetRaw) Mode() fs.FileMode {
	if ts.Has(TIDsize) { // file size is absent for dir
		return 0444
	}
	return fs.ModeDir
}

// ModTime returns file modification timestamp of nested into package file.
// fs.FileInfo & times.Timespec implementation.
func (ts TagsetRaw) ModTime() time.Time {
	var t, _ = ts.TagTime(TIDmtime)
	return t
}

// IsDir detects that object presents a directory. Directory can not have file ID.
// fs.FileInfo implementation.
func (ts TagsetRaw) IsDir() bool {
	return !ts.Has(TIDsize) // file size is absent for dir
}

// Sys is for fs.FileInfo interface compatibility.
func (ts TagsetRaw) Sys() interface{} {
	return ts
}

// Type is for fs.DirEntry interface compatibility.
func (ts TagsetRaw) Type() fs.FileMode {
	if ts.Has(TIDsize) { // file size is absent for dir
		return 0444
	}
	return fs.ModeDir
}

// Info returns the FileInfo for the file or subdirectory described by the entry.
// fs.DirEntry interface implementation.
func (ts TagsetRaw) Info() (fs.FileInfo, error) {
	return ts, nil
}

// AccessTime returns file access timestamp of nested into package file.
// times.Timespec implementation.
func (ts TagsetRaw) AccessTime() time.Time {
	var t, _ = ts.TagUint64(TIDatime)
	return time.Unix(int64(t), 0)
}

// ChangeTime returns file change timestamp of nested into package file.
// times.Timespec implementation.
func (ts TagsetRaw) ChangeTime() time.Time {
	var t, _ = ts.TagUint64(TIDctime)
	return time.Unix(int64(t), 0)
}

// BirthTime returns file access timestamp of nested into package file.
// times.Timespec implementation.
func (ts TagsetRaw) BirthTime() time.Time {
	var t, _ = ts.TagUint64(TIDbtime)
	return time.Unix(int64(t), 0)
}

// HasChangeTime is times.Timespec interface implementation.
// Returns whether change timestamp is present.
func (ts TagsetRaw) HasChangeTime() bool {
	return ts.Has(TIDctime)
}

// HasBirthTime is times.Timespec interface implementation.
// Returns whether birth timestamp is present.
func (ts TagsetRaw) HasBirthTime() bool {
	return ts.Has(TIDbtime)
}

// Iterator clones this tagset to iterate through all tags.
func (ts TagsetRaw) Iterator() TagsetIterator {
	return TagsetIterator{
		TagsetRaw: ts,
	}
}

// TagsetIterator helps to iterate through all tags.
type TagsetIterator struct {
	TagsetRaw
	tid TID // tag ID of last readed tag
	pos int // current position in the slice
	tag int // start position of last readed tag content
}

// Reset restarts iterator for new iterations loop.
func (tsi *TagsetIterator) Reset() {
	tsi.tid = TIDnone
	tsi.pos = 0
	tsi.tag = 0
}

// TID returns the tag ID of the last readed tag.
func (tsi *TagsetIterator) TID() TID {
	return tsi.tid
}

// Tag returns tag slice of the last readed tag content.
func (tsi *TagsetIterator) Tag() TagRaw {
	if tsi.Failed() {
		return nil
	}
	return TagRaw(tsi.TagsetRaw[tsi.tag:tsi.pos])
}

// TagLen returns length of last readed tag content.
func (tsi *TagsetIterator) TagLen() int {
	if tsi.Failed() {
		return 0
	}
	return tsi.pos - tsi.tag
}

// Passed returns true if the end of iterations is successfully reached.
func (tsi *TagsetIterator) Passed() bool {
	return tsi.pos == len(tsi.TagsetRaw)
}

// Failed points that iterator is finished by broken tagset state.
func (tsi *TagsetIterator) Failed() bool {
	return tsi.pos > len(tsi.TagsetRaw)
}

// Next carries to the next tag position.
func (tsi *TagsetIterator) Next() (ok bool) {
	var tsl = len(tsi.TagsetRaw)
	tsi.tid = TIDnone

	// check up the end of tagset is reached by any reason
	if tsi.pos >= tsl {
		return
	}

	// get tag identifier
	if tsi.pos += PTStidsz; tsi.pos > tsl {
		return
	}
	var tid = util.GetU16(tsi.TagsetRaw[tsi.pos-PTStidsz : tsi.pos])

	// get tag length
	if tsi.pos += PTStagsz; tsi.pos > tsl {
		return
	}
	var len = util.GetU16(tsi.TagsetRaw[tsi.pos-PTStagsz : tsi.pos])
	// store tag content position
	var tag = tsi.pos

	// prepare to get tag content
	if tsi.pos += int(len); tsi.pos > tsl {
		return
	}

	tsi.tid, tsi.tag = tid, tag
	ok = true
	return
}

// Put is the stub to disable any changes to data content of iterator.
func (tsi *TagsetIterator) Put(tid TID, tag TagRaw) TagsetRaw {
	panic(ErrTsiConst)
}

// Set is the stub to disable any changes to data content of iterator.
func (tsi *TagsetIterator) Add(tid TID, tag TagRaw) TagsetRaw {
	panic(ErrTsiConst)
}

// Set is the stub to disable any changes to data content of iterator.
func (tsi *TagsetIterator) AddOk(tid TID, tag TagRaw) (TagsetRaw, bool) {
	panic(ErrTsiConst)
}

// Set is the stub to disable any changes to data content of iterator.
func (tsi *TagsetIterator) Set(tid TID, tag TagRaw) TagsetRaw {
	panic(ErrTsiConst)
}

// Set is the stub to disable any changes to data content of iterator.
func (tsi *TagsetIterator) SetOk(tid TID, tag TagRaw) (TagsetRaw, bool) {
	panic(ErrTsiConst)
}

// Del is the stub to disable any changes to data content of iterator.
func (tsi *TagsetIterator) Del(tid TID) TagsetRaw {
	panic(ErrTsiConst)
}

// DelOk is the stub to disable any changes to data content of iterator.
func (tsi *TagsetIterator) DelOk(tid TID) (TagsetRaw, bool) {
	panic(ErrTsiConst)
}

// The End.
