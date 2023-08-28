package wpk

import (
	"encoding/binary"
	"io/fs"
	"math"
	"path"
	"time"
)

// TagRaw - file description item.
type TagRaw []byte

// TagStr tag converter.
func (t TagRaw) TagStr() (string, bool) {
	return B2S(t), true
}

// StrTag is string tag constructor.
func StrTag(val string) TagRaw {
	return S2B(val)
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
		return uint16(binary.LittleEndian.Uint16(t)), true
	}
	return 0, false
}

// Uint16Tag is 16-bit unsigned int tag constructor.
func Uint16Tag(val uint16) TagRaw {
	var buf [2]byte
	binary.LittleEndian.PutUint16(buf[:], uint16(val))
	return buf[:]
}

// TagUint32 is 32-bit unsigned int tag converter.
func (t TagRaw) TagUint32() (uint32, bool) {
	if len(t) == 4 {
		return binary.LittleEndian.Uint32(t), true
	}
	return 0, false
}

// Uint32Tag is 32-bit unsigned int tag constructor.
func Uint32Tag(val uint32) TagRaw {
	var buf [4]byte
	binary.LittleEndian.PutUint32(buf[:], val)
	return buf[:]
}

// TagUint64 is 64-bit unsigned int tag converter.
func (t TagRaw) TagUint64() (uint64, bool) {
	if len(t) == 8 {
		return binary.LittleEndian.Uint64(t), true
	}
	return 0, false
}

// Uint64Tag is 64-bit unsigned int tag constructor.
func Uint64Tag(val uint64) TagRaw {
	var buf [8]byte
	binary.LittleEndian.PutUint64(buf[:], val)
	return buf[:]
}

// TagUint is unspecified size unsigned int tag converter.
func (t TagRaw) TagUint() (ret Uint, ok bool) {
	switch len(t) {
	case 8:
		ret |= Uint(t[7]) << 56
		ret |= Uint(t[6]) << 48
		ret |= Uint(t[5]) << 40
		ret |= Uint(t[4]) << 32
		fallthrough
	case 4:
		ret |= Uint(t[3]) << 24
		ret |= Uint(t[2]) << 16
		fallthrough
	case 2:
		ret |= Uint(t[1]) << 8
		fallthrough
	case 1:
		ret |= Uint(t[0])
		ok = true
	}
	return
}

// UintTag is unspecified size unsigned int tag constructor.
func UintTag(val Uint) TagRaw {
	var l int
	var buf [8]byte
	switch {
	case val > 0xffffffff:
		if l == 0 {
			l = 8
		}
		buf[7] = byte(val >> 56)
		buf[6] = byte(val >> 48)
		buf[5] = byte(val >> 40)
		buf[4] = byte(val >> 32)
		fallthrough
	case val > 0xffff:
		if l == 0 {
			l = 4
		}
		buf[3] = byte(val >> 24)
		buf[2] = byte(val >> 16)
		fallthrough
	case val > 0xff:
		if l == 0 {
			l = 2
		}
		buf[1] = byte(val >> 8)
		fallthrough
	default:
		if l == 0 {
			l = 1
		}
		buf[0] = byte(val)
	}
	return buf[:l]
}

// UintLenTag is unsigned int tag constructor with specified length in bytes.
func UintLenTag(val Uint, l byte) TagRaw {
	var buf [8]byte
	switch l {
	case 8:
		buf[7] = byte(val >> 56)
		buf[6] = byte(val >> 48)
		buf[5] = byte(val >> 40)
		buf[4] = byte(val >> 32)
		fallthrough
	case 4:
		buf[3] = byte(val >> 24)
		buf[2] = byte(val >> 16)
		fallthrough
	case 2:
		buf[1] = byte(val >> 8)
		fallthrough
	case 1:
		buf[0] = byte(val)
	default:
		panic("undefined condition")
	}
	return buf[:l]
}

// TagNumber is 64-bit float tag converter.
func (t TagRaw) TagNumber() (float64, bool) {
	if len(t) == 8 {
		return math.Float64frombits(binary.LittleEndian.Uint64(t)), true
	}
	return 0, false
}

