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
	Offset(string) (Offset_t, bool)
	Enum(func(string, Offset_t) bool)
	NamedTags(string) (Tagset_t, bool)
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

// Tagmap_t is tags set for each file in package.
type Tagmap_t map[TID_t]Tag_t

// FID returns file ID.
func (ts Tagmap_t) FID() FID_t {
	if data, ok := ts[TIDfid]; ok {
		var fid, _ = data.Uint32()
		return FID_t(fid)
	}
	return 0
}

// Path returns path of nested into package file.
func (ts Tagmap_t) Path() string {
	if data, ok := ts[TIDpath]; ok {
		return string(data)
	}
	return ""
}

// Name returns name of nested into package file.
func (ts Tagmap_t) Name() string {
	if data, ok := ts[TIDpath]; ok {
		return filepath.Base(string(data))
	}
	return ""
}

// Size returns size of nested into package file.
func (ts Tagmap_t) Size() int64 {
	if data, ok := ts[TIDsize]; ok {
		var size, _ = data.Uint64()
		return int64(size)
	}
	return 0
}

// Offset returns offset of nested into package file.
func (ts Tagmap_t) Offset() int64 {
	if data, ok := ts[TIDoffset]; ok {
		var offset, _ = data.Uint64()
		return int64(offset)
	}
	return 0
}

// ReadFrom reads tags set from stream.
func (ts Tagmap_t) ReadFrom(r io.Reader) (n int64, err error) {
	var num, id, l TID_t
	if err = binary.Read(r, binary.LittleEndian, &num); err != nil {
		return
	}
	n += 2
	for i := TID_t(0); i < num; i++ {
		if err = binary.Read(r, binary.LittleEndian, &id); err != nil {
			return
		}
		n += 2
		if err = binary.Read(r, binary.LittleEndian, &l); err != nil {
			return
		}
		n += 2
		var data = make([]byte, l)
		if err = binary.Read(r, binary.LittleEndian, &data); err != nil {
			return
		}
		n += int64(l)
		ts[id] = data
	}
	return
}

// WriteTo writes tags set to stream.
func (ts Tagmap_t) WriteTo(w io.Writer) (n int64, err error) {
	if err = binary.Write(w, binary.LittleEndian, TID_t(len(ts))); err != nil {
		return
	}
	n += 2
	for id, data := range ts {
		if err = binary.Write(w, binary.LittleEndian, id); err != nil {
			return
		}
		n += 2
		if err = binary.Write(w, binary.LittleEndian, TID_t(len(data))); err != nil {
			return
		}
		n += 2
		if err = binary.Write(w, binary.LittleEndian, data); err != nil {
			return
		}
		n += int64(len(data))
	}
	return
}

// Tagset_t is slice of bytes with tags set. Length of slice can be
// not determined to record end, i.e. slice starts at record beginning
// (at number of tags), and can continues after record end.
// fs.FileInfo interface implementation.
type Tagset_t struct {
	Data []byte
}

// Num returns number of tags in tags set.
func (ts *Tagset_t) Num() uint16 {
	if len(ts.Data) < 2 {
		return 0
	}
	return binary.LittleEndian.Uint16(ts.Data[:2])
}

// GetPos returns position of tag with given identifier.
// If tag is not found, returns ErrNoTag.
// If slice content is broken, returns io.EOF.
func (ts *Tagset_t) Pos(tid TID_t) (pos uint16, size uint16, err error) {
	var tsl = uint16(len(ts.Data))
	err = io.EOF
	if pos+2 > tsl {
		return
	}
	var num = binary.LittleEndian.Uint16(ts.Data[pos : pos+2])
	pos += 2
	for i := uint16(0); i < num; i++ {
		if pos+2 > tsl {
			return
		}
		var id = TID_t(binary.LittleEndian.Uint16(ts.Data[pos : pos+2]))
		pos += 2
		if pos+2 > tsl {
			return
		}
		size = binary.LittleEndian.Uint16(ts.Data[pos : pos+2])
		pos += 2
		if pos+size > tsl {
			return
		}
		if id == tid {
			err = nil
			return
		}
		pos += size
	}
	err = ErrNoTag
	return
}

// GetTag returns Tag_t with given identifier.
// If tag is not found, returns ErrNoTag.
// If slice content is broken, returns io.EOF.
func (ts *Tagset_t) Get(tid TID_t) (Tag_t, error) {
	var pos, size, err = ts.Pos(tid)
	if err != nil {
		return nil, err
	}
	return Tag_t(ts.Data[pos : pos+size]), nil
}

