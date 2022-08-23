package wpk

import (
	"encoding/binary"
	"io/fs"
	"math"
	"path/filepath"
	"time"
)

// Tag_t - file description item.
type Tag_t []byte

// String tag converter.
func (t Tag_t) String() (string, bool) {
	return string(t), true
}

// TagString is string tag constructor.
func TagString(val string) Tag_t {
	return Tag_t(val)
}

// Bool is boolean tag converter.
func (t Tag_t) Bool() (bool, bool) {
	if len(t) == 1 {
		return t[0] > 0, true
	}
	return false, false
}

// TagBool is boolean tag constructor.
func TagBool(val bool) Tag_t {
	var buf [1]byte
	if val {
		buf[0] = 1
	}
	return buf[:]
}

// Byte tag converter.
func (t Tag_t) Byte() (byte, bool) {
	if len(t) == 1 {
		return t[0], true
	}
	return 0, false
}

// TagByte is Byte tag constructor.
func TagByte(val byte) Tag_t {
	var buf = [1]byte{val}
	return buf[:]
}

// Uint8 is 8-bit unsigned int tag converter.
func (t Tag_t) Uint8() (uint8, bool) {
	if len(t) == 1 {
		return t[0], true
	}
	return 0, false
}

// TagUint8 is 8-bit unsigned int tag constructor.
func TagUint8(val uint8) Tag_t {
	var buf = [1]byte{val}
	return buf[:]
}

// Uint16 is 16-bit unsigned int tag converter.
func (t Tag_t) Uint16() (uint16, bool) {
	if len(t) == 2 {
		return uint16(binary.LittleEndian.Uint16(t)), true
	}
	return 0, false
}

// TagUint16 is 16-bit unsigned int tag constructor.
func TagUint16(val uint16) Tag_t {
	var buf [2]byte
	binary.LittleEndian.PutUint16(buf[:], uint16(val))
	return buf[:]
}

// Uint32 is 32-bit unsigned int tag converter.
func (t Tag_t) Uint32() (uint32, bool) {
	if len(t) == 4 {
		return binary.LittleEndian.Uint32(t), true
	}
	return 0, false
}

// TagUint32 is 32-bit unsigned int tag constructor.
func TagUint32(val uint32) Tag_t {
	var buf [4]byte
	binary.LittleEndian.PutUint32(buf[:], val)
	return buf[:]
}

// Uint64 is 64-bit unsigned int tag converter.
func (t Tag_t) Uint64() (uint64, bool) {
	if len(t) == 8 {
		return binary.LittleEndian.Uint64(t), true
	}
	return 0, false
}

// TagUint64 is 64-bit unsigned int tag constructor.
func TagUint64(val uint64) Tag_t {
	var buf [8]byte
	binary.LittleEndian.PutUint64(buf[:], val)
	return buf[:]
}

// Uint is unspecified size unsigned int tag converter.
func (t Tag_t) Uint() (ret uint, ok bool) {
	switch len(t) {
	case 8:
		ret |= uint(t[7]) << 56
		ret |= uint(t[6]) << 48
		ret |= uint(t[5]) << 40
		ret |= uint(t[4]) << 32
		fallthrough
	case 4:
		ret |= uint(t[3]) << 24
		ret |= uint(t[2]) << 16
		fallthrough
	case 2:
		ret |= uint(t[1]) << 8
		fallthrough
	case 1:
		ret |= uint(t[0])
		ok = true
	}
	return
}