// NumberTag is 64-bit float tag constructor.
func NumberTag(val float64) TagRaw {
	var buf [8]byte
	binary.LittleEndian.PutUint64(buf[:], math.Float64bits(val))
	return buf[:]
}

// TagTime is 8/12-bytes time tag converter.
func (t TagRaw) TagTime() (time.Time, bool) {
	switch len(t) {
	case 8:
		var milli = int64(binary.LittleEndian.Uint64(t))
		return time.Unix(milli/1000, (milli%1000)*1000000), true
	case 12:
		var sec = int64(binary.LittleEndian.Uint64(t[:8]))
		var nsec = int64(binary.LittleEndian.Uint32(t[8:]))
		return time.Unix(sec, nsec), true
	}
	return time.Time{}, false
}

// UnixTag is 8-bytes UNIX time in milliseconds tag constructor.
func UnixTag(val time.Time) TagRaw {
	var buf [8]byte
	binary.LittleEndian.PutUint64(buf[:], uint64(val.UnixMilli()))
	return buf[:]
}

// TimeTag is 12-bytes UNIX time tag constructor.
func TimeTag(val time.Time) TagRaw {
	var buf [12]byte
	binary.LittleEndian.PutUint64(buf[:8], uint64(val.Unix()))
	binary.LittleEndian.PutUint32(buf[8:], uint32(val.Nanosecond()))
	return buf[:]
}

// TagsetRaw is slice of bytes with tags set. Length of slice can be
// not determined to record end, i.e. slice starts at record beginning
// (at number of tags), and can continues after record end.
// fs.FileInfo interface implementation.
type TagsetRaw struct {
	data []byte
}

// MakeTagset returns tagset with given slice.
func MakeTagset(data []byte) *TagsetRaw {
	return &TagsetRaw{data}
}

// Data returns whole tagset content.
func (ts *TagsetRaw) Data() []byte {
	return ts.data
}

// Num returns number of tags in tagset.
func (ts *TagsetRaw) Num() (n int) {
	var tsi = ts.Iterator()
	for tsi.Next() {
		n++
	}
	return
}

// Has checks existence of tag with given ID.
func (ts *TagsetRaw) Has(tid Uint) bool {
	var tsi = ts.Iterator()
	for tsi.Next() && tsi.tid != tid {
	}
	return tsi.tid == tid
}

// GetTag returns TagRaw with given identifier.
// If tag is not found, slice content is broken,
// returns false.
func (ts *TagsetRaw) Get(tid Uint) (TagRaw, bool) {
	var tsi = ts.Iterator()
	for tsi.Next() && tsi.tid != tid {
	}
	if tsi.Failed() {
		return nil, false // io.ErrUnexpectedEOF
	}
	if tsi.tid != tid {
		return nil, false // ErrNoTag
	}
	return tsi.data[tsi.tag:tsi.pos], true
}

// Put appends new tag to tagset.
// Can be used in chain calls at initialization.
func (ts *TagsetRaw) Put(tid Uint, tag TagRaw) *TagsetRaw {
	if tid == TIDnone { // special case
		return ts
	}

	var buf = make([]byte, PTStidsz+PTStagsz)
	WriteUintBuf(buf[:PTStidsz], tid)
	WriteUintBuf(buf[PTStidsz:], Uint(len(tag)))
	ts.data = append(ts.data, buf...)
	ts.data = append(ts.data, tag...)
	return ts
}

// Set replaces tag with given ID and equal size, or
// appends it to tagset. Returns true if new one added.
func (ts *TagsetRaw) Set(tid Uint, tag TagRaw) bool {
	if tid == TIDnone { // special case
		return false
	}

	var tsi = ts.Iterator()
	for tsi.Next() && tsi.tid != tid {
	}
	if tsi.tid != tid {
		ts.Put(tid, tag)
		return true
	}

	var tl = Uint(len(tag))
	if tl == Uint(tsi.pos-tsi.tag) {
		copy(ts.data[tsi.tag:tsi.pos], tag)
	} else {
		WriteUintBuf(ts.data[tsi.tag-PTStagsz:tsi.tag], tl) // set tag length
		var suff = ts.data[tsi.pos:]
		ts.data = append(ts.data[:tsi.tag], tag...)
		ts.data = append(ts.data, suff...)
	}
	return false
}

