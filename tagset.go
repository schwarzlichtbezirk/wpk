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

// FOffset is FOffset_t tag converter.
func (t Tag_t) FOffset() (FOffset_t, bool) {
	if len(t) == FOffset_l {
		return FOffset_r(t), true
	}
	return 0, false
}

// TagFOffset FOffset_t tag constructor.
func TagFOffset(val FOffset_t) Tag_t {
	var buf [FOffset_l]byte
	FOffset_w(buf[:], val)
	return buf[:]
}

// FSize is FSize_t tag converter.
func (t Tag_t) FSize() (FSize_t, bool) {
	if len(t) == FSize_l {
		return FSize_r(t), true
	}
	return 0, false
}

// TagFSize is 64-bit unsigned int tag constructor.
func TagFSize(val FSize_t) Tag_t {
	var buf [FSize_l]byte
	FSize_w(buf[:], val)
	return buf[:]
}

// FID is FID_t tag converter.
func (t Tag_t) FID() (FID_t, bool) {
	if len(t) == FID_l {
		return FID_r(t), true
	}
	return 0, false
}

// TagFID is FID_t tag constructor.
func TagFID(val FID_t) Tag_t {
	var buf [FID_l]byte
	FID_w(buf[:], val)
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
		return nil, false // io.ErrUnexpectedEOF
	}
	if tsi.tid != tid {
		return nil, false // ErrNoTag
	}
	return Tag_t(tsi.data[tsi.tag:tsi.pos]), true
}

// Put appends new tag to tagset.
// Can be used in chain calls at initialization.
func (ts *Tagset_t) Put(tid TID_t, tag Tag_t) *Tagset_t {
	if tid == TIDnone { // special case
		return ts
	}

	var buf [TID_l + TSize_l]byte
	TID_w(buf[0:TID_l], tid)
	TSize_w(buf[TID_l:TID_l+TSize_l], TSize_t(len(tag)))
	ts.data = append(ts.data, buf[:]...)
	ts.data = append(ts.data, tag...)
	return ts
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

	var tl = TSize_t(len(tag))
	if tl == TSize_t(tsi.pos-tsi.tag) {
		copy(ts.data[tsi.tag:tsi.pos], tag)
	} else {
		TSize_w(ts.data[tsi.tag-TSize_l:tsi.tag], tl) // set tag length
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
	ts.data = append(ts.data[:tsi.tag-TSize_l-TID_l], ts.data[tsi.pos:]...)
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

// FOffset returns offset of nested into package file.
func (ts *Tagset_t) FOffset() (FOffset_t, bool) {
	if data, ok := ts.Get(TIDoffset); ok && len(data) == FOffset_l {
		return FOffset_r(data), true
	}
	return 0, false
}

// FSize returns size of nested into package file.
// fs.FileInfo implementation.
func (ts *Tagset_t) FSize() (FSize_t, bool) {
	if data, ok := ts.Get(TIDsize); ok && len(data) == FSize_l {
		return FSize_r(data), true
	}
	return 0, false
}

// FID returns file ID.
func (ts *Tagset_t) FID() (FID_t, bool) {
	if data, ok := ts.Get(TIDfid); ok && len(data) == FID_l {
		return FID_r(data), true
	}
	return 0, false
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

// Size returns size of nested into package file.
// fs.FileInfo implementation.
func (ts *Tagset_t) Size() int64 {
	var size, _ = ts.FSize()
	return int64(size)
}

// Mode is for fs.FileInfo interface compatibility.
func (ts *Tagset_t) Mode() fs.FileMode {
	if ts.Has(TIDfid) { // file ID is absent for dir
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
	return !ts.Has(TIDfid) // file ID is absent for dir
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
	var tid = TID_r(tsi.data[tsi.pos-TID_l:])

	// get tag length
	if tsi.pos += TSize_l; tsi.pos > tsl {
		return
	}
	var len = TSize_r(tsi.data[tsi.pos-TSize_l:])
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
func (tsi *TagsetIterator) Put(tid TID_t, tag Tag_t) *Tagset_t {
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