// Put appends new tag to tags set.
func (ts *Tagset_t) Put(tid TID_t, tag Tag_t) {
	var num uint16
	if len(ts.Data) >= 2 {
		num = binary.LittleEndian.Uint16(ts.Data[:2])
		num++
		binary.LittleEndian.PutUint16(ts.Data[:2], num)
	} else {
		ts.Data = []byte{0, 1}
	}
	var size [2]byte
	binary.LittleEndian.PutUint16(size[:], uint16(len(tag)))
	ts.Data = append(ts.Data, size[:]...)
	ts.Data = append(ts.Data, tag...)
}

// Set replaces tag with given ID and equal size,
// or appends it to tags set.
func (ts *Tagset_t) Set(tid TID_t, tag Tag_t) (err error) {
	var pos, size uint16
	if pos, size, err = ts.Pos(tid); err != nil {
		if err != ErrNoTag {
			return
		}
		ts.Put(tid, tag)
		err = nil
		return
	}
	var tl = uint16(len(tag))
	if tl == size {
		copy(ts.Data[pos:pos+size], tag)
	} else {
		binary.LittleEndian.PutUint16(ts.Data[pos-2:pos], tl)
		var suff = ts.Data[pos+size:]
		ts.Data = append(ts.Data[:pos], tag...)
		ts.Data = append(ts.Data, suff...)
	}
	return
}

// Del deletes tag with given ID.
func (ts *Tagset_t) Del(tid TID_t) (err error) {
	var pos, size uint16
	if pos, size, err = ts.Pos(tid); err != nil {
		return
	}
	ts.Data = append(ts.Data[:pos-2], ts.Data[pos+size:]...)
	return
}

// String tag getter.
func (ts *Tagset_t) String(tid TID_t) (string, bool) {
	if data, err := ts.Get(tid); err == nil {
		return data.String()
	}
	return "", false
}

// Bool is boolean tag getter.
func (ts *Tagset_t) Bool(tid TID_t) (bool, bool) {
	if data, err := ts.Get(tid); err == nil {
		return data.Bool()
	}
	return false, false
}

// Byte tag getter.
func (ts *Tagset_t) Byte(tid TID_t) (byte, bool) {
	if data, err := ts.Get(tid); err == nil {
		return data.Byte()
	}
	return 0, false
}

// Uint16 is 16-bit unsigned int tag getter.
// Conversion can be used to get signed 16-bit integers.
func (ts *Tagset_t) Uint16(tid TID_t) (TID_t, bool) {
	if data, err := ts.Get(tid); err == nil {
		return data.Uint16()
	}
	return 0, false
}

// Uint32 is 32-bit unsigned int tag getter.
// Conversion can be used to get signed 32-bit integers.
func (ts *Tagset_t) Uint32(tid TID_t) (uint32, bool) {
	if data, err := ts.Get(tid); err == nil {
		return data.Uint32()
	}
	return 0, false
}

// Uint64 is 64-bit unsigned int tag getter.
// Conversion can be used to get signed 64-bit integers.
func (ts *Tagset_t) Uint64(tid TID_t) (uint64, bool) {
	if data, err := ts.Get(tid); err == nil {
		return data.Uint64()
	}
	return 0, false
}

// Uint is unspecified size unsigned int tag getter.
func (ts *Tagset_t) Uint(tid TID_t) (uint, bool) {
	if data, err := ts.Get(tid); err == nil {
		return data.Uint()
	}
	return 0, false
}

// Number is 64-bit float tag getter.
func (ts *Tagset_t) Number(tid TID_t) (float64, bool) {
	if data, err := ts.Get(tid); err == nil {
		return data.Number()
	}
	return 0, false
}

// FID returns file ID.
func (ts *Tagset_t) FID() FID_t {
	var fid, _ = ts.Uint32(TIDfid)
	return FID_t(fid)
}

// Path returns path of nested into package file.
func (ts *Tagset_t) Path() string {
	var kpath, _ = ts.String(TIDpath)
	return kpath
}

// Name returns base name of nested into package file.
// fs.FileInfo implementation.
func (ts *Tagset_t) Name() string {
	var kpath, _ = ts.String(TIDpath)
	return filepath.Base(kpath)
}

// Size returns size of nested into package file.
// fs.FileInfo implementation.
func (ts *Tagset_t) Size() int64 {
	var size, _ = ts.Uint64(TIDsize)
	return int64(size)
}

