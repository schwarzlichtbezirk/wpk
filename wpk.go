package wpk

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	Signature = "Whirlwind 3.2 Package   " // package is ready for use
	Prebuild  = "Whirlwind 3.2 Prebuild  " // package is in building progress
)

type (
	TID    uint16 // tag identifier
	FID    uint32 // file index/identifier
	OFFSET uint64 // data block offset
	SIZE   uint64 // data block size
)

// List of predefined tags IDs.
const (
	TID_FID        TID = 0 // required, uint32
	TID_offset     TID = 1 // required, uint64
	TID_size       TID = 2 // required, uint64
	TID_path       TID = 3 // required, unique, string
	TID_created    TID = 4 // required for files, uint64
	TID_lastwrite  TID = 5 // uint64
	TID_lastaccess TID = 6 // uint64
	TID_change     TID = 7 // uint64
	TID_fileattr   TID = 8 // uint32

	TID_SYS TID = 10 // system protection marker

	TID_CRC32IEEE TID = 10 // uint32, CRC-32-IEEE 802.3, poly = 0x04C11DB7, init = -1
	TID_CRC32C    TID = 11 // uint32, (Castagnoli), poly = 0x1EDC6F41, init = -1
	TID_CRC32K    TID = 12 // uint32, (Koopman), poly = 0x741B8CD7, init = -1
	TID_CRC64ISO  TID = 14 // uint64, poly = 0xD800000000000000, init = -1

	TID_MD5    TID = 20 // [16]byte
	TID_SHA1   TID = 21 // [20]byte
	TID_SHA224 TID = 22 // [28]byte
	TID_SHA256 TID = 23 // [32]byte
	TID_SHA384 TID = 24 // [48]byte
	TID_SHA512 TID = 25 // [64]byte

	TID_mime     TID = 100 // string
	TID_link     TID = 101 // string
	TID_keywords TID = 102 // string
	TID_category TID = 103 // string
	TID_version  TID = 104 // string
	TID_author   TID = 105 // string
	TID_comment  TID = 106 // string
)

// Some error with key of tags set.
type ErrKey struct {
	What error  // error message
	Key  string // file key
}

func (e *ErrKey) Error() string {
	return fmt.Sprintf("key '%s': %s", e.Key, e.What)
}

func (e *ErrKey) Unwrap() error {
	return e.What
}

// Error on some field of tags set.
type ErrTag struct {
	ErrKey
	TID TID // tag ID
}

func (e *ErrTag) Error() string {
	return fmt.Sprintf("key '%s', tag ID %d: %s", e.Key, e.TID, e.What)
}

func (e *ErrTag) Unwrap() error {
	return &e.ErrKey
}

var (
	ErrSignPre  = errors.New("package is not ready yet")
	ErrSignBad  = errors.New("signature does not pass")
	ErrNotFound = errors.New("file is not found in package")
	ErrAlready  = errors.New("file already present in package")

	ErrNoTag    = errors.New("tag with given ID not found")
	ErrNoPath   = errors.New("file name is absent")
	ErrNoFID    = errors.New("file ID is absent")
	ErrOutFID   = errors.New("file ID is out of range")
	ErrNoOffset = errors.New("file offset is absent")
	ErrOutOff   = errors.New("file offset is out of bounds")
	ErrNoSize   = errors.New("file size is absent")
	ErrOutSize  = errors.New("file size is out of bounds")
)

// Tags attributes table map.
type TATMap map[string]OFFSET

// Provide tags set access.
type Tagger interface {
	Enum() TATMap
	NamedTags(string) (TagSlice, bool)
}

// Refer to package data access management implementation.
type Packager interface {
	OpenWPK(string) error
	io.Closer
	Tagger
	SubDir(string) Packager
	Extract(string) ([]byte, error)
	http.FileSystem
}

// Package header.
type PackHdr struct {
	Signature [0x18]byte `json:"signature"`
	TagOffset OFFSET     `json:"tagoffset"` // tags table offset
	RecNumber FID        `json:"recnumber"` // number of records
	TagNumber FID        `json:"tagnumber"` // number of tagset entries
}

// Package header length.
const PackHdrSize = 40

// Tag - file description item.
type Tag []byte

// String tag converter.
func (t Tag) String() (string, bool) {
	return string(t), true
}

// String tag constructor.
func TagString(val string) Tag {
	return Tag(val)
}

// Boolean tag converter.
func (t Tag) Bool() (bool, bool) {
	if len(t) == 1 {
		return t[0] > 0, true
	}
	return false, false
}

// Boolean tag constructor.
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

// Byte tag constructor.
func TagByte(val byte) Tag {
	var buf = [1]byte{val}
	return buf[:]
}

// 16-bit unsigned int tag converter.
func (t Tag) Uint16() (TID, bool) {
	if len(t) == 2 {
		return TID(binary.LittleEndian.Uint16(t)), true
	}
	return 0, false
}

// 16-bit unsigned int tag constructor.
func TagUint16(val TID) Tag {
	var buf [2]byte
	binary.LittleEndian.PutUint16(buf[:], uint16(val))
	return buf[:]
}

