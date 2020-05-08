package wpk

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"strings"
)

const (
	Signature = "Whirlwind 3.1 Package   " // package is ready for use
	Prebuild  = "Whirlwind 3.1 Prebuild  " // package is in building progress
)

type (
	TID  uint16
	FID  uint32
	SIZE uint64
)

// List of predefined tags IDs.
const (
	TID_FID        = 0 // required, uint64
	TID_name       = 1 // required, unique, string
	TID_created    = 2 // required, uint64
	TID_lastaccess = 3 // uint64
	TID_lastwrite  = 4 // uint64
	TID_change     = 5 // uint64
	TID_fileattr   = 6 // uint64

	TID_CRC32IEEE = 10 // uint32, CRC-32-IEEE 802.3, poly = 0x04C11DB7, init = -1
	TID_CRC32C    = 11 // uint32, (Castagnoli), poly = 0x1EDC6F41, init = -1
	TID_CRC32K    = 12 // uint32, (Koopman), poly = 0x741B8CD7, init = -1
	TID_CRC64ISO  = 14 // uint64, poly = 0xD800000000000000, init = -1

	TID_MD5    = 20 // [16]byte
	TID_SHA1   = 21 // [20]byte
	TID_SHA224 = 22 // [28]byte
	TID_SHA256 = 23 // [32]byte
	TID_SHA384 = 24 // [48]byte
	TID_SHA512 = 25 // [64]byte

	TID_mime     = 100 // string
	TID_keywords = 101 // string
	TID_category = 102 // string
	TID_version  = 103 // string
	TID_author   = 104 // string
	TID_comment  = 105 // string
)

var (
	ErrNotFound = errors.New("file not found")
	ErrNoName   = errors.New("file name expected")
	ErrAlready  = errors.New("file with this name already packed")
	ErrSignPre  = errors.New("package is not ready yet")
	ErrSignBad  = errors.New("signature does not pass")
)

// Package header.
type PackHdr struct {
	Signature [0x18]byte
	RecOffset SIZE  // file allocation table offset
	RecNumber int64 // number of records
	TagOffset SIZE  // tags table offset
	TagNumber int64 // number of tagset entries
}

// Package record item.
type PackRec struct {
	Offset int64 // datablock offset
	Size   int64 // datablock size
}

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

// 64-bit float tag getter.
func (ts Tagset) Number(tid TID) (float64, bool) {
	if data, ok := ts[tid]; ok {
		return data.Number()
	}
	return 0, false
}

// Reads tags set from stream.
func (t Tagset) Read(r io.Reader) (err error) {
	var num TID
	if err = binary.Read(r, binary.LittleEndian, &num); err != nil {
		return
	}
	for i := TID(0); i < num; i++ {
		var id, l TID
		if err = binary.Read(r, binary.LittleEndian, &id); err != nil {
			return
		}
		if err = binary.Read(r, binary.LittleEndian, &l); err != nil {
			return
		}
		var data = make([]byte, l)
		if err = binary.Read(r, binary.LittleEndian, &data); err != nil {
			return
		}
		t[id] = data
	}
	return
}

// Writes tags set to stream.
func (t Tagset) Write(w io.Writer) (err error) {
	if err = binary.Write(w, binary.LittleEndian, TID(len(t))); err != nil {
		return
	}
	for id, data := range t {
		if err = binary.Write(w, binary.LittleEndian, id); err != nil {
			return
		}
		if err = binary.Write(w, binary.LittleEndian, TID(len(data))); err != nil {
			return
		}
		if err = binary.Write(w, binary.LittleEndian, data); err != nil {
			return
		}
	}
	return
}

// Contains all data needed for package representation.
type Package struct {
	PackHdr
	FAT  []PackRec         // file allocation table
	Tags map[string]Tagset // keys - package filenames in lower case
}

// Returns record associated with given filename.
func (pack *Package) NamedRecord(fname string) (*PackRec, error) {
	var key = strings.ToLower(filepath.ToSlash(fname))
	var tags, is = pack.Tags[key]
	if !is {
		return nil, ErrNotFound
	}
	var fid, _ = tags.Uint64(TID_FID)
	return &pack.FAT[fid], nil
}

