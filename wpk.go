package wpk

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"math"
	"path"
	"path/filepath"
	"strings"
	"time"
)

// File format signatures.
const (
	Signature = "Whirlwind 3.2 Package   " // package is ready for use
	Prebuild  = "Whirlwind 3.2 Prebuild  " // package is in building progress
)

type (
	// TID - tag identifier.
	TID uint16
	// FID - file index/identifier.
	FID uint32
	// OFFSET - data block offset.
	OFFSET uint64
	// SIZE - data block size.
	SIZE uint64
)

// List of predefined tags IDs.
const (
	TIDfid        TID = 0 // required, uint32
	TIDoffset     TID = 1 // required, uint64
	TIDsize       TID = 2 // required, uint64
	TIDpath       TID = 3 // required, unique, string
	TIDcreated    TID = 4 // required for files, uint64
	TIDlastwrite  TID = 5 // uint64
	TIDlastaccess TID = 6 // uint64
	TIDchange     TID = 7 // uint64
	TIDfileattr   TID = 8 // uint32

	TIDsys TID = 10 // system protection marker

	TIDcrc32ieee TID = 10 // uint32, CRC-32-IEEE 802.3, poly = 0x04C11DB7, init = -1
	TIDcrc32c    TID = 11 // uint32, (Castagnoli), poly = 0x1EDC6F41, init = -1
	TIDcrc32k    TID = 12 // uint32, (Koopman), poly = 0x741B8CD7, init = -1
	TIDcrc64iso  TID = 14 // uint64, poly = 0xD800000000000000, init = -1

	TIDmd5    TID = 20 // [16]byte
	TIDsha1   TID = 21 // [20]byte
	TIDsha224 TID = 22 // [28]byte
	TIDsha256 TID = 23 // [32]byte
	TIDsha384 TID = 24 // [48]byte
	TIDsha512 TID = 25 // [64]byte

	TIDmime     TID = 100 // string
	TIDlink     TID = 101 // string
	TIDkeywords TID = 102 // string
	TIDcategory TID = 103 // string
	TIDversion  TID = 104 // string
	TIDauthor   TID = 105 // string
	TIDcomment  TID = 106 // string
)

// ErrTag is error on some field of tags set.
type ErrTag struct {
	What error  // error message
	Key  string // normalized file name
	TID  TID    // tag ID
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
	ErrOutFID   = errors.New("file ID is out of range")
	ErrNoOffset = errors.New("file offset is absent")
	ErrOutOff   = errors.New("file offset is out of bounds")
	ErrNoSize   = errors.New("file size is absent")
	ErrOutSize  = errors.New("file size is out of bounds")
)

// FTTMap is tags files tags table map.
type FTTMap map[string]OFFSET

// Tagger provides tags set access.
type Tagger interface {
	Enum() FTTMap
	NamedTags(string) (TagSlice, bool)
}

// Packager refers to package data access management implementation.
type Packager interface {
	OpenWPK(string) error
	io.Closer
	fs.SubFS
	fs.StatFS
	fs.GlobFS
	fs.ReadFileFS
	fs.ReadDirFS
	Tagger
}

// PackHdr - package header.
type PackHdr struct {
	Signature [0x18]byte `json:"signature"`
	TagOffset OFFSET     `json:"tagoffset"` // tags table offset
	RecNumber FID        `json:"recnumber"` // number of records
	TagNumber FID        `json:"tagnumber"` // number of tagset entries
}

// PackHdrSize - package header length.
const PackHdrSize = 40

// Tag - file description item.
type Tag []byte

// String tag converter.
func (t Tag) String() (string, bool) {
	return string(t), true
}

// TagString is string tag constructor.
func TagString(val string) Tag {
	return Tag(val)
}

// Bool is boolean tag converter.
func (t Tag) Bool() (bool, bool) {
	if len(t) == 1 {
		return t[0] > 0, true
	}
	return false, false
}

// TagBool is boolean tag constructor.
func TagBool(val bool) Tag {
	var buf [1]byte
	if val {
		buf[0] = 1
	}
	return buf[:]
}