// 32-bit unsigned int tag converter.
func (t Tag) Uint32() (uint32, bool) {
	if len(t) == 4 {
		return binary.LittleEndian.Uint32(t), true
	}
	return 0, false
}

// 32-bit unsigned int tag constructor.
func TagUint32(val uint32) Tag {
	var buf [4]byte
	binary.LittleEndian.PutUint32(buf[:], val)
	return buf[:]
}

// 64-bit unsigned int tag converter.
func (t Tag) Uint64() (uint64, bool) {
	if len(t) == 8 {
		return binary.LittleEndian.Uint64(t), true
	}
	return 0, false
}

// 64-bit unsigned int tag constructor.
func TagUint64(val uint64) Tag {
	var buf [8]byte
	binary.LittleEndian.PutUint64(buf[:], val)
	return buf[:]
}

// unspecified size unsigned int tag converter.
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

// 64-bit float tag converter.
func (t Tag) Number() (float64, bool) {
	if len(t) == 8 {
		return math.Float64frombits(binary.LittleEndian.Uint64(t)), true
	}
	return 0, false
}

// 64-bit float tag constructor.
func TagNumber(val float64) Tag {
	var buf [8]byte
	binary.LittleEndian.PutUint64(buf[:], math.Float64bits(val))
	return buf[:]
}

// Tags set for each file in package.
type Tagset map[TID]Tag

// Returns file ID.
func (ts Tagset) FID() FID {
	if data, ok := ts[TID_FID]; ok {
		var fid, _ = data.Uint32()
		return FID(fid)
	}
	return 0
}

// Returns path of nested into package file.
func (ts Tagset) Path() string {
	if data, ok := ts[TID_path]; ok {
		return string(data)
	}
	return ""
}

// Returns name of nested into package file.
func (ts Tagset) Name() string {
	if data, ok := ts[TID_path]; ok {
		return filepath.Base(string(data))
	}
	return ""
}

// Returns size of nested into package file.
func (ts Tagset) Size() int64 {
	if data, ok := ts[TID_size]; ok {
		var size, _ = data.Uint64()
		return int64(size)
	}
	return 0
}

// Returns offset of nested into package file.
func (ts Tagset) Offset() int64 {
	if data, ok := ts[TID_offset]; ok {
		var offset, _ = data.Uint64()
		return int64(offset)
	}
	return 0
}

// Reads tags set from stream.
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

// Writes tags set to stream.
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

// Slice of bytes with tags set. Length of slice can be not determined
// to record end, i.e. slice starts at record beginning
// (at number of tags), and can continues after record end.
// os.FileInfo interface implementation.
type TagSlice []byte

// Returns number of tags in tags set.
func (ts TagSlice) Num() int {
	if 2 > len(ts) {
		return 0
	}
	return int(binary.LittleEndian.Uint16(ts))
}

// Returns Tag with given identifier.
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

// Boolean tag getter.
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

// 16-bit unsigned int tag getter. Conversion can be used to get signed 16-bit integers.
func (ts TagSlice) Uint16(tid TID) (TID, bool) {
	if data, err := ts.GetTag(tid); err == nil {
		return data.Uint16()
	}
	return 0, false
}

// 32-bit unsigned int tag getter. Conversion can be used to get signed 32-bit integers.
func (ts TagSlice) Uint32(tid TID) (uint32, bool) {
	if data, err := ts.GetTag(tid); err == nil {
		return data.Uint32()
	}
	return 0, false
}

// 64-bit unsigned int tag getter. Conversion can be used to get signed 64-bit integers.
func (ts TagSlice) Uint64(tid TID) (uint64, bool) {
	if data, err := ts.GetTag(tid); err == nil {
		return data.Uint64()
	}
	return 0, false
}

// Unspecified size unsigned int tag getter.
func (ts TagSlice) Uint(tid TID) (uint, bool) {
	if data, err := ts.GetTag(tid); err == nil {
		return data.Uint()
	}
	return 0, false
}

// 64-bit float tag getter.
func (ts TagSlice) Number(tid TID) (float64, bool) {
	if data, err := ts.GetTag(tid); err == nil {
		return data.Number()
	}
	return 0, false
}

// Returns file ID.
func (ts TagSlice) FID() FID {
	var fid, _ = ts.Uint32(TID_FID)
	return FID(fid)
}

// Returns path of nested into package file.
func (ts TagSlice) Path() string {
	var kpath, _ = ts.String(TID_path)
	return kpath
}

// Returns name of nested into package file.
func (ts TagSlice) Name() string {
	var kpath, _ = ts.String(TID_path)
	return filepath.Base(kpath)
}

// Returns size of nested into package file.
func (ts TagSlice) Size() int64 {
	var size, _ = ts.Uint64(TID_size)
	return int64(size)
}

// Returns offset of nested into package file.
func (ts TagSlice) Offset() int64 {
	var offset, _ = ts.Uint64(TID_offset)
	return int64(offset)
}

// For os.FileInfo interface compatibility.
func (ts TagSlice) Mode() os.FileMode {
	return 0444
}

