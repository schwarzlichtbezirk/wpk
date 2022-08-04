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

// FOffset is FOffset_t tag converter.
func (t Tag_t) FOffset() (FOffset_t, bool) {
	if len(t) == Uint_l[FOffset_t]() {
		return FOffset_r[FOffset_t](t), true
	}
	return 0, false
}

// TagFOffset FOffset_t tag constructor.
func TagFOffset(val FOffset_t) Tag_t {
	var buf = make([]byte, Uint_l[FOffset_t]())
	FOffset_w(buf, val)
	return buf
}

// FSize is FSize_t tag converter.
func (t Tag_t) FSize() (FSize_t, bool) {
	if len(t) == Uint_l[FSize_t]() {
		return FSize_r[FSize_t](t), true
	}
	return 0, false
}

// TagFSize is 64-bit unsigned int tag constructor.
func TagFSize(val FSize_t) Tag_t {
	var buf = make([]byte, Uint_l[FSize_t]())
	FSize_w(buf, val)
	return buf
}

// FID is FID_t tag converter.
func (t Tag_t) FID() (FID_t, bool) {
	if len(t) == Uint_l[FID_t]() {
		return FID_r[FID_t](t), true
	}
	return 0, false
}

// TagFID is FID_t tag constructor.
func TagFID(val FID_t) Tag_t {
	var buf = make([]byte, Uint_l[FID_t]())
	FID_w(buf, val)
	return buf
}

// Tagset_t is slice of bytes with tags set. Length of slice can be
// not determined to record end, i.e. slice starts at record beginning
// (at number of tags), and can continues after record end.
// fs.FileInfo interface implementation.
type Tagset_t[TID_t TID_i, TSize_t TSize_i] struct {
	data []byte
}

// MakeTagset returns tagset with given slice.
func MakeTagset[TID_t TID_i, TSize_t TSize_i](data []byte) *Tagset_t[TID_t, TSize_t] {
	return &Tagset_t[TID_t, TSize_t]{data}
}

// Data returns whole tagset content.
func (ts *Tagset_t[TID_t, TSize_t]) Data() []byte {
	return ts.data
}

// Num returns number of tags in tagset.
func (ts *Tagset_t[TID_t, TSize_t]) Num() (n int) {
	var tsi = ts.Iterator()
	for tsi.Next() {
		n++
	}
	return
}

// Has checks existence of tag with given ID.
func (ts *Tagset_t[TID_t, TSize_t]) Has(tid TID_t) bool {
	var tsi = ts.Iterator()
	for tsi.Next() && tsi.tid != tid {
	}
	return tsi.tid == tid
}

// GetTag returns Tag_t with given identifier.
// If tag is not found, slice content is broken,
// returns false.
func (ts *Tagset_t[TID_t, TSize_t]) Get(tid TID_t) (Tag_t, bool) {
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
func (ts *Tagset_t[TID_t, TSize_t]) Put(tid TID_t, tag Tag_t) *Tagset_t[TID_t, TSize_t] {
	if tid == TIDnone { // special case
		return ts
	}

	var buf = make([]byte, Uint_l[TID_t]()+Uint_l[TSize_t]())
	TID_w(buf[:Uint_l[TID_t]()], tid)
	TSize_w(buf[Uint_l[TID_t]():], TSize_t(len(tag)))
	ts.data = append(ts.data, buf...)
	ts.data = append(ts.data, tag...)
	return ts
}

// Set replaces tag with given ID and equal size, or
// appends it to tagset. Returns true if new one added.
func (ts *Tagset_t[TID_t, TSize_t]) Set(tid TID_t, tag Tag_t) bool {
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
		TSize_w(ts.data[tsi.tag-Uint_l[TSize_t]():tsi.tag], tl) // set tag length
		var suff = ts.data[tsi.pos:]
		ts.data = append(ts.data[:tsi.tag], tag...)
		ts.data = append(ts.data, suff...)
	}
	return false
}

// Del deletes tag with given ID.
func (ts *Tagset_t[TID_t, TSize_t]) Del(tid TID_t) bool {
	if tid == TIDnone { // special case
		return false
	}

	var tsi = ts.Iterator()
	for tsi.Next() && tsi.tid != tid {
	}
	if tsi.tid != tid {
		return false // ErrNoTag
	}
	ts.data = append(ts.data[:tsi.tag-Uint_l[TSize_t]()-Uint_l[TID_t]()], ts.data[tsi.pos:]...)
	return true
}

// String tag getter.
func (ts *Tagset_t[TID_t, TSize_t]) String(tid TID_t) (string, bool) {
	if data, ok := ts.Get(tid); ok {
		return data.String()
	}
	return "", false
}

