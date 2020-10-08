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

	ErrNoPath   = errors.New("file name is absent")
	ErrNoFID    = errors.New("file ID is absent")
	ErrOutFID   = errors.New("file ID is out of range")
	ErrNoOffset = errors.New("file offset is absent")
	ErrOutOff   = errors.New("file offset is out of bounds")
	ErrNoSize   = errors.New("file size is absent")
	ErrOutSize  = errors.New("file size is out of bounds")
)

// Refers to package data access management implementation.
type Packager interface {
	OpenWPK(string) error
	io.Closer
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
// os.FileInfo interface implementation.
type Tagset map[TID]Tag

// String tag getter.
func (ts Tagset) String(tid TID) (string, bool) {
	if data, ok := ts[tid]; ok {
		return data.String()
	}
	return "", false
}

// Boolean tag getter.
func (ts Tagset) Bool(tid TID) (bool, bool) {
	if data, ok := ts[tid]; ok {
		return data.Bool()
	}
	return false, false
}

// Byte tag getter.
func (ts Tagset) Byte(tid TID) (byte, bool) {
	if data, ok := ts[tid]; ok {
		return data.Byte()
	}
	return 0, false
}

// 16-bit unsigned int tag getter. Conversion can be used to get signed 16-bit integers.
func (ts Tagset) Uint16(tid TID) (TID, bool) {
	if data, ok := ts[tid]; ok {
		return data.Uint16()
	}
	return 0, false
}

// 32-bit unsigned int tag getter. Conversion can be used to get signed 32-bit integers.
func (ts Tagset) Uint32(tid TID) (uint32, bool) {
	if data, ok := ts[tid]; ok {
		return data.Uint32()
	}
	return 0, false
}

// 64-bit unsigned int tag getter. Conversion can be used to get signed 64-bit integers.
func (ts Tagset) Uint64(tid TID) (uint64, bool) {
	if data, ok := ts[tid]; ok {
		return data.Uint64()
	}
	return 0, false
}

// Unspecified size unsigned int tag getter.
func (ts Tagset) Uint(tid TID) (uint, bool) {
	if data, ok := ts[tid]; ok {
		return data.Uint()
	}
	return 0, false
}

// 64-bit float tag getter.
func (ts Tagset) Number(tid TID) (float64, bool) {
	if data, ok := ts[tid]; ok {
		return data.Number()
	}
	return 0, false
}

// Returns file ID.
func (t Tagset) FID() FID {
	var fid, _ = t.Uint32(TID_FID)
	return FID(fid)
}

// Returns file offset & size.
func (t Tagset) Record() (int64, int64) {
	var offset, _ = t.Uint64(TID_offset)
	var size, _ = t.Uint64(TID_size)
	return int64(offset), int64(size)
}

// Reads tags set from stream.
func (t Tagset) ReadFrom(r io.Reader) (n int64, err error) {
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
		t[id] = data
	}
	return
}