// Offset returns offset of nested into package file.
func (ts *Tagset_t) Offset() int64 {
	var offset, _ = ts.Uint64(TIDoffset)
	return int64(offset)
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
func (ts *Tagset_t) Iterator() TagsetIterator {
	return TagsetIterator{
		Tagset_t: *ts,
	}
}

// TagsetIterator helps to iterate through all tags.
type TagsetIterator struct {
	Tagset_t
	pos uint16
	tid TID_t
	len uint16
}

// TID returns the tag ID of the current element pointed to by position.
func (ts *TagsetIterator) TID() TID_t {
	return ts.tid
}

// Num returns number of tags in tags set.
func (ts *TagsetIterator) Num() (n uint16) {
	if len(ts.Data) < 2 {
		return
	}
	n = binary.LittleEndian.Uint16(ts.Data[:2])
	ts.pos += 2
	return
}

// Get returns tag with given ID using iterator.
func (ts *TagsetIterator) Get(tid TID_t) (Tag_t, error) {
	ts.pos = 0
	var n = ts.Num()
	for i := uint16(0); i < n && ts.Next() && ts.tid != tid; i++ {
	}
	if ts.pos > uint16(len(ts.Data)) {
		return nil, io.EOF
	}
	if ts.tid != tid {
		return nil, ErrNoTag
	}
	return Tag_t(ts.Data[ts.pos : ts.pos+ts.len]), nil
}

// Next carries to the next tag position.
func (ts *TagsetIterator) Next() (ok bool) {
	if ts.pos < 2 {
		ts.pos = 2
	}

	var tsl = uint16(len(ts.Data))

	if ts.pos += ts.len; ts.pos > tsl {
		return
	}

	if ts.pos += 2; ts.pos > tsl {
		return
	}
	ts.tid = TID_t(binary.LittleEndian.Uint16(ts.Data[ts.pos-2:]))

	if ts.pos += 2; ts.pos > tsl {
		return
	}
	ts.len = binary.LittleEndian.Uint16(ts.Data[ts.pos-2:])

	ok = true
	return
}

// Extract returns tag on which current position is pointed to.
func (ts *TagsetIterator) Extract() (tid TID_t, tag Tag_t) {
	if ts.pos < 2 || ts.pos+ts.len > uint16(len(ts.Data)) {
		return 0, nil
	}
	return ts.tid, ts.Data[ts.pos : ts.pos+ts.len]
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
	tom sync.Map // keys - package filenames in lower case, values - tags slices offsets.
}

// Offset returns offset in package of file with given filename.
func (pack *Package) Offset(fname string) (Offset_t, bool) {
	var val, ok = pack.tom.Load(fname)
	var ret = val.(Offset_t)
	return ret, ok
}

// Enum calls given closure for each file in package.
func (pack *Package) Enum(f func(string, Offset_t) bool) {
	pack.tom.Range(func(key, value interface{}) bool {
		return f(key.(string), value.(Offset_t))
	})
}

// Glob returns the names of all files in package matching pattern or nil
// if there is no matching file.
func (pack *Package) Glob(pattern string) (res []string, err error) {
	pattern = Normalize(pattern)
	if _, err = filepath.Match(pattern, ""); err != nil {
		return
	}
	pack.Enum(func(key string, offset Offset_t) bool {
		if matched, _ := filepath.Match(pattern, key); matched {
			res = append(res, key)
		}
		return true
	})
	return
}

// Opens package for reading. At first its checkup file signature, then
// reads records table, and reads file tags set table. Tags set for each
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

	// read file tags table
	if _, err = r.Seek(int64(pack.fttoffset), io.SeekStart); err != nil {
		return
	}
	var n int64
	for {
		var tagpos int64
		if tagpos, err = r.Seek(0, io.SeekCurrent); err != nil {
			return
		}
		var tm = Tagmap_t{}
		if n, err = tm.ReadFrom(r); err != nil {
			return
		}
		if n == 2 {
			break // end marker was readed
		}

		// check tags fields
		if _, ok := tm[TIDpath]; !ok {
			return &ErrTag{ErrNoPath, "", TIDpath}
		}
		var key = Normalize(tm.Path())
		if _, ok := pack.tom.Load(key); ok {
			return &ErrTag{fs.ErrExist, key, TIDpath}
		}

		if _, ok := tm[TIDfid]; !ok {
			return &ErrTag{ErrNoFID, key, TIDfid}
		}

		if _, ok := tm[TIDoffset]; !ok {
			return &ErrTag{ErrNoOffset, key, TIDoffset}
		}
		if _, ok := tm[TIDsize]; !ok {
			return &ErrTag{ErrNoSize, key, TIDsize}
		}
		var offset, size = tm.Offset(), tm.Size()
		if offset < HeaderSize || offset >= int64(pack.fttoffset) {
			return &ErrTag{ErrOutOff, key, TIDoffset}
		}
		if offset+size > int64(pack.fttoffset) {
			return &ErrTag{ErrOutSize, key, TIDsize}
		}

		// insert file tags
		pack.tom.Store(key, Offset_t(tagpos)-pack.fttoffset)
	}
	return
}

// The End.