// Bool is boolean tag getter.
func (ts *Tagset_t[TID_t, TSize_t]) Bool(tid TID_t) (bool, bool) {
	if data, ok := ts.Get(tid); ok {
		return data.Bool()
	}
	return false, false
}

// Byte tag getter.
func (ts *Tagset_t[TID_t, TSize_t]) Byte(tid TID_t) (byte, bool) {
	if data, ok := ts.Get(tid); ok {
		return data.Byte()
	}
	return 0, false
}

// Uint16 is 16-bit unsigned int tag getter.
// Conversion can be used to get signed 16-bit integers.
func (ts *Tagset_t[TID_t, TSize_t]) Uint16(tid TID_t) (uint16, bool) {
	if data, ok := ts.Get(tid); ok {
		return data.Uint16()
	}
	return 0, false
}

// Uint32 is 32-bit unsigned int tag getter.
// Conversion can be used to get signed 32-bit integers.
func (ts *Tagset_t[TID_t, TSize_t]) Uint32(tid TID_t) (uint32, bool) {
	if data, ok := ts.Get(tid); ok {
		return data.Uint32()
	}
	return 0, false
}

// Uint64 is 64-bit unsigned int tag getter.
// Conversion can be used to get signed 64-bit integers.
func (ts *Tagset_t[TID_t, TSize_t]) Uint64(tid TID_t) (uint64, bool) {
	if data, ok := ts.Get(tid); ok {
		return data.Uint64()
	}
	return 0, false
}

// Uint is unspecified size unsigned int tag getter.
func (ts *Tagset_t[TID_t, TSize_t]) Uint(tid TID_t) (uint, bool) {
	if data, ok := ts.Get(tid); ok {
		return data.Uint()
	}
	return 0, false
}

// Number is 64-bit float tag getter.
func (ts *Tagset_t[TID_t, TSize_t]) Number(tid TID_t) (float64, bool) {
	if data, ok := ts.Get(tid); ok {
		return data.Number()
	}
	return 0, false
}

// Time is time tag getter.
func (ts *Tagset_t[TID_t, TSize_t]) Time(tid TID_t) (time.Time, bool) {
	if data, ok := ts.Get(tid); ok {
		return data.Time()
	}
	return time.Time{}, false
}

// FOffset returns offset of nested into package file.
func (ts *Tagset_t[TID_t, TSize_t]) FOffset() (FOffset_t, bool) {
	if data, ok := ts.Get(TIDoffset); ok && len(data) == Uint_l[FOffset_t]() {
		return FOffset_r[FOffset_t](data), true
	}
	return 0, false
}

// FSize returns size of nested into package file.
// fs.FileInfo implementation.
func (ts *Tagset_t[TID_t, TSize_t]) FSize() (FSize_t, bool) {
	if data, ok := ts.Get(TIDsize); ok && len(data) == Uint_l[FSize_t]() {
		return FSize_r[FSize_t](data), true
	}
	return 0, false
}

// FID returns file ID.
func (ts *Tagset_t[TID_t, TSize_t]) FID() (FID_t, bool) {
	if data, ok := ts.Get(TIDfid); ok && len(data) == Uint_l[FID_t]() {
		return FID_r[FID_t](data), true
	}
	return 0, false
}

// Path returns path of nested into package file.
func (ts *Tagset_t[TID_t, TSize_t]) Path() string {
	var fpath, _ = ts.String(TIDpath)
	return fpath
}

// Name returns base name of nested into package file.
// fs.FileInfo implementation.
func (ts *Tagset_t[TID_t, TSize_t]) Name() string {
	var fpath, _ = ts.String(TIDpath)
	return filepath.Base(fpath)
}

// Size returns size of nested into package file.
// fs.FileInfo implementation.
func (ts *Tagset_t[TID_t, TSize_t]) Size() int64 {
	var size, _ = ts.FSize()
	return int64(size)
}

// Mode is for fs.FileInfo interface compatibility.
func (ts *Tagset_t[TID_t, TSize_t]) Mode() fs.FileMode {
	if ts.Has(TIDsize) { // file size is absent for dir
		return 0444
	}
	return fs.ModeDir
}

// ModTime returns file modification timestamp of nested into package file.
// fs.FileInfo & times.Timespec implementation.
func (ts *Tagset_t[TID_t, TSize_t]) ModTime() time.Time {
	var t, _ = ts.Time(TIDmtime)
	return t
}

// IsDir detects that object presents a directory. Directory can not have file ID.
// fs.FileInfo implementation.
func (ts *Tagset_t[TID_t, TSize_t]) IsDir() bool {
	return !ts.Has(TIDsize) // file size is absent for dir
}