// Returns file timestamp of nested into package file.
func (ts TagSlice) ModTime() time.Time {
	var crt, _ = ts.Uint64(TID_created)
	return time.Unix(int64(crt), 0)
}

// Detects that object presents a directory. Directory can not have file ID.
func (ts TagSlice) IsDir() bool {
	var _, ok = ts.Uint32(TID_FID) // file ID is absent for dir
	return !ok
}

// For os.FileInfo interface compatibility.
func (ts TagSlice) Sys() interface{} {
	return nil
}

// MakesMakes object compatible with http.File interface
// to present nested into package directory.
func NewDirTagset(dir string) TagSlice {
	var buf bytes.Buffer
	var tags = Tagset{
		TID_path: TagString(ToSlash(dir)),
		TID_size: TagUint64(0),
	}
	tags.WriteTo(&buf)
	return buf.Bytes()
}

// Gives access to nested into package file.
// http.File interface implementation.
type File struct {
	TagSlice
	bytes.Reader
	Pack Tagger
}

// For http.File interface compatibility.
func (f *File) Close() error {
	return nil
}

// For http.File interface compatibility.
func (f *File) Stat() (os.FileInfo, error) {
	return f.TagSlice, nil
}

// Returns os.FileInfo array with nested into given package directory presentation.
func (f *File) Readdir(count int) (matches []os.FileInfo, err error) {
	var pref = ToKey(f.Path())
	if len(pref) > 0 && pref[len(pref)-1] != '/' {
		pref += "/" // set terminated slash
	}
	var dirs = map[string]os.FileInfo{}
	var tat = f.Pack.Enum()
	for key := range tat {
		if strings.HasPrefix(key, pref) {
			var suff = key[len(pref):]
			var sp = strings.IndexByte(suff, '/')
			if sp < 0 {
				var ts, _ = f.Pack.NamedTags(key)
				matches = append(matches, ts)
				count--
			} else { // dir detected
				var dir = pref + suff[:sp+1] // with terminates slash
				if _, ok := dirs[dir]; !ok {
					var fi = NewDirTagset(dir)
					dirs[dir] = fi
					matches = append(matches, fi)
					count--
				}
			}
			if count == 0 {
				break
			}
		}
	}
	return
}

// Returns File structure associated with group of files in package pooled with
// common directory prefix. Usable to implement http.FileSystem interface.
func OpenDir(pack Tagger, dir string) (http.File, error) {
	var pref = ToKey(dir)
	if len(pref) > 0 && pref[len(pref)-1] != '/' {
		pref += "/" // set terminated slash
	}
	var tat = pack.Enum()
	for key := range tat {
		if strings.HasPrefix(key, pref) {
			return &File{
				TagSlice: NewDirTagset(dir),
				Pack:     pack,
			}, nil
		}
	}
	return nil, &ErrKey{ErrNotFound, pref}
}

// Brings filenames to true slashes.
var ToSlash = filepath.ToSlash

// Format file path to tags set key. Make argument lowercase,
// change back slashes to normal slashes.
func ToKey(kpath string) string {
	return strings.ToLower(ToSlash(kpath))
}

// Contains all data needed for package representation.
type Package struct {
	PackHdr
	TAT TATMap // keys - package filenames in lower case
}

// Returns map with file names of all files in package.
func (pack *Package) Enum() TATMap {
	return pack.TAT
}

// Returns the names of all files in package matching pattern or nil
// if there is no matching file.
func (pack *Package) Glob(pattern string, found func(key string) error) (err error) {
	pattern = ToKey(pattern)
	var matched bool
	for key := range pack.TAT {
		if matched, err = filepath.Match(pattern, key); err != nil {
			return
		}
		if matched {
			if err = found(key); err != nil {
				return
			}
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
	pack.TAT = make(TATMap, pack.TagNumber)

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
		if _, ok := tags[TID_path]; !ok {
			return &ErrTag{ErrKey{ErrNoPath, ""}, TID_path}
		}
		var key = ToKey(tags.Path())
		if _, ok := pack.TAT[key]; ok {
			return &ErrTag{ErrKey{ErrAlready, key}, TID_path}
		}

		if _, ok := tags[TID_FID]; !ok {
			return &ErrTag{ErrKey{ErrNoFID, key}, TID_FID}
		}
		var fid = tags.FID()
		if fid > pack.RecNumber {
			return &ErrTag{ErrKey{ErrOutFID, key}, TID_FID}
		}

		if _, ok := tags[TID_offset]; !ok {
			return &ErrTag{ErrKey{ErrNoOffset, key}, TID_offset}
		}
		if _, ok := tags[TID_size]; !ok {
			return &ErrTag{ErrKey{ErrNoSize, key}, TID_size}
		}
		var offset, size = tags.Offset(), tags.Size()
		if offset < PackHdrSize || offset >= int64(pack.TagOffset) {
			return &ErrTag{ErrKey{ErrOutOff, key}, TID_offset}
		}
		if offset+size > int64(pack.TagOffset) {
			return &ErrTag{ErrKey{ErrOutSize, key}, TID_size}
		}

		// insert file tags
		pack.TAT[key] = OFFSET(tagpos)
	}

	return
}

// The End.
