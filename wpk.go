package wpk

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"math"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// File format signatures.
const (
	Signature = "Whirlwind 3.3 Package   " // package is ready for use
	Prebuild  = "Whirlwind 3.3 Prebuild  " // package is in building progress
)

type (
	// TID_t - tag identifier type.
	TID_t uint16
	// FID_t - file index/identifier type.
	FID_t uint32
	// Offset_t - data block offset type.
	Offset_t uint64
	// Size_t - data block size type.
	Size_t uint64
)

// List of predefined tags IDs.
const (
	TIDnone TID_t = 0xffff

	TIDfid        TID_t = 0 // required, uint32
	TIDoffset     TID_t = 1 // required, uint64
	TIDsize       TID_t = 2 // required, uint64
	TIDpath       TID_t = 3 // required, unique, string
	TIDcreated    TID_t = 4 // required for files, uint64
	TIDlastwrite  TID_t = 5 // uint64
	TIDlastaccess TID_t = 6 // uint64
	TIDchange     TID_t = 7 // uint64
	TIDfileattr   TID_t = 8 // uint32

	TIDsys TID_t = 10 // system protection marker

	TIDcrc32ieee TID_t = 10 // uint32, CRC-32-IEEE 802.3, poly = 0x04C11DB7, init = -1
	TIDcrc32c    TID_t = 11 // uint32, (Castagnoli), poly = 0x1EDC6F41, init = -1
	TIDcrc32k    TID_t = 12 // uint32, (Koopman), poly = 0x741B8CD7, init = -1
	TIDcrc64iso  TID_t = 14 // uint64, poly = 0xD800000000000000, init = -1

	TIDmd5    TID_t = 20 // [16]byte
	TIDsha1   TID_t = 21 // [20]byte
	TIDsha224 TID_t = 22 // [28]byte
	TIDsha256 TID_t = 23 // [32]byte
	TIDsha384 TID_t = 24 // [48]byte
	TIDsha512 TID_t = 25 // [64]byte

	TIDmime     TID_t = 100 // string
	TIDlink     TID_t = 101 // string
	TIDkeywords TID_t = 102 // string
	TIDcategory TID_t = 103 // string
	TIDversion  TID_t = 104 // string
	TIDauthor   TID_t = 105 // string
	TIDcomment  TID_t = 106 // string
)

// ErrTag is error on some field of tags set.
type ErrTag struct {
	What error  // error message
	Key  string // normalized file name
	TID  TID_t  // tag ID
}

func (e *ErrTag) Error() string {
	return fmt.Sprintf("key '%s', tag ID %d: %s", e.Key, e.TID, e.What)
}

func (e *ErrTag) Unwrap() error {
	return e.What
}

// Errors on WPK-API.
var (
	ErrSignPre = errors.New("package is not ready")
	ErrSignBad = errors.New("signature does not pass")

	ErrCorrupt  = errors.New("file tags table is corrupted")
	ErrNoData   = errors.New("data is absent")
	ErrNoTag    = errors.New("tag with given ID not found")
	ErrNoPath   = errors.New("file name is absent")
	ErrNoFID    = errors.New("file ID is absent")
	ErrNoOffset = errors.New("file offset is absent")
	ErrOutOff   = errors.New("file offset is out of bounds")
	ErrNoSize   = errors.New("file size is absent")
	ErrOutSize  = errors.New("file size is out of bounds")
)

// FileReader is interface for nested package files access.
type FileReader interface {
	io.Reader
	io.ReaderAt
	io.Seeker
	Size() int64
}

// NestedFile is interface for access to nested into package files.
type NestedFile interface {
	fs.File
	FileReader
}

// Tagger provides file tags access.
type Tagger interface {
	Tagset(string) (*Tagset_t, bool)
	Enum(func(string, *Tagset_t) bool)
}

// Packager refers to package data access management implementation.
type Packager interface {
	DataSize() Size_t
	Tagger

	OpenTags(Tagset_t) (NestedFile, error)
	io.Closer
	fs.SubFS
	fs.StatFS
	fs.GlobFS
	fs.ReadFileFS
	fs.ReadDirFS
}

const (
	// HeaderSize - package header size in bytes.
	HeaderSize = 64
	// SignSize - signature field size.
	SignSize = 0x18
	// LabelSize - disk label field size.
	LabelSize = 0x18
)

// Header - package header.
type Header struct {
	signature [SignSize]byte
	disklabel [LabelSize]byte
	fttoffset Offset_t // file tags table offset
	fttsize   Size_t   // file tags table size

	mux sync.RWMutex
}

