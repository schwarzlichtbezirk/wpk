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

// Uint16 is 16-bit unsigned int tag converter.
func (t Tag_t) Uint16() (TID_t, bool) {
	if len(t) == 2 {
		return TID_t(binary.LittleEndian.Uint16(t)), true
	}
	return 0, false
}

// TagUint16 is 16-bit unsigned int tag constructor.
func TagUint16(val TID_t) Tag_t {
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
func (t Tag_t) Uint() (uint, bool) {
	switch len(t) {
	case 1:
		return uint(t[0]), true
	case 2:
		return uint(binary.LittleEndian.Uint16(t)), true
	case 4:
		return uint(binary.LittleEndian.Uint32(t)), true
	case 8:
		return uint(binary.LittleEndian.Uint64(t)), true
	}
	return 0, false
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

// Tagset_t is slice of bytes with tags set. Length of slice can be
// not determined to record end, i.e. slice starts at record beginning
// (at number of tags), and can continues after record end.
// fs.FileInfo interface implementation.
type Tagset_t struct {
	data []byte
}

// NewTagset returns new tagset with given slice.
func NewTagset(data []byte) *Tagset_t {
	return &Tagset_t{data}
}

// Data returns whole tagset content.
func (ts *Tagset_t) Data() []byte {
	return ts.data
}

// Num returns number of tags in tagset.
func (ts *Tagset_t) Num() uint16 {
	if len(ts.data) < 2 {
		return 0
	}
	return binary.LittleEndian.Uint16(ts.data[:2])
}

// Has checks existence of tag with given ID.
func (ts *Tagset_t) Has(tid TID_t) bool {
	var tsi = ts.Iterator()
	if tsi == nil {
		return false
	}
	for tsi.Next() && tsi.tid != tid {
	}
	return tsi.tid == tid
}

// GetTag returns Tag_t with given identifier.
// If tag is not found, slice content is broken,
// returns false.
func (ts *Tagset_t) Get(tid TID_t) (Tag_t, bool) {
	var tsi = ts.Iterator()
	if tsi == nil {
		return nil, false // ErrNoData
	}
	for tsi.Next() && tsi.tid != tid {
	}
	if tsi.pos > uint16(len(tsi.data)) {
		return nil, false // io.EOF
	}
	if tsi.tid != tid {
		return nil, false // ErrNoTag
	}
	return Tag_t(tsi.data[tsi.tag:tsi.pos]), true
}

// Put appends new tag to tagset.
func (ts *Tagset_t) Put(tid TID_t, tag Tag_t) {
	if len(ts.data) < 2 { // init empty slice
		ts.data = make([]byte, 2)
	}
	var num = binary.LittleEndian.Uint16(ts.data[:2])
	num++
	binary.LittleEndian.PutUint16(ts.data[:2], num)

	var buf [4]byte
	binary.LittleEndian.PutUint16(buf[0:2], uint16(tid))
	binary.LittleEndian.PutUint16(buf[2:4], uint16(len(tag)))
	ts.data = append(ts.data, buf[:]...)
	ts.data = append(ts.data, tag...)
}

// Set replaces tag with given ID and equal size, or
// appends it to tagset. Returns true if new one added.
func (ts *Tagset_t) Set(tid TID_t, tag Tag_t) bool {
	var tsi = ts.Iterator()
	if tsi == nil {
		ts.Put(tid, tag)
		return true
	}
	for tsi.Next() && tsi.tid != tid {
	}
	if tsi.tid != tid {
		ts.Put(tid, tag)
		return true
	}

	var tl = uint16(len(tag))
	if tl == tsi.pos-tsi.tag {
		copy(ts.data[tsi.tag:tsi.pos], tag)
	} else {
		binary.LittleEndian.PutUint16(ts.data[tsi.tag-2:tsi.tag], tl) // set tag length
		var suff = ts.data[tsi.pos:]
		ts.data = append(ts.data[:tsi.tag], tag...)
		ts.data = append(ts.data, suff...)
	}
	return false
}

// Del deletes tag with given ID.
func (ts *Tagset_t) Del(tid TID_t) bool {
	var tsi = ts.Iterator()
	if tsi == nil {
		return false // ErrNoData
	}
	for tsi.Next() && tsi.tid != tid {
	}
	if tsi.tid != tid {
		return false // ErrNoTag
	}
	tsi.num--
	binary.LittleEndian.PutUint16(ts.data[:2], tsi.num)
	ts.data = append(ts.data[:tsi.tag-4], ts.data[tsi.pos:]...)
	return true
}

// String tag getter.
func (ts *Tagset_t) String(tid TID_t) (string, bool) {
	if data, ok := ts.Get(tid); ok {
		return data.String()
	}
	return "", false
}

// Bool is boolean tag getter.
func (ts *Tagset_t) Bool(tid TID_t) (bool, bool) {
	if data, ok := ts.Get(tid); ok {
		return data.Bool()
	}
	return false, false
}

// Byte tag getter.
func (ts *Tagset_t) Byte(tid TID_t) (byte, bool) {
	if data, ok := ts.Get(tid); ok {
		return data.Byte()
	}
	return 0, false
}

// Uint16 is 16-bit unsigned int tag getter.
// Conversion can be used to get signed 16-bit integers.
func (ts *Tagset_t) Uint16(tid TID_t) (TID_t, bool) {
	if data, ok := ts.Get(tid); ok {
		return data.Uint16()
	}
	return 0, false
}

// Uint32 is 32-bit unsigned int tag getter.
// Conversion can be used to get signed 32-bit integers.
func (ts *Tagset_t) Uint32(tid TID_t) (uint32, bool) {
	if data, ok := ts.Get(tid); ok {
		return data.Uint32()
	}
	return 0, false
}

// Uint64 is 64-bit unsigned int tag getter.
// Conversion can be used to get signed 64-bit integers.
func (ts *Tagset_t) Uint64(tid TID_t) (uint64, bool) {
	if data, ok := ts.Get(tid); ok {
		return data.Uint64()
	}
	return 0, false
}

// Uint is unspecified size unsigned int tag getter.
func (ts *Tagset_t) Uint(tid TID_t) (uint, bool) {
	if data, ok := ts.Get(tid); ok {
		return data.Uint()
	}
	return 0, false
}

// Number is 64-bit float tag getter.
func (ts *Tagset_t) Number(tid TID_t) (float64, bool) {
	if data, ok := ts.Get(tid); ok {
		return data.Number()
	}
	return 0, false
}

// Offset returns offset of nested into package file.
func (ts *Tagset_t) Offset() int64 {
	var offset, _ = ts.Uint64(TIDoffset)
	return int64(offset)
}

// Size returns size of nested into package file.
// fs.FileInfo implementation.
func (ts *Tagset_t) Size() int64 {
	var size, _ = ts.Uint64(TIDsize)
	return int64(size)
}

// FID returns file ID.
func (ts *Tagset_t) FID() FID_t {
	var fid, _ = ts.Uint32(TIDfid)
	return FID_t(fid)
}

// Path returns path of nested into package file.
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

// Mode is for fs.FileInfo interface compatibility.
func (ts *Tagset_t) Mode() fs.FileMode {
	if _, ok := ts.Uint32(TIDfid); ok { // file ID is absent for dir
		return 0444
	}
	return fs.ModeDir
}

// ModTime returns file timestamp of nested into package file.
// fs.FileInfo implementation.
func (ts *Tagset_t) ModTime() time.Time {
	var crt, _ = ts.Uint64(TIDcreated)
	return time.Unix(int64(crt), 0)
}

// IsDir detects that object presents a directory. Directory can not have file ID.
// fs.FileInfo implementation.
func (ts *Tagset_t) IsDir() bool {
	var _, ok = ts.Uint32(TIDfid) // file ID is absent for dir
	return !ok
}

// Sys is for fs.FileInfo interface compatibility.
func (ts *Tagset_t) Sys() interface{} {
	return nil
}

// Iterator clones this tagset to iterate through all tags.
func (ts *Tagset_t) Iterator() *TagsetIterator {
	if len(ts.data) < 2 {
		return nil
	}
	var tsi TagsetIterator
	tsi.data = ts.data
	tsi.Reset()
	return &tsi
}

// TagsetIterator helps to iterate through all tags.
type TagsetIterator struct {
	Tagset_t
	num uint16 // number of tags in tagset
	idx uint16 // index of tag to read by "Next" call
	pos uint16 // current position in the slice
	tid TID_t  // tag ID of last readed tag
	tag uint16 // start position of last readed tag content
}

// Reset restarts iterator for new iterations loop.
func (tsi *TagsetIterator) Reset() {
	tsi.num = binary.LittleEndian.Uint16(tsi.data[:2])
	tsi.idx = 0
	tsi.pos = 2
	tsi.tid = TIDnone
	tsi.tag = 0
}

// TID returns the tag ID of the last readed tag.
func (tsi *TagsetIterator) TID() TID_t {
	return tsi.tid
}

// Tag returns tag slice of the last readed tag content.
func (tsi *TagsetIterator) Tag() Tag_t {
	if tsi.tid == TIDnone || tsi.pos > uint16(len(tsi.data)) {
		return nil
	}
	return tsi.data[tsi.tag:tsi.pos]
}

// TagLen returns length of last readed tag content.
func (tsi *TagsetIterator) TagLen() uint16 {
	if tsi.tid == TIDnone {
		return 0
	}
	return tsi.pos - tsi.tag
}

// Passed returns true if the end of iterations is reached.
func (tsi *TagsetIterator) Passed() bool {
	return tsi.idx == tsi.num
}

// Next carries to the next tag position.
func (tsi *TagsetIterator) Next() (ok bool) {
	tsi.tid = TIDnone
	// check up the end of tagset is reached
	if tsi.idx >= tsi.num {
		return
	}
	var tsl = uint16(len(tsi.data))

	// get tag identifier
	if tsi.pos += 2; tsi.pos > tsl {
		return
	}
	tsi.tid = TID_t(binary.LittleEndian.Uint16(tsi.data[tsi.pos-2:]))

	// get tag length
	if tsi.pos += 2; tsi.pos > tsl {
		return
	}
	tsi.tag = tsi.pos // store tag content position
	var len = binary.LittleEndian.Uint16(tsi.data[tsi.pos-2:])

	// prepare to get tag content
	if tsi.pos += len; tsi.pos > tsl {
		return
	}

	tsi.idx++
	ok = true
	return
}

const tsiconst = "content changes are disabled for iterator"

// Put is the stub to disable any changes to data content of iterator.
func (tsi *TagsetIterator) Put(tid TID_t, tag Tag_t) {
	panic(tsiconst)
}

// Set is the stub to disable any changes to data content of iterator.
func (tsi *TagsetIterator) Set(tid TID_t, tag Tag_t) bool {
	panic(tsiconst)
}

// Del is the stub to disable any changes to data content of iterator.
func (tsi *TagsetIterator) Del(tid TID_t) bool {
	panic(tsiconst)
}

// The End.