// Returns copy of data block tagged by given file name.
func (pack *Package) Extract(r io.ReaderAt, fname string) (buf []byte, err error) {
	var rec *PackRec
	if rec, err = pack.NamedRecord(fname); err != nil {
		return
	}
	buf = make([]byte, rec.Size)
	_, err = r.ReadAt(buf, rec.Offset)
	return
}

// Error in tags set. Shows errors associated with any tags.
type TagError struct {
	Index int64  // index in tags table
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
func (pack *Package) Open(r io.ReadSeeker) (err error) {
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
	pack.FAT = make([]PackRec, pack.RecNumber)
	pack.Tags = make(map[string]Tagset, pack.TagNumber)

	// read records table
	if _, err = r.Seek(int64(pack.RecOffset), io.SeekStart); err != nil {
		return
	}
	if err = binary.Read(r, binary.LittleEndian, &pack.FAT); err != nil {
		return
	}

	// read file tags set table
	if _, err = r.Seek(int64(pack.TagOffset), io.SeekStart); err != nil {
		return
	}
	for i := int64(0); i < pack.TagNumber; i++ {
		var tags = Tagset{}
		if err = tags.Read(r); err != nil {
			return
		}
		// check tags fields
		var ok bool
		var fid uint64
		var fname string
		if fid, ok = tags.Uint64(TID_FID); !ok {
			return &TagError{i, TID_FID, "file ID is absent"}
		}
		if fid >= uint64(len(pack.FAT)) {
			return &TagError{i, TID_FID, fmt.Sprintf("file ID '%d' is out of range", fid)}
		}
		if fname, ok = tags.String(TID_name); !ok {
			return &TagError{i, TID_name, fmt.Sprintf("file name is absent for file ID '%d'", fid)}
		}
		var key = strings.ToLower(filepath.ToSlash(fname))
		if _, ok = pack.Tags[key]; ok {
			return &TagError{i, TID_name, fmt.Sprintf("file name '%s' is not unique", fname)}
		}
		if _, ok = tags.Uint64(TID_created); !ok {
			return &TagError{i, TID_created, fmt.Sprintf("creation time is absent for file name '%s'", fname)}
		}
		// insert file tags
		pack.Tags[key] = tags
	}

	return
}

func (pack *Package) PackData(w io.WriteSeeker, r io.Reader, tags Tagset) (err error) {
	var key string
	if tags != nil {
		var fname, ok = tags.String(TID_name)
		if !ok {
			return ErrNoName
		}
		key = strings.ToLower(filepath.ToSlash(fname))
		if _, ok = pack.Tags[key]; ok {
			return ErrAlready
		}
	}

	var rec PackRec
	if rec.Offset, err = w.Seek(0, io.SeekCurrent); err != nil {
		return err
	}
	if rec.Size, err = io.Copy(w, r); err != nil {
		return err
	}
	pack.FAT = append(pack.FAT, rec)

	if tags != nil {
		var fid = uint64(len(pack.FAT)) - 1
		tags[TID_FID] = TagUint64(fid)
		pack.Tags[key] = tags
	}
	return nil
}

func (pack *Package) PackFile(w io.WriteSeeker, fname, fpath string) (tags Tagset, err error) {
	var file *os.File
	if file, err = os.Open(fpath); err != nil {
		return
	}
	defer func() {
		err = file.Close()
	}()

	var fi os.FileInfo
	if fi, err = file.Stat(); err != nil {
		return
	}
	tags = Tagset{
		TID_name:    TagString(fname),
		TID_created: TagUint64(uint64(fi.ModTime().Unix())),
	}
	if err = pack.PackData(w, file, tags); err != nil {
		return
	}
	return
}

// Function to report about each file start processing by PackDir function.
type FileReport = func(fi os.FileInfo, fname, fpath string)

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
		var fname = prefix + fi.Name()
		var fpath = dirname + fi.Name()
		if fi.IsDir() {
			if err = pack.PackDir(w, fpath+"/", fname+"/", report); err != nil {
				return
			}
		} else {
			if report != nil {
				report(fi, fname, fpath)
			}
			if _, err = pack.PackFile(w, fname, fpath); err != nil {
				return
			}
		}
	}
	return
}

// The End.