// Byte tag converter.
func (t Tag) Byte() (byte, bool) {
	if len(t) == 1 {
		return t[0], true
	}
	return 0, false
}

// TagByte is Byte tag constructor.
func TagByte(val byte) Tag {
	var buf = [1]byte{val}
	return buf[:]
}

// Uint16 is 16-bit unsigned int tag converter.
func (t Tag) Uint16() (TID, bool) {
	if len(t) == 2 {
		return TID(binary.LittleEndian.Uint16(t)), true
	}
	return 0, false
}

// TagUint16 is 16-bit unsigned int tag constructor.
func TagUint16(val TID) Tag {
	var buf [2]byte
	binary.LittleEndian.PutUint16(buf[:], uint16(val))
	return buf[:]
}

// Uint32 is 32-bit unsigned int tag converter.
func (t Tag) Uint32() (uint32, bool) {
	if len(t) == 4 {
		return binary.LittleEndian.Uint32(t), true
	}
	return 0, false
}

// TagUint32 is 32-bit unsigned int tag constructor.
func TagUint32(val uint32) Tag {
	var buf [4]byte
	binary.LittleEndian.PutUint32(buf[:], val)
	return buf[:]
}

// Uint64 is 64-bit unsigned int tag converter.
func (t Tag) Uint64() (uint64, bool) {
	if len(t) == 8 {
		return binary.LittleEndian.Uint64(t), true
	}
	return 0, false
}

// TagUint64 is 64-bit unsigned int tag constructor.
func TagUint64(val uint64) Tag {
	var buf [8]byte
	binary.LittleEndian.PutUint64(buf[:], val)
	return buf[:]
}