// Label returns string with disk label, copied from header fixed field.
// Maximum length of label is 24 bytes.
func (pack *Header) Label() string {
	pack.mux.RLock()
	defer pack.mux.RUnlock()

	var i int
	for ; i < LabelSize && pack.disklabel[i] > 0; i++ {
	}
	return string(pack.disklabel[:i])
}

// SetLabel setups header fixed label field to given string.
// Maximum length of label is 24 bytes.
func (pack *Header) SetLabel(label string) {
	pack.mux.Lock()
	defer pack.mux.Unlock()

	for i := copy(pack.disklabel[:], []byte(label)); i < LabelSize; i++ {
		pack.disklabel[i] = 0 // make label zero-terminated
	}
}

// FTTOffset returns file tags table offset in the package.
func (pack *Header) FTTOffset() Offset_t {
	pack.mux.RLock()
	defer pack.mux.RUnlock()

	return pack.fttoffset
}

// FTTSize returns file tags table size in the package.
func (pack *Header) FTTSize() Size_t {
	pack.mux.RLock()
	defer pack.mux.RUnlock()

	return pack.fttsize
}

// DataSize returns sum size of all real stored records in package.
func (pack *Header) DataSize() Size_t {
	pack.mux.RLock()
	defer pack.mux.RUnlock()

	if pack.fttoffset > HeaderSize {
		return Size_t(pack.fttoffset - HeaderSize)
	}
	return 0
}

// IsReady determines that package is ready for read the data.
func (pack *Header) IsReady() (err error) {
	pack.mux.RLock()
	defer pack.mux.RUnlock()

	if string(pack.signature[:]) == Prebuild {
		return ErrSignPre
	}
	if string(pack.signature[:]) != Signature {
		return ErrSignBad
	}
	return
}

// ReadFrom reads header from stream as binary data of constant length in little endian order.
func (pack *Header) ReadFrom(r io.Reader) (n int64, err error) {
	pack.mux.Lock()
	defer pack.mux.Unlock()

	if err = binary.Read(r, binary.LittleEndian, pack.signature[:]); err != nil {
		return
	}
	n += SignSize
	if err = binary.Read(r, binary.LittleEndian, pack.disklabel[:]); err != nil {
		return
	}
	n += LabelSize
	if err = binary.Read(r, binary.LittleEndian, &pack.fttoffset); err != nil {
		return
	}
	n += 8
	if err = binary.Read(r, binary.LittleEndian, &pack.fttsize); err != nil {
		return
	}
	n += 8
	return
}

// WriteTo writes header to stream as binary data of constant length in little endian order.
func (pack *Header) WriteTo(w io.Writer) (n int64, err error) {
	pack.mux.RLock()
	defer pack.mux.RUnlock()

	if err = binary.Write(w, binary.LittleEndian, pack.signature[:]); err != nil {
		return
	}
	n += SignSize
	if err = binary.Write(w, binary.LittleEndian, pack.disklabel[:]); err != nil {
		return
	}
	n += LabelSize
	if err = binary.Write(w, binary.LittleEndian, &pack.fttoffset); err != nil {
		return
	}
	n += 8
	if err = binary.Write(w, binary.LittleEndian, &pack.fttsize); err != nil {
		return
	}
	n += 8
	return
}

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

