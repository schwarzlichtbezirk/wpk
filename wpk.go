package wpk

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
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
	SIZE   uint64 // data block size
	OFFSET uint64 // data block offset
)

// List of predefined tags IDs.
const (
	TID_FID        TID = 0 // required, uint32
	TID_size       TID = 1 // required, uint64
	TID_offset     TID = 2 // required, uint64
	TID_path       TID = 3 // required, unique, string
	TID_created    TID = 4 // required, uint64
	TID_lastaccess TID = 5 // uint64
	TID_lastwrite  TID = 6 // uint64
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
	TID_keywords TID = 101 // string
	TID_category TID = 102 // string
	TID_link     TID = 103 // string
	TID_version  TID = 104 // string
	TID_author   TID = 105 // string
	TID_comment  TID = 106 // string
)

var (
	ErrNotFound = errors.New("file not found")
	ErrAlready  = errors.New("file with this name already packed")
	ErrSignPre  = errors.New("package is not ready yet")
	ErrSignBad  = errors.New("signature does not pass")
)

// Package header.
type PackHdr struct {
	Signature [0x18]byte `json:"signature"`
	TagOffset OFFSET     `json:"tagoffset"` // tags table offset
	TagNumber FID        `json:"tagnumber"` // number of tagset entries
	RecNumber FID        `json:"recnumber"` // number of records
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
	var size, _ = t.Uint64(TID_size)
	var offset, _ = t.Uint64(TID_offset)
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
		TID_path: TagString(dir),
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
	var kpath, _ = f.String(TID_path)
	var pref = ToKey(kpath)
	if len(pref) > 0 && pref[len(pref)-1] != '/' {
		pref += "/"
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
				var dir = pref + suff[:sp]
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

// Format file path to tags set key. Make argument lowercase,
// change back slashes to normal slashes.
func ToKey(kpath string) string {
	return strings.ToLower(filepath.ToSlash(kpath))
}

// Contains all data needed for package representation.
type Package struct {
	PackHdr
	Tags map[string]Tagset // keys - package filenames in lower case
}

// Returns File structure associated with group of files in package pooled with
// common directory prefix. Usable to implement http.FileSystem interface.
func (pack *Package) NewDir(dir string) *File {
	return &File{
		Tagset: NewDirTagset(dir),
		Pack:   pack,
	}
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
func (pack *Package) NamedRecord(kpath string) (offset int64, size int64, err error) {
	var key = ToKey(kpath)
	if tags, is := pack.Tags[key]; is {
		offset, size = tags.Record()
	} else {
		err = ErrNotFound
	}
	return
}

// Error in tags set. Shows errors associated with any tags.
type TagError struct {
	Index FID    // index in tags table
	TagID TID    // tag ID
	What  string // error message
}

// Format error message of tag error.
func (e *TagError) Error() string {
	return fmt.Sprintf("tag index %d, tag ID %d, %s", e.Index, e.TagID, e.What)
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
	for i := FID(0); i < pack.TagNumber; i++ {
		var tags = Tagset{}
		if _, err = tags.ReadFrom(r); err != nil {
			return
		}

		// check tags fields
		var ok bool

		var fid uint32
		if fid, ok = tags.Uint32(TID_FID); !ok {
			return &TagError{i, TID_FID, "file ID is absent"}
		}
		if fid > uint32(pack.RecNumber) {
			return &TagError{i, TID_FID, fmt.Sprintf("file ID '%d' is out of range", fid)}
		}

		var kpath string
		if kpath, ok = tags.String(TID_path); !ok {
			return &TagError{i, TID_path, fmt.Sprintf("file name is absent for file ID '%d'", fid)}
		}
		var key = ToKey(kpath)
		if _, ok = pack.Tags[key]; ok {
			return &TagError{i, TID_path, fmt.Sprintf("file name '%s' is not unique", kpath)}
		}

		var size, offset uint64
		if size, ok = tags.Uint64(TID_size); !ok {
			return &TagError{i, TID_size, fmt.Sprintf("size is absent for file name '%s'", kpath)}
		}
		if offset, ok = tags.Uint64(TID_offset); !ok {
			return &TagError{i, TID_offset, fmt.Sprintf("offset is absent for file name '%s'", kpath)}
		}
		if offset < PackHdrSize || offset >= uint64(pack.TagOffset) {
			return &TagError{i, TID_offset, fmt.Sprintf("offset is out of bounds for file name '%s'", kpath)}
		}
		if offset+size > uint64(pack.TagOffset) {
			return &TagError{i, TID_size, fmt.Sprintf("file size is out of bounds for file name '%s'", kpath)}
		}

		if _, ok = tags.Uint64(TID_created); !ok {
			return &TagError{i, TID_created, fmt.Sprintf("creation time is absent for file name '%s'", kpath)}
		}

		// insert file tags
		pack.Tags[key] = tags
	}

	return
}

// Writes prebuild header for new empty package.
func (pack *Package) Begin(w io.WriteSeeker) (err error) {
	// reset header
	copy(pack.Signature[:], Prebuild)
	pack.TagOffset = PackHdrSize
	pack.TagNumber = 0
	pack.RecNumber = 0
	// setup empty tags table
	pack.Tags = map[string]Tagset{}
	// go to file start
	if _, err = w.Seek(0, io.SeekStart); err != nil {
		return
	}
	// write prebuild header
	if err = binary.Write(w, binary.LittleEndian, &pack.PackHdr); err != nil {
		return
	}
	return
}

// Writes prebuild header for previously opened package to append new files.
func (pack *Package) Append(w io.WriteSeeker) (err error) {
	// partially reset header
	copy(pack.Signature[:], Prebuild)
	// go to file start
	if _, err = w.Seek(0, io.SeekStart); err != nil {
		return
	}
	// rewrite prebuild header
	if err = binary.Write(w, binary.LittleEndian, &pack.PackHdr); err != nil {
		return
	}
	// go to tags table start to replace it by new data
	if _, err = w.Seek(int64(pack.TagOffset), io.SeekStart); err != nil {
		return
	}
	return
}

// Finalize package writing. Writes true signature and header settings.
func (pack *Package) Complete(w io.WriteSeeker) (err error) {
	// get tags table offset as actual end of file
	var tagoffset int64
	if tagoffset, err = w.Seek(0, io.SeekEnd); err != nil {
		return
	}
	pack.TagOffset = OFFSET(tagoffset)
	pack.TagNumber = FID(len(pack.Tags))
	// write files tags table
	for _, tags := range pack.Tags {
		if _, err = tags.WriteTo(w); err != nil {
			return
		}
	}

	// rewrite true header
	if _, err = w.Seek(0, io.SeekStart); err != nil {
		return
	}
	copy(pack.Signature[:], Signature)
	if err = binary.Write(w, binary.LittleEndian, &pack.PackHdr); err != nil {
		return
	}
	return
}

// Puts data streamed by given reader into package as a file and associate keyname "kpath" with it.
func (pack *Package) PackData(w io.WriteSeeker, r io.Reader, kpath string) (tags Tagset, err error) {
	var key = ToKey(kpath)
	if _, ok := pack.Tags[key]; ok {
		err = ErrAlready
		return
	}

	// get offset and put data ckage
	var offset, size int64
	if offset, err = w.Seek(0, io.SeekCurrent); err != nil {
		return
	}
	if size, err = io.Copy(w, r); err != nil {
		return
	}

	// insert new entry to tags table
	tags = Tagset{
		TID_FID:    TagUint32(uint32(pack.RecNumber + 1)),
		TID_size:   TagUint64(uint64(size)),
		TID_offset: TagUint64(uint64(offset)),
		TID_path:   TagString(kpath),
	}
	pack.Tags[key] = tags

	// update header
	pack.TagOffset = OFFSET(offset + size)
	pack.TagNumber = FID(len(pack.Tags))
	pack.RecNumber++
	return
}

// Puts file with given file full path "fpath" into package and associate keyname "kpath" with it.
func (pack *Package) PackFile(w io.WriteSeeker, kpath, fpath string) (tags Tagset, err error) {
	var file *os.File
	if file, err = os.Open(fpath); err != nil {
		return
	}
	defer file.Close()

	var fi os.FileInfo
	if fi, err = file.Stat(); err != nil {
		return
	}
	if tags, err = pack.PackData(w, file, kpath); err != nil {
		return
	}

	tags[TID_created] = TagUint64(uint64(fi.ModTime().Unix()))
	return
}

// Wrapper to hold file name with error.
type FileError struct {
	What error
	Name string
}

func (e *FileError) Error() string {
	return fmt.Sprintf("error on file '%s': %s", e.Name, e.What.Error())
}

func (e *FileError) Unwrap() error {
	return e.What
}

// Function to report about each file start processing by PackDir function.
type FileReport = func(fi os.FileInfo, kpath, fpath string)

// Puts all files of given folder and it's subfolders into package.
func (pack *Package) PackDir(w io.WriteSeeker, dirname, prefix string, report FileReport) (err error) {
	var fis []os.FileInfo
	if func() {
		var dir *os.File
		if dir, err = os.Open(dirname); err != nil {
			return
		}
		defer dir.Close()

		if fis, err = dir.Readdir(-1); err != nil {
			return
		}
	}(); err != nil {
		return
	}
	for _, fi := range fis {
		var kpath = prefix + fi.Name()
		var fpath = dirname + fi.Name()
		if fi.IsDir() {
			if err = pack.PackDir(w, fpath+"/", kpath+"/", report); err != nil {
				return
			}
		} else {
			if _, err = pack.PackFile(w, kpath, fpath); err != nil {
				err = &FileError{What: err, Name: kpath}
				return
			}
			if report != nil {
				report(fi, kpath, fpath)
			}
		}
	}
	return
}

// The End.