// Uint is unspecified size unsigned int tag converter.
func (t Tag) Uint() (uint, bool) {
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
func (t Tag) Number() (float64, bool) {
	if len(t) == 8 {
		return math.Float64frombits(binary.LittleEndian.Uint64(t)), true
	}
	return 0, false
}

// TagNumber is 64-bit float tag constructor.
func TagNumber(val float64) Tag {
	var buf [8]byte
	binary.LittleEndian.PutUint64(buf[:], math.Float64bits(val))
	return buf[:]
}

// Tagset is tags set for each file in package.
type Tagset map[TID]Tag

// FID returns file ID.
func (ts Tagset) FID() FID {
	if data, ok := ts[TIDfid]; ok {
		var fid, _ = data.Uint32()
		return FID(fid)
	}
	return 0
}

// Path returns path of nested into package file.
func (ts Tagset) Path() string {
	if data, ok := ts[TIDpath]; ok {
		return string(data)
	}
	return ""
}

// Name returns name of nested into package file.
func (ts Tagset) Name() string {
	if data, ok := ts[TIDpath]; ok {
		return filepath.Base(string(data))
	}
	return ""
}

// Size returns size of nested into package file.
func (ts Tagset) Size() int64 {
	if data, ok := ts[TIDsize]; ok {
		var size, _ = data.Uint64()
		return int64(size)
	}
	return 0
}

// Offset returns offset of nested into package file.
func (ts Tagset) Offset() int64 {
	if data, ok := ts[TIDoffset]; ok {
		var offset, _ = data.Uint64()
		return int64(offset)
	}
	return 0
}

// ReadFrom reads tags set from stream.
func (ts Tagset) ReadFrom(r io.Reader) (n int64, err error) {
	var num, id, l TID
	if err = binary.Read(r, binary.LittleEndian, &num); err != nil {
		return
	}
	n += 2
	for i := TID(0); i < num; i++ {
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
func (ts Tagset) WriteTo(w io.Writer) (n int64, err error) {
	if err = binary.Write(w, binary.LittleEndian, TID(len(ts))); err != nil {
		return
	}
	n += 2
	for id, data := range ts {
		if err = binary.Write(w, binary.LittleEndian, id); err != nil {
			return
		}
		n += 2
		if err = binary.Write(w, binary.LittleEndian, TID(len(data))); err != nil {
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

// TagSlice is slice of bytes with tags set. Length of slice can be
// not determined to record end, i.e. slice starts at record beginning
// (at number of tags), and can continues after record end.
// fs.FileInfo interface implementation.
type TagSlice []byte

// Num returns number of tags in tags set.
func (ts TagSlice) Num() int {
	if 2 > len(ts) {
		return 0
	}
	return int(binary.LittleEndian.Uint16(ts))
}

// GetTag returns Tag with given identifier.
// If tag is not found, returns ErrNoTag.
// If slice content is broken, returns io.EOF.
func (ts TagSlice) GetTag(tid TID) (Tag, error) {
	var n, tsl = 0, len(ts)
	if n+2 > tsl {
		return nil, io.EOF
	}
	var num = TID(binary.LittleEndian.Uint16(ts[n:]))
	n += 2
	for i := TID(0); i < num; i++ {
		if n+2 > tsl {
			return nil, io.EOF
		}
		var id = TID(binary.LittleEndian.Uint16(ts[n:]))
		n += 2
		if n+2 > tsl {
			return nil, io.EOF
		}
		var l = int(binary.LittleEndian.Uint16(ts[n:]))
		n += 2
		if n+l > tsl {
			return nil, io.EOF
		}
		if id == tid {
			return Tag(ts[n : n+l]), nil
		}
		n += l
	}
	return nil, ErrNoTag
}

// String tag getter.
func (ts TagSlice) String(tid TID) (string, bool) {
	if data, err := ts.GetTag(tid); err == nil {
		return data.String()
	}
	return "", false
}

// Bool is boolean tag getter.
func (ts TagSlice) Bool(tid TID) (bool, bool) {
	if data, err := ts.GetTag(tid); err == nil {
		return data.Bool()
	}
	return false, false
}

// Byte tag getter.
func (ts TagSlice) Byte(tid TID) (byte, bool) {
	if data, err := ts.GetTag(tid); err == nil {
		return data.Byte()
	}
	return 0, false
}

// Uint16 is 16-bit unsigned int tag getter.
// Conversion can be used to get signed 16-bit integers.
func (ts TagSlice) Uint16(tid TID) (TID, bool) {
	if data, err := ts.GetTag(tid); err == nil {
		return data.Uint16()
	}
	return 0, false
}

// Uint32 is 32-bit unsigned int tag getter.
// Conversion can be used to get signed 32-bit integers.
func (ts TagSlice) Uint32(tid TID) (uint32, bool) {
	if data, err := ts.GetTag(tid); err == nil {
		return data.Uint32()
	}
	return 0, false
}

// Uint64 is 64-bit unsigned int tag getter.
// Conversion can be used to get signed 64-bit integers.
func (ts TagSlice) Uint64(tid TID) (uint64, bool) {
	if data, err := ts.GetTag(tid); err == nil {
		return data.Uint64()
	}
	return 0, false
}

// Uint is unspecified size unsigned int tag getter.
func (ts TagSlice) Uint(tid TID) (uint, bool) {
	if data, err := ts.GetTag(tid); err == nil {
		return data.Uint()
	}
	return 0, false
}

// Number is 64-bit float tag getter.
func (ts TagSlice) Number(tid TID) (float64, bool) {
	if data, err := ts.GetTag(tid); err == nil {
		return data.Number()
	}
	return 0, false
}

// FID returns file ID.
func (ts TagSlice) FID() FID {
	var fid, _ = ts.Uint32(TIDfid)
	return FID(fid)
}

// Path returns path of nested into package file.
func (ts TagSlice) Path() string {
	var kpath, _ = ts.String(TIDpath)
	return kpath
}

// Name returns base name of nested into package file.
// fs.FileInfo implementation.
func (ts TagSlice) Name() string {
	var kpath, _ = ts.String(TIDpath)
	return filepath.Base(kpath)
}

// Size returns size of nested into package file.
// fs.FileInfo implementation.
func (ts TagSlice) Size() int64 {
	var size, _ = ts.Uint64(TIDsize)
	return int64(size)
}

// Offset returns offset of nested into package file.
func (ts TagSlice) Offset() int64 {
	var offset, _ = ts.Uint64(TIDoffset)
	return int64(offset)
}

// Mode is for fs.FileInfo interface compatibility.
func (ts TagSlice) Mode() fs.FileMode {
	if _, ok := ts.Uint32(TIDfid); ok { // file ID is absent for dir
		return 0444
	}
	return fs.ModeDir
}

// ModTime returns file timestamp of nested into package file.
// fs.FileInfo implementation.
func (ts TagSlice) ModTime() time.Time {
	var crt, _ = ts.Uint64(TIDcreated)
	return time.Unix(int64(crt), 0)
}

// IsDir detects that object presents a directory. Directory can not have file ID.
// fs.FileInfo implementation.
func (ts TagSlice) IsDir() bool {
	var _, ok = ts.Uint32(TIDfid) // file ID is absent for dir
	return !ok
}

// Sys is for fs.FileInfo interface compatibility.
func (ts TagSlice) Sys() interface{} {
	return nil
}

// DirEntry is directory representation of nested into package files.
// No any reader for directory implementation.
// fs.DirEntry interface implementation.
type DirEntry struct {
	TagSlice // has fs.FileInfo interface
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
	return f.TagSlice, nil
}

// File structure gives access to nested into package file.
// fs.File interface implementation.
type File struct {
	TagSlice // has fs.FileInfo interface
	bytes.Reader
}

// Stat is for fs.File interface compatibility.
func (f *File) Stat() (fs.FileInfo, error) {
	return f.TagSlice, nil
}

// Close is for fs.File interface compatibility.
func (f *File) Close() error {
	return nil
}

// ReadDirFile is a directory file whose entries can be read with the ReadDir method.
// fs.ReadDirFile interface implementation.
type ReadDirFile struct {
	TagSlice // has fs.FileInfo interface
	Pack     Tagger
}

// Stat is for fs.ReadDirFile interface compatibility.
func (f *ReadDirFile) Stat() (fs.FileInfo, error) {
	return f.TagSlice, nil
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
	PackHdr
	FTT FTTMap // keys - package filenames in lower case, values - tags slices.
}

// Enum returns map with file names of all files in package.
func (pack *Package) Enum() FTTMap {
	return pack.FTT
}

// Glob returns the names of all files in package matching pattern or nil
// if there is no matching file.
func (pack *Package) Glob(pattern string) (res []string, err error) {
	pattern = Normalize(pattern)
	var matched bool
	for key := range pack.FTT {
		if matched, err = filepath.Match(pattern, key); err != nil {
			return
		}
		if matched {
			res = append(res, key)
		}
	}
	return
}

// Opens package for reading. At first its checkup file signature, then
// reads records table, and reads file tags set table. Tags set
// for each file must contain at least file ID, file name and creation time.
func (pack *Package) Read(r io.ReadSeeker) (err error) {
	// go to file start
	if _, err = r.Seek(0, io.SeekStart); err != nil {
		return
	}
	// read header
	if err = binary.Read(r, binary.LittleEndian, &pack.PackHdr); err != nil {
		return
	}
	if string(pack.Signature[:]) == Prebuild {
		return ErrSignPre
	}
	if string(pack.Signature[:]) != Signature {
		return ErrSignBad
	}
	pack.FTT = make(FTTMap, pack.TagNumber)

	// read file tags set table
	if _, err = r.Seek(int64(pack.TagOffset), io.SeekStart); err != nil {
		return
	}
	var n int64
	for {
		var tagpos int64
		if tagpos, err = r.Seek(0, io.SeekCurrent); err != nil {
			return
		}
		var tags = Tagset{}
		if n, err = tags.ReadFrom(r); err != nil {
			return
		}
		if n == 2 {
			break // end marker was readed
		}

		// check tags fields
		if _, ok := tags[TIDpath]; !ok {
			return &ErrTag{ErrNoPath, "", TIDpath}
		}
		var key = Normalize(tags.Path())
		if _, ok := pack.FTT[key]; ok {
			return &ErrTag{fs.ErrExist, key, TIDpath}
		}

		if _, ok := tags[TIDfid]; !ok {
			return &ErrTag{ErrNoFID, key, TIDfid}
		}
		var fid = tags.FID()
		if fid > pack.RecNumber {
			return &ErrTag{ErrOutFID, key, TIDfid}
		}

		if _, ok := tags[TIDoffset]; !ok {
			return &ErrTag{ErrNoOffset, key, TIDoffset}
		}
		if _, ok := tags[TIDsize]; !ok {
			return &ErrTag{ErrNoSize, key, TIDsize}
		}
		var offset, size = tags.Offset(), tags.Size()
		if offset < PackHdrSize || offset >= int64(pack.TagOffset) {
			return &ErrTag{ErrOutOff, key, TIDoffset}
		}
		if offset+size > int64(pack.TagOffset) {
			return &ErrTag{ErrOutSize, key, TIDsize}
		}

		// insert file tags
		pack.FTT[key] = OFFSET(tagpos)
	}

	return
}

// ToSlash brings filenames to true slashes.
var ToSlash = filepath.ToSlash

// Normalize brings file path to normalized form. It makes argument lowercase,
// change back slashes to normal slashes. Normalized path is the key to FTTMap.
func Normalize(kpath string) string {
	return strings.ToLower(ToSlash(kpath))
}

// ReadDir returns fs.FileInfo array with nested into given package directory presentation.
// It's core function for ReadDirFile and ReadDirFS structures.
func ReadDir(pack Tagger, dir string, n int) (matches []fs.DirEntry, err error) {
	if !fs.ValidPath(dir) {
		return nil, &fs.PathError{Op: "readdir", Path: dir, Err: fs.ErrInvalid}
	}
	var prefix string
	if dir != "." {
		prefix = Normalize(dir) + "/" // set terminated slash
	}
	var dirs = map[string]struct{}{}
	for key := range pack.Enum() {
		if strings.HasPrefix(key, prefix) {
			var suffix = key[len(prefix):]
			var sp = strings.IndexByte(suffix, '/')
			if sp < 0 { // file detected
				var ts, _ = pack.NamedTags(key)
				matches = append(matches, &DirEntry{ts})
				n--
			} else { // dir detected
				var subdir = path.Join(prefix, suffix[:sp])
				if _, ok := dirs[subdir]; !ok {
					dirs[subdir] = struct{}{}
					var ts, _ = pack.NamedTags(key)
					var fp = ts.Path() // extract not normalized path
					var buf bytes.Buffer
					Tagset{
						TIDpath: TagString(fp[:len(subdir)]),
					}.WriteTo(&buf)
					matches = append(matches, &DirEntry{buf.Bytes()})
					n--
				}
			}
		}
		if n == 0 {
			return
		}
	}
	if n > 0 {
		err = io.EOF
	}
	return
}

// OpenDir returns ReadDirFile structure associated with group of files in package
// pooled with common directory prefix. Usable to implement fs.FileSystem interface.
func OpenDir(pack Tagger, dir string) (fs.ReadDirFile, error) {
	if !fs.ValidPath(dir) {
		return nil, &fs.PathError{Op: "open", Path: dir, Err: fs.ErrInvalid}
	}
	var prefix string
	if dir != "." {
		prefix = Normalize(dir) + "/" // set terminated slash
	}
	for key := range pack.Enum() {
		if strings.HasPrefix(key, prefix) {
			var buf bytes.Buffer
			Tagset{
				TIDpath: TagString(ToSlash(dir)),
			}.WriteTo(&buf)
			return &ReadDirFile{
				TagSlice: buf.Bytes(),
				Pack:     pack,
			}, nil
		}
	}
	return nil, &fs.PathError{Op: "opendir", Path: dir, Err: fs.ErrNotExist}
}

// The End.