// FID returns file ID.
func (ts *Tagset_t) FID() FID_t {
	var fid, _ = ts.Uint32(TIDfid)
	return FID_t(fid)
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
	return &TagsetIterator{
		Tagset_t: *ts,
		num:      binary.LittleEndian.Uint16(ts.data[:2]),
		idx:      0,
		pos:      2,
		tid:      TIDnone,
	}
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

// DirEntry is directory representation of nested into package files.
// No any reader for directory implementation.
// fs.DirEntry interface implementation.
type DirEntry struct {
	Tagset_t // has fs.FileInfo interface
}

// Type is for fs.DirEntry interface compatibility.
func (f *DirEntry) Type() fs.FileMode {
	if _, ok := f.Uint32(TIDfid); ok { // file ID is absent for dir
		return 0444
	}
	return fs.ModeDir
}

// Info returns the FileInfo for the file or subdirectory described by the entry.
func (f *DirEntry) Info() (fs.FileInfo, error) {
	return &f.Tagset_t, nil
}

// ReadDirFile is a directory file whose entries can be read with the ReadDir method.
// fs.ReadDirFile interface implementation.
type ReadDirFile struct {
	Tagset_t // has fs.FileInfo interface
	Pack     Tagger
}

// Stat is for fs.ReadDirFile interface compatibility.
func (f *ReadDirFile) Stat() (fs.FileInfo, error) {
	return &f.Tagset_t, nil
}

// Read is for fs.ReadDirFile interface compatibility.
func (f *ReadDirFile) Read(b []byte) (n int, err error) {
	return 0, io.EOF
}

// Close is for fs.ReadDirFile interface compatibility.
func (f *ReadDirFile) Close() error {
	return nil
}

// ReadDir returns fs.FileInfo array with nested into given package directory presentation.
func (f *ReadDirFile) ReadDir(n int) (matches []fs.DirEntry, err error) {
	return ReadDir(f.Pack, strings.TrimSuffix(f.Path(), "/"), n)
}

// Package structure contains all data needed for package representation.
type Package struct {
	Header
	// File tags table.
	// Keys - package filenames in lower case, values - tagset slices.
	ftt sync.Map
}

// Tagset returns offset in package of file with given filename.
func (pack *Package) Tagset(fkey string) (ts *Tagset_t, ok bool) {
	var val interface{}
	if val, ok = pack.ftt.Load(fkey); ok {
		ts = val.(*Tagset_t)
	}
	return
}

// Enum calls given closure for each file in package.
func (pack *Package) Enum(f func(string, *Tagset_t) bool) {
	pack.ftt.Range(func(key, value interface{}) bool {
		return f(key.(string), value.(*Tagset_t))
	})
}

// Glob returns the names of all files in package matching pattern or nil
// if there is no matching file.
func (pack *Package) Glob(pattern string) (res []string, err error) {
	pattern = Normalize(pattern)
	if _, err = filepath.Match(pattern, ""); err != nil {
		return
	}
	pack.Enum(func(fkey string, ts *Tagset_t) bool {
		if matched, _ := filepath.Match(pattern, fkey); matched {
			res = append(res, fkey)
		}
		return true
	})
	return
}

// Opens package for reading. At first its checkup file signature, then
// reads records table, and reads file tagset table. Tags set for each
// file must contain at least file ID, file name and creation time.
func (pack *Package) Read(r io.ReadSeeker) (err error) {
	// go to file start
	if _, err = r.Seek(0, io.SeekStart); err != nil {
		return
	}
	// read header
	if _, err = pack.Header.ReadFrom(r); err != nil {
		return
	}
	if err = pack.Header.IsReady(); err != nil {
		return
	}

	// setup empty tags table
	pack.ftt = sync.Map{}

	// read file tags table
	if _, err = r.Seek(int64(pack.fttoffset), io.SeekStart); err != nil {
		return
	}
	var fttbulk = make([]byte, pack.fttsize)
	if _, err = r.Read(fttbulk); err != nil {
		return
	}

	var tspos int64
	for {
		var data = fttbulk[tspos:]
		var dl = uint16(len(data))
		if dl < 2 {
			return io.EOF
		}
		var tsi = TagsetIterator{
			Tagset_t: Tagset_t{data},
			num:      binary.LittleEndian.Uint16(data[:2]),
			idx:      0,
			pos:      2,
			tid:      TIDnone,
		}
		if tsi.num == 0 {
			if dl > 2 {
				return ErrCorrupt
			}
			break // end marker was reached
		}
		for tsi.Next() {
		}
		if tsi.pos > dl {
			return io.EOF
		}

		var ts = NewTagset(data[:tsi.pos])

		// get file key and check tags fields
		var (
			ok           bool
			fkey, fpath  string
			offset, size uint64
		)
		if fpath, ok = ts.String(TIDpath); !ok {
			return &ErrTag{ErrNoPath, "", TIDpath}
		}
		fkey = Normalize(fpath)
		if _, ok = pack.ftt.Load(fkey); ok {
			return &ErrTag{fs.ErrExist, fkey, TIDpath}
		}
		if _, ok = ts.Uint32(TIDfid); !ok {
			return &ErrTag{ErrNoFID, fkey, TIDfid}
		}
		if offset, ok = ts.Uint64(TIDoffset); !ok {
			return &ErrTag{ErrNoOffset, fkey, TIDoffset}
		}
		if size, ok = ts.Uint64(TIDsize); !ok {
			return &ErrTag{ErrNoSize, fkey, TIDsize}
		}
		if offset < HeaderSize || offset >= uint64(pack.fttoffset) {
			return &ErrTag{ErrOutOff, fkey, TIDoffset}
		}
		if offset+size > uint64(pack.fttoffset) {
			return &ErrTag{ErrOutSize, fkey, TIDsize}
		}

		tspos += int64(tsi.pos)
		pack.ftt.Store(fkey, ts)
	}
	return
}

// The End.