// Del deletes tag with given ID.
func (ts *TagsetRaw) Del(tid Uint) bool {
	if tid == TIDnone { // special case
		return false
	}

	var tsi = ts.Iterator()
	for tsi.Next() && tsi.tid != tid {
	}
	if tsi.tid != tid {
		return false // ErrNoTag
	}
	ts.data = append(ts.data[:tsi.tag-PTStagsz-PTStidsz], ts.data[tsi.pos:]...)
	return true
}

// TagStr tag getter.
func (ts *TagsetRaw) TagStr(tid Uint) (string, bool) {
	if data, ok := ts.Get(tid); ok {
		return data.TagStr()
	}
	return "", false
}

// TagBool is boolean tag getter.
func (ts *TagsetRaw) TagBool(tid Uint) (bool, bool) {
	if data, ok := ts.Get(tid); ok {
		return data.TagBool()
	}
	return false, false
}

// TagByte tag getter.
func (ts *TagsetRaw) TagByte(tid Uint) (byte, bool) {
	if data, ok := ts.Get(tid); ok {
		return data.TagByte()
	}
	return 0, false
}

// TagUint16 is 16-bit unsigned int tag getter.
// Conversion can be used to get signed 16-bit integers.
func (ts *TagsetRaw) TagUint16(tid Uint) (uint16, bool) {
	if data, ok := ts.Get(tid); ok {
		return data.TagUint16()
	}
	return 0, false
}

// TagUint32 is 32-bit unsigned int tag getter.
// Conversion can be used to get signed 32-bit integers.
func (ts *TagsetRaw) TagUint32(tid Uint) (uint32, bool) {
	if data, ok := ts.Get(tid); ok {
		return data.TagUint32()
	}
	return 0, false
}

// TagUint64 is 64-bit unsigned int tag getter.
// Conversion can be used to get signed 64-bit integers.
func (ts *TagsetRaw) TagUint64(tid Uint) (uint64, bool) {
	if data, ok := ts.Get(tid); ok {
		return data.TagUint64()
	}
	return 0, false
}

// TagUint is unspecified size unsigned int tag getter.
func (ts *TagsetRaw) TagUint(tid Uint) (Uint, bool) {
	if data, ok := ts.Get(tid); ok {
		return data.TagUint()
	}
	return 0, false
}

// TagNumber is 64-bit float tag getter.
func (ts *TagsetRaw) TagNumber(tid Uint) (float64, bool) {
	if data, ok := ts.Get(tid); ok {
		return data.TagNumber()
	}
	return 0, false
}

// TagTime is time tag getter.
func (ts *TagsetRaw) TagTime(tid Uint) (time.Time, bool) {
	if data, ok := ts.Get(tid); ok {
		return data.TagTime()
	}
	return time.Time{}, false
}

// Pos returns file offset and file size in package.
// Those values required to be present in any tagset.
func (ts *TagsetRaw) Pos() (offset, size Uint) {
	offset, _ = ts.TagUint(TIDoffset)
	size, _ = ts.TagUint(TIDsize)
	return
}

// Path returns path of nested into package file.
// Path required to be present in any tagset.
func (ts *TagsetRaw) Path() string {
	var fpath, _ = ts.TagStr(TIDpath)
	return fpath
}

// Name returns base name of nested into package file.
// fs.FileInfo implementation.
func (ts *TagsetRaw) Name() string {
	var fpath, _ = ts.TagStr(TIDpath)
	return path.Base(fpath) // path should be here with true slashes
}

// Size returns size of nested into package file.
// fs.FileInfo implementation.
func (ts *TagsetRaw) Size() int64 {
	var size, _ = ts.TagUint(TIDsize)
	return int64(size)
}

// Mode is for fs.FileInfo interface compatibility.
func (ts *TagsetRaw) Mode() fs.FileMode {
	if ts.Has(TIDsize) { // file size is absent for dir
		return 0444
	}
	return fs.ModeDir
}

// ModTime returns file modification timestamp of nested into package file.
// fs.FileInfo & times.Timespec implementation.
func (ts *TagsetRaw) ModTime() time.Time {
	var t, _ = ts.TagTime(TIDmtime)
	return t
}