// TagUint is unspecified size unsigned int tag constructor.
func TagUint(val uint) Tag_t {
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

// TagUintLen is unsigned int tag constructor with specified length in bytes.
func TagUintLen(val uint, l byte) Tag_t {
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

// Number is 64-bit float tag converter.
func (t Tag_t) Number() (float64, bool) {
	if len(t) == 8 {
		return math.Float64frombits(binary.LittleEndian.Uint64(t)), true
	}
	return 0, false
}

// TagNumber is 64-bit float tag constructor.
func TagNumber(val float64) Tag_t {
	var buf [8]byte
	binary.LittleEndian.PutUint64(buf[:], math.Float64bits(val))
	return buf[:]
}

// Time is 8/12-bytes time tag converter.
func (t Tag_t) Time() (time.Time, bool) {
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

// TagUnix is 8-bytes UNIX time in milliseconds tag constructor.
func TagUnix(val time.Time) Tag_t {
	var buf [8]byte
	binary.LittleEndian.PutUint64(buf[:], uint64(val.UnixMilli()))
	return buf[:]
}

// TagTime is 12-bytes UNIX time tag constructor.
func TagTime(val time.Time) Tag_t {
	var buf [12]byte
	binary.LittleEndian.PutUint64(buf[:8], uint64(val.Unix()))
	binary.LittleEndian.PutUint32(buf[8:], uint32(val.Nanosecond()))
	return buf[:]
}

// Tagset_t is slice of bytes with tags set. Length of slice can be
// not determined to record end, i.e. slice starts at record beginning
// (at number of tags), and can continues after record end.
// fs.FileInfo interface implementation.
type Tagset_t struct {
	data  []byte
	tidsz byte
	tagsz byte
}

// MakeTagset returns tagset with given slice.
func MakeTagset(data []byte, tidsz, tagsz byte) *Tagset_t {
	return &Tagset_t{data, tidsz, tagsz}
}

// Data returns whole tagset content.
func (ts *Tagset_t) Data() []byte {
	return ts.data
}

// Num returns number of tags in tagset.
func (ts *Tagset_t) Num() (n int) {
	var tsi = ts.Iterator()
	for tsi.Next() {
		n++
	}
	return
}

// Has checks existence of tag with given ID.
func (ts *Tagset_t) Has(tid uint) bool {
	var tsi = ts.Iterator()
	for tsi.Next() && tsi.tid != tid {
	}
	return tsi.tid == tid
}

// GetTag returns Tag_t with given identifier.
// If tag is not found, slice content is broken,
// returns false.
func (ts *Tagset_t) Get(tid uint) (Tag_t, bool) {
	var tsi = ts.Iterator()
	for tsi.Next() && tsi.tid != tid {
	}
	if tsi.Failed() {
		return nil, false // io.ErrUnexpectedEOF
	}
	if tsi.tid != tid {
		return nil, false // ErrNoTag
	}
	return Tag_t(tsi.data[tsi.tag:tsi.pos]), true
}

// Put appends new tag to tagset.
// Can be used in chain calls at initialization.
func (ts *Tagset_t) Put(tid uint, tag Tag_t) *Tagset_t {
	if tid == TIDnone { // special case
		return ts
	}

	var buf = make([]byte, ts.tidsz+ts.tagsz)
	WriteUintBuf(buf[:ts.tidsz], tid)
	WriteUintBuf(buf[ts.tidsz:], uint(len(tag)))
	ts.data = append(ts.data, buf...)
	ts.data = append(ts.data, tag...)
	return ts
}

// Set replaces tag with given ID and equal size, or
// appends it to tagset. Returns true if new one added.
func (ts *Tagset_t) Set(tid uint, tag Tag_t) bool {
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

	var tl = uint(len(tag))
	if tl == uint(tsi.pos-tsi.tag) {
		copy(ts.data[tsi.tag:tsi.pos], tag)
	} else {
		WriteUintBuf(ts.data[tsi.tag-int(ts.tagsz):tsi.tag], tl) // set tag length
		var suff = ts.data[tsi.pos:]
		ts.data = append(ts.data[:tsi.tag], tag...)
		ts.data = append(ts.data, suff...)
	}
	return false
}

// Del deletes tag with given ID.
func (ts *Tagset_t) Del(tid uint) bool {
	if tid == TIDnone { // special case
		return false
	}

	var tsi = ts.Iterator()
	for tsi.Next() && tsi.tid != tid {
	}
	if tsi.tid != tid {
		return false // ErrNoTag
	}
	ts.data = append(ts.data[:tsi.tag-int(ts.tagsz)-int(ts.tidsz)], ts.data[tsi.pos:]...)
	return true
}

// String tag getter.
func (ts *Tagset_t) String(tid uint) (string, bool) {
	if data, ok := ts.Get(tid); ok {
		return data.String()
	}
	return "", false
}

// Bool is boolean tag getter.
func (ts *Tagset_t) Bool(tid uint) (bool, bool) {
	if data, ok := ts.Get(tid); ok {
		return data.Bool()
	}
	return false, false
}

// Byte tag getter.
func (ts *Tagset_t) Byte(tid uint) (byte, bool) {
	if data, ok := ts.Get(tid); ok {
		return data.Byte()
	}
	return 0, false
}

// Uint16 is 16-bit unsigned int tag getter.
// Conversion can be used to get signed 16-bit integers.
func (ts *Tagset_t) Uint16(tid uint) (uint16, bool) {
	if data, ok := ts.Get(tid); ok {
		return data.Uint16()
	}
	return 0, false
}

// Uint32 is 32-bit unsigned int tag getter.
// Conversion can be used to get signed 32-bit integers.
func (ts *Tagset_t) Uint32(tid uint) (uint32, bool) {
	if data, ok := ts.Get(tid); ok {
		return data.Uint32()
	}
	return 0, false
}

// Uint64 is 64-bit unsigned int tag getter.
// Conversion can be used to get signed 64-bit integers.
func (ts *Tagset_t) Uint64(tid uint) (uint64, bool) {
	if data, ok := ts.Get(tid); ok {
		return data.Uint64()
	}
	return 0, false
}

// Uint is unspecified size unsigned int tag getter.
func (ts *Tagset_t) Uint(tid uint) (uint, bool) {
	if data, ok := ts.Get(tid); ok {
		return data.Uint()
	}
	return 0, false
}

// Number is 64-bit float tag getter.
func (ts *Tagset_t) Number(tid uint) (float64, bool) {
	if data, ok := ts.Get(tid); ok {
		return data.Number()
	}
	return 0, false
}

// Time is time tag getter.
func (ts *Tagset_t) Time(tid uint) (time.Time, bool) {
	if data, ok := ts.Get(tid); ok {
		return data.Time()
	}
	return time.Time{}, false
}

// Pos returns file offset and file size in package.
// Those values required to be present in any tagset.
func (ts *Tagset_t) Pos() (offset, size uint) {
	offset, _ = ts.Uint(TIDoffset)
	size, _ = ts.Uint(TIDsize)
	return
}

// Path returns path of nested into package file.
// Path required to be present in any tagset.
func (ts *Tagset_t) Path() string {
	var fpath, _ = ts.String(TIDpath)
	return fpath
}

// Name returns base name of nested into package file.
// fs.FileInfo implementation.
func (ts *Tagset_t) Name() string {
	var fpath, _ = ts.String(TIDpath)
	return filepath.Base(fpath)
}

// Size returns size of nested into package file.
// fs.FileInfo implementation.
func (ts *Tagset_t) Size() int64 {
	var size, _ = ts.Uint(TIDsize)
	return int64(size)
}

// Mode is for fs.FileInfo interface compatibility.
func (ts *Tagset_t) Mode() fs.FileMode {
	if ts.Has(TIDsize) { // file size is absent for dir
		return 0444
	}
	return fs.ModeDir
}

// ModTime returns file modification timestamp of nested into package file.
// fs.FileInfo & times.Timespec implementation.
func (ts *Tagset_t) ModTime() time.Time {
	var t, _ = ts.Time(TIDmtime)
	return t
}

// IsDir detects that object presents a directory. Directory can not have file ID.
// fs.FileInfo implementation.
func (ts *Tagset_t) IsDir() bool {
	return !ts.Has(TIDsize) // file size is absent for dir
}

// Sys is for fs.FileInfo interface compatibility.
func (ts *Tagset_t) Sys() interface{} {
	return nil
}

// Type is for fs.DirEntry interface compatibility.
func (ts *Tagset_t) Type() fs.FileMode {
	if ts.Has(TIDsize) { // file size is absent for dir
		return 0444
	}
	return fs.ModeDir
}

// Info returns the FileInfo for the file or subdirectory described by the entry.
// fs.DirEntry interface implementation.
func (ts *Tagset_t) Info() (fs.FileInfo, error) {
	return ts, nil
}

// AccessTime returns file access timestamp of nested into package file.
// times.Timespec implementation.
func (ts *Tagset_t) AccessTime() time.Time {
	var t, _ = ts.Uint64(TIDatime)
	return time.Unix(int64(t), 0)
}

// ChangeTime returns file change timestamp of nested into package file.
// times.Timespec implementation.
func (ts *Tagset_t) ChangeTime() time.Time {
	var t, _ = ts.Uint64(TIDctime)
	return time.Unix(int64(t), 0)
}

// BirthTime returns file access timestamp of nested into package file.
// times.Timespec implementation.
func (ts *Tagset_t) BirthTime() time.Time {
	var t, _ = ts.Uint64(TIDbtime)
	return time.Unix(int64(t), 0)
}

// HasChangeTime is times.Timespec interface implementation.
// Returns whether change timestamp is present.
func (ts *Tagset_t) HasChangeTime() bool {
	return ts.Has(TIDctime)
}

// HasBirthTime is times.Timespec interface implementation.
// Returns whether birth timestamp is present.
func (ts *Tagset_t) HasBirthTime() bool {
	return ts.Has(TIDbtime)
}

// Iterator clones this tagset to iterate through all tags.
func (ts *Tagset_t) Iterator() *TagsetIterator {
	var tsi TagsetIterator
	tsi.Tagset_t = *ts
	return &tsi
}

// TagsetIterator helps to iterate through all tags.
type TagsetIterator struct {
	Tagset_t
	tid uint // tag ID of last readed tag
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
func (tsi *TagsetIterator) TID() uint {
	return tsi.tid
}

// Tag returns tag slice of the last readed tag content.
func (tsi *TagsetIterator) Tag() Tag_t {
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
	if tsi.pos += int(tsi.tidsz); tsi.pos > tsl {
		return
	}
	var tid = ReadUintBuf(tsi.data[tsi.pos-int(tsi.tidsz) : tsi.pos])

	// get tag length
	if tsi.pos += int(tsi.tagsz); tsi.pos > tsl {
		return
	}
	var len = ReadUintBuf(tsi.data[tsi.pos-int(tsi.tagsz) : tsi.pos])
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
func (tsi *TagsetIterator) Put(tid uint, tag Tag_t) *Tagset_t {
	panic(tsiconst)
}

// Set is the stub to disable any changes to data content of iterator.
func (tsi *TagsetIterator) Set(tid uint, tag Tag_t) bool {
	panic(tsiconst)
}

// Del is the stub to disable any changes to data content of iterator.
func (tsi *TagsetIterator) Del(tid uint) bool {
	panic(tsiconst)
}

// The End.