// Writes tags set to stream.
func (t Tagset) WriteTo(w io.Writer) (n int64, err error) {
	if err = binary.Write(w, binary.LittleEndian, TID(len(t))); err != nil {
		return
	}
	n += 2
	for id, data := range t {
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

// Returns path of nested into package file.
func (t Tagset) Path() string {
	var kpath, _ = t.String(TID_path)
	return kpath
}

// Returns name of nested into package file.
func (t Tagset) Name() string {
	var kpath, _ = t.String(TID_path)
	return filepath.Base(kpath)
}

// Returns size of nested into package file.
func (t Tagset) Size() int64 {
	var size, _ = t.Uint64(TID_size)
	return int64(size)
}

// For os.FileInfo interface compatibility.
func (t Tagset) Mode() os.FileMode {
	return 0444
}

// Returns file timestamp of nested into package file.
func (t Tagset) ModTime() time.Time {
	var crt, _ = t.Uint64(TID_created)
	return time.Unix(int64(crt), 0)
}

// Detects that object presents a directory. Directory can not have file ID.
func (t Tagset) IsDir() bool {
	var _, ok = t.Uint32(TID_FID) // file ID is absent for dir
	return !ok
}

// For os.FileInfo interface compatibility.
func (t Tagset) Sys() interface{} {
	return nil
}

// MakesMakes object compatible with http.File interface
// to present nested into package directory.
func NewDirTagset(dir string) Tagset {
	return Tagset{
		TID_path: TagString(ToSlash(dir)),
		TID_size: TagUint64(0),
	}
}

// Gives access to nested into package file.
// http.File interface implementation.
type File struct {
	Tagset
	bytes.Reader
	Pack *Package
}

// For http.File interface compatibility.
func (f *File) Close() error {
	return nil
}

// For http.File interface compatibility.
func (f *File) Stat() (os.FileInfo, error) {
	return f.Tagset, nil
}

// Returns os.FileInfo array with nested into given package directory presentation.
func (f *File) Readdir(count int) (matches []os.FileInfo, err error) {
	var pref = ToKey(f.Path())
	if len(pref) > 0 && pref[len(pref)-1] != '/' {
		pref += "/" // set terminated slash
	}
	var dirs = map[string]os.FileInfo{}
	for key, tags := range f.Pack.Tags {
		if strings.HasPrefix(key, pref) {
			var suff = key[len(pref):]
			var sp = strings.IndexByte(suff, '/')
			if sp < 0 {
				matches = append(matches, tags)
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
	Tags map[string]Tagset // keys - package filenames in lower case
}

// Returns File structure associated with group of files in package pooled with
// common directory prefix. Usable to implement http.FileSystem interface.
func (pack *Package) OpenDir(dir string) (http.File, error) {
	var pref = ToKey(dir)
	if len(pref) > 0 && pref[len(pref)-1] != '/' {
		pref += "/" // set terminated slash
	}
	for key := range pack.Tags {
		if strings.HasPrefix(key, pref) {
			return &File{
				Tagset: NewDirTagset(dir),
				Pack:   pack,
			}, nil
		}
	}
	return nil, &ErrKey{ErrNotFound, pref}
}

// Returns the names of all files in package matching pattern or nil
// if there is no matching file.
func (pack *Package) Glob(pattern string, found func(key string) error) (err error) {
	pattern = ToKey(pattern)
	var matched bool
	for key := range pack.Tags {
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

// Returns record associated with given filename.
func (pack *Package) NamedRecord(key string) (offset int64, size int64, err error) {
	if tags, is := pack.Tags[key]; is {
		offset, size = tags.Record()
	} else {
		err = &ErrKey{ErrNotFound, key}
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
	pack.Tags = make(map[string]Tagset, pack.TagNumber)

	// read file tags set table
	if _, err = r.Seek(int64(pack.TagOffset), io.SeekStart); err != nil {
		return
	}
	var n int64
	for {
		var tags = Tagset{}
		if n, err = tags.ReadFrom(r); err != nil {
			return
		}
		if n == 2 {
			break // end marker was readed
		}

		// check tags fields
		var ok bool
		var kpath string
		if kpath, ok = tags.String(TID_path); !ok {
			return &ErrTag{ErrKey{ErrNoPath, ""}, TID_path}
		}
		var key = strings.ToLower(kpath)
		if _, ok = pack.Tags[key]; ok {
			return &ErrTag{ErrKey{ErrAlready, key}, TID_path}
		}

		var fid uint32
		if fid, ok = tags.Uint32(TID_FID); !ok {
			return &ErrTag{ErrKey{ErrNoFID, key}, TID_FID}
		}
		if fid > uint32(pack.RecNumber) {
			return &ErrTag{ErrKey{ErrOutFID, key}, TID_FID}
		}

		var offset, size uint64
		if offset, ok = tags.Uint64(TID_offset); !ok {
			return &ErrTag{ErrKey{ErrNoOffset, key}, TID_offset}
		}
		if size, ok = tags.Uint64(TID_size); !ok {
			return &ErrTag{ErrKey{ErrNoSize, key}, TID_size}
		}
		if offset < PackHdrSize || offset >= uint64(pack.TagOffset) {
			return &ErrTag{ErrKey{ErrOutOff, key}, TID_offset}
		}
		if offset+size > uint64(pack.TagOffset) {
			return &ErrTag{ErrKey{ErrOutSize, key}, TID_size}
		}

		// insert file tags
		pack.Tags[key] = tags
	}

	return
}

// The End.