// Sys is for fs.FileInfo interface compatibility.
func (ts *Tagset_t[TID_t, TSize_t]) Sys() interface{} {
	return nil
}

// AccessTime returns file access timestamp of nested into package file.
// times.Timespec implementation.
func (ts *Tagset_t[TID_t, TSize_t]) AccessTime() time.Time {
	var t, _ = ts.Uint64(TIDatime)
	return time.Unix(int64(t), 0)
}

// ChangeTime returns file change timestamp of nested into package file.
// times.Timespec implementation.
func (ts *Tagset_t[TID_t, TSize_t]) ChangeTime() time.Time {
	var t, _ = ts.Uint64(TIDctime)
	return time.Unix(int64(t), 0)
}

// BirthTime returns file access timestamp of nested into package file.
// times.Timespec implementation.
func (ts *Tagset_t[TID_t, TSize_t]) BirthTime() time.Time {
	var t, _ = ts.Uint64(TIDbtime)
	return time.Unix(int64(t), 0)
}

// HasChangeTime is times.Timespec interface implementation.
// Returns whether change timestamp is present.
func (ts *Tagset_t[TID_t, TSize_t]) HasChangeTime() bool {
	return ts.Has(TIDctime)
}

// HasBirthTime is times.Timespec interface implementation.
// Returns whether birth timestamp is present.
func (ts *Tagset_t[TID_t, TSize_t]) HasBirthTime() bool {
	return ts.Has(TIDbtime)
}

// Iterator clones this tagset to iterate through all tags.
func (ts *Tagset_t[TID_t, TSize_t]) Iterator() *TagsetIterator[TID_t, TSize_t] {
	var tsi TagsetIterator[TID_t, TSize_t]
	tsi.data = ts.data
	return &tsi
}

// TagsetIterator helps to iterate through all tags.
type TagsetIterator[TID_t TID_i, TSize_t TSize_i] struct {
	Tagset_t[TID_t, TSize_t]
	tid TID_t // tag ID of last readed tag
	pos int   // current position in the slice
	tag int   // start position of last readed tag content
}

// Reset restarts iterator for new iterations loop.
func (tsi *TagsetIterator[TID_t, TSize_t]) Reset() {
	tsi.tid = TIDnone
	tsi.pos = 0
	tsi.tag = 0
}

// TID returns the tag ID of the last readed tag.
func (tsi *TagsetIterator[TID_t, TSize_t]) TID() TID_t {
	return tsi.tid
}

// Tag returns tag slice of the last readed tag content.
func (tsi *TagsetIterator[TID_t, TSize_t]) Tag() Tag_t {
	if tsi.Failed() {
		return nil
	}
	return tsi.data[tsi.tag:tsi.pos]
}

// TagLen returns length of last readed tag content.
func (tsi *TagsetIterator[TID_t, TSize_t]) TagLen() int {
	if tsi.Failed() {
		return 0
	}
	return tsi.pos - tsi.tag
}

// Passed returns true if the end of iterations is successfully reached.
func (tsi *TagsetIterator[TID_t, TSize_t]) Passed() bool {
	return tsi.pos == len(tsi.data)
}

// Failed points that iterator is finished by broken tagset state.
func (tsi *TagsetIterator[TID_t, TSize_t]) Failed() bool {
	return tsi.pos > len(tsi.data)
}

// Next carries to the next tag position.
func (tsi *TagsetIterator[TID_t, TSize_t]) Next() (ok bool) {
	var tsl = len(tsi.data)
	tsi.tid = TIDnone

	// check up the end of tagset is reached by any reason
	if tsi.pos >= tsl {
		return
	}

	// get tag identifier
	if tsi.pos += Uint_l[TID_t](); tsi.pos > tsl {
		return
	}
	var tid = TID_r[TID_t](tsi.data[tsi.pos-Uint_l[TID_t]():])

	// get tag length
	if tsi.pos += Uint_l[TSize_t](); tsi.pos > tsl {
		return
	}
	var len = TSize_r[TSize_t](tsi.data[tsi.pos-Uint_l[TSize_t]():])
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
func (tsi *TagsetIterator[TID_t, TSize_t]) Put(tid TID_t, tag Tag_t) *Tagset_t[TID_t, TSize_t] {
	panic(tsiconst)
}

// Set is the stub to disable any changes to data content of iterator.
func (tsi *TagsetIterator[TID_t, TSize_t]) Set(tid TID_t, tag Tag_t) bool {
	panic(tsiconst)
}

// Del is the stub to disable any changes to data content of iterator.
func (tsi *TagsetIterator[TID_t, TSize_t]) Del(tid TID_t) bool {
	panic(tsiconst)
}

// The End.
