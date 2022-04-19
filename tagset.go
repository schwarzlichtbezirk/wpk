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

// MakeTagset returns tagset with given slice.
func MakeTagset(data []byte) *Tagset_t {
	return &Tagset_t{data}
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
func (ts *Tagset_t) Has(tid TID_t) bool {
	var tsi = ts.Iterator()
	for tsi.Next() && tsi.tid != tid {
	}
	return tsi.tid == tid
}

// GetTag returns Tag_t with given identifier.
// If tag is not found, slice content is broken,
// returns false.
func (ts *Tagset_t) Get(tid TID_t) (Tag_t, bool) {
	var tsi = ts.Iterator()
	for tsi.Next() && tsi.tid != tid {
	}
	if tsi.Failed() {
		return nil, false // io.EOF
	}
	if tsi.tid != tid {
		return nil, false // ErrNoTag
	}
	return Tag_t(tsi.data[tsi.tag:tsi.pos]), true
}

// Put appends new tag to tagset.
func (ts *Tagset_t) Put(tid TID_t, tag Tag_t) {
	if tid == TIDnone { // special case
		return
	}

	var buf [TID_l + TID_l]byte
	binary.LittleEndian.PutUint16(buf[0:TID_l], uint16(tid))
	binary.LittleEndian.PutUint16(buf[TID_l:TID_l+TID_l], uint16(len(tag)))
	ts.data = append(ts.data, buf[:]...)
	ts.data = append(ts.data, tag...)
}

// Set replaces tag with given ID and equal size, or
// appends it to tagset. Returns true if new one added.
func (ts *Tagset_t) Set(tid TID_t, tag Tag_t) bool {
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

	var tl = len(tag)
	if TSSize_t(tl) == tsi.pos-tsi.tag {
		copy(ts.data[tsi.tag:tsi.pos], tag)
	} else {
		binary.LittleEndian.PutUint16(ts.data[tsi.tag-TID_l:tsi.tag], uint16(tl)) // set tag length
		var suff = ts.data[tsi.pos:]
		ts.data = append(ts.data[:tsi.tag], tag...)
		ts.data = append(ts.data, suff...)
	}
	return false
}

// Del deletes tag with given ID.
func (ts *Tagset_t) Del(tid TID_t) bool {
	if tid == TIDnone { // special case
		return false
	}

	var tsi = ts.Iterator()
	for tsi.Next() && tsi.tid != tid {
	}
	if tsi.tid != tid {
		return false // ErrNoTag
	}
	ts.data = append(ts.data[:tsi.tag-TID_l-TID_l], ts.data[tsi.pos:]...)
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
func (ts *Tagset_t) Uint16(tid TID_t) (uint16, bool) {
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
	var tsi TagsetIterator
	tsi.data = ts.data
	return &tsi
}

// TagsetIterator helps to iterate through all tags.
type TagsetIterator struct {
	Tagset_t
	tid TID_t    // tag ID of last readed tag
	pos TSSize_t // current position in the slice
	tag TSSize_t // start position of last readed tag content
}

// Reset restarts iterator for new iterations loop.
func (tsi *TagsetIterator) Reset() {
	tsi.tid = TIDnone
	tsi.pos = 0
	tsi.tag = 0
}

// TID returns the tag ID of the last readed tag.
func (tsi *TagsetIterator) TID() TID_t {
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
func (tsi *TagsetIterator) TagLen() TSSize_t {
	if tsi.Failed() {
		return 0
	}
	return tsi.pos - tsi.tag
}

// Passed returns true if the end of iterations is successfully reached.
func (tsi *TagsetIterator) Passed() bool {
	return tsi.pos == TSSize_t(len(tsi.data))
}

// Failed points that iterator is finished by broken tagset state.
func (tsi *TagsetIterator) Failed() bool {
	return tsi.pos > TSSize_t(len(tsi.data))
}

// Next carries to the next tag position.
func (tsi *TagsetIterator) Next() (ok bool) {
	var tsl = TSSize_t(len(tsi.data))
	tsi.tid = TIDnone

	// check up the end of tagset is reached by any reason
	if tsi.pos >= tsl {
		return
	}

	// get tag identifier
	if tsi.pos += TID_l; tsi.pos > tsl {
		return
	}
	var tid = TID_t(binary.LittleEndian.Uint16(tsi.data[tsi.pos-TID_l:]))

	// get tag length
	if tsi.pos += TID_l; tsi.pos > tsl {
		return
	}
	var len = binary.LittleEndian.Uint16(tsi.data[tsi.pos-TID_l:])
	// store tag content position
	var tag = tsi.pos

	// prepare to get tag content
	if tsi.pos += TSSize_t(len); tsi.pos > tsl {
		return
	}

	tsi.tid, tsi.tag = tid, tag
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