// IsDir detects that object presents a directory. Directory can not have file ID.
// fs.FileInfo implementation.
func (ts *TagsetRaw) IsDir() bool {
	return !ts.Has(TIDsize) // file size is absent for dir
}

// Sys is for fs.FileInfo interface compatibility.
func (ts *TagsetRaw) Sys() interface{} {
	return nil
}

// Type is for fs.DirEntry interface compatibility.
func (ts *TagsetRaw) Type() fs.FileMode {
	if ts.Has(TIDsize) { // file size is absent for dir
		return 0444
	}
	return fs.ModeDir
}

// Info returns the FileInfo for the file or subdirectory described by the entry.
// fs.DirEntry interface implementation.
func (ts *TagsetRaw) Info() (fs.FileInfo, error) {
	return ts, nil
}

// AccessTime returns file access timestamp of nested into package file.
// times.Timespec implementation.
func (ts *TagsetRaw) AccessTime() time.Time {
	var t, _ = ts.TagUint64(TIDatime)
	return time.Unix(int64(t), 0)
}

// ChangeTime returns file change timestamp of nested into package file.
// times.Timespec implementation.
func (ts *TagsetRaw) ChangeTime() time.Time {
	var t, _ = ts.TagUint64(TIDctime)
	return time.Unix(int64(t), 0)
}

// BirthTime returns file access timestamp of nested into package file.
// times.Timespec implementation.
func (ts *TagsetRaw) BirthTime() time.Time {
	var t, _ = ts.TagUint64(TIDbtime)
	return time.Unix(int64(t), 0)
}

// HasChangeTime is times.Timespec interface implementation.
// Returns whether change timestamp is present.
func (ts *TagsetRaw) HasChangeTime() bool {
	return ts.Has(TIDctime)
}

// HasBirthTime is times.Timespec interface implementation.
// Returns whether birth timestamp is present.
func (ts *TagsetRaw) HasBirthTime() bool {
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
	tid Uint // tag ID of last readed tag
	pos int  // current position in the slice
	tag int  // start position of last readed tag content
}

// Reset restarts iterator for new iterations loop.
func (tsi *TagsetIterator) Reset() {
	tsi.tid = TIDnone
	tsi.pos = 0
	tsi.tag = 0
}

// TID returns the tag ID of the last readed tag.
func (tsi *TagsetIterator) TID() Uint {
	return tsi.tid
}

// Tag returns tag slice of the last readed tag content.
func (tsi *TagsetIterator) Tag() TagRaw {
	if tsi.Failed() {
		return nil
	}
	return tsi.data[tsi.tag:tsi.pos]
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
	return tsi.pos == len(tsi.data)
}

// Failed points that iterator is finished by broken tagset state.
func (tsi *TagsetIterator) Failed() bool {
	return tsi.pos > len(tsi.data)
}

// Next carries to the next tag position.
func (tsi *TagsetIterator) Next() (ok bool) {
	var tsl = len(tsi.data)
	tsi.tid = TIDnone

	// check up the end of tagset is reached by any reason
	if tsi.pos >= tsl {
		return
	}

	// get tag identifier
	if tsi.pos += PTStidsz; tsi.pos > tsl {
		return
	}
	var tid = ReadUintBuf(tsi.data[tsi.pos-PTStidsz : tsi.pos])

	// get tag length
	if tsi.pos += PTStagsz; tsi.pos > tsl {
		return
	}
	var len = ReadUintBuf(tsi.data[tsi.pos-PTStagsz : tsi.pos])
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

const tsiconst = "content changes are disabled for iterator"

// Put is the stub to disable any changes to data content of iterator.
func (tsi *TagsetIterator) Put(tid Uint, tag TagRaw) *TagsetRaw {
	panic(tsiconst)
}

// Set is the stub to disable any changes to data content of iterator.
func (tsi *TagsetIterator) Set(tid Uint, tag TagRaw) bool {
	panic(tsiconst)
}

// Del is the stub to disable any changes to data content of iterator.
func (tsi *TagsetIterator) Del(tid Uint) bool {
	panic(tsiconst)
}

// The End.
