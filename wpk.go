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

// List of predefined attributes IDs.
const (
	AID_FID        = 0 // required, uint64
	AID_name       = 1 // required, unique, string
	AID_created    = 2 // required, uint64
	AID_lastaccess = 3 // uint64
	AID_lastwrite  = 4 // uint64
	AID_change     = 5 // uint64
	AID_fileattr   = 6 // uint64

	AID_CRC32IEEE = 10 // uint32, CRC-32-IEEE 802.3, poly = 0x04C11DB7, init = -1
	AID_CRC32C    = 11 // uint32, (Castagnoli), poly = 0x1EDC6F41, init = -1
	AID_CRC32K    = 12 // uint32, (Koopman), poly = 0x741B8CD7, init = -1
	AID_CRC64ISO  = 14 // uint64, poly = 0xD800000000000000, init = -1

	AID_MD5    = 20 // [16]byte
	AID_SHA1   = 21 // [20]byte
	AID_SHA224 = 22 // [28]byte
	AID_SHA256 = 23 // [32]byte
	AID_SHA384 = 24 // [48]byte
	AID_SHA512 = 25 // [64]byte

	AID_mime     = 100 // string
	AID_keywords = 101 // string
	AID_category = 102 // string
	AID_version  = 103 // string
	AID_author   = 104 // string
	AID_comment  = 105 // string
)

var (
	ErrNotFound = errors.New("file not found")
	ErrAlready  = errors.New("file with this name already packed")
	ErrSignPre  = errors.New("package is not ready yet")
	ErrSignBad  = errors.New("signature does not pass")
)

// Package header.
type PackHdr struct {
	Signature [0x18]byte
	RecOffset int64 // file allocation table offset
	RecNumber int64 // number of records
	TagOffset int64 // tags table offset
	TagNumber int64 // number of tagset entries
}

// Package record item.
type PackRec struct {
	Offset int64 // datablock offset
	Size   int64 // datablock size
}

// Tags set for each file in package.
type TagSet map[uint16][]byte

// String tag getter.
func (t TagSet) String(id uint16) (string, bool) {
	if data, ok := t[id]; ok {
		return string(data), ok
	}
	return "", false
}

// String tag setter.
func (t TagSet) SetString(id uint16, val string) {
	t[id] = []byte(val)
}

// Boolean tag getter.
func (t TagSet) Bool(id uint16) (bool, bool) {
	if data, ok := t[id]; ok && len(data) == 1 {
		return data[0] > 0, ok
	}
	return false, false
}

// Boolean tag setter.
func (t TagSet) SetBool(id uint16, val bool) {
	var buf [1]byte
	if val {
		buf[0] = 1
	}
	t[id] = buf[:]
}

// 16-bit unsigned int tag getter. Conversion can be used to get signed 16-bit integers.
func (t TagSet) Uint16(id uint16) (uint16, bool) {
	if data, ok := t[id]; ok && len(data) == 2 {
		return binary.LittleEndian.Uint16(data), ok
	}
	return 0, false
}

// 16-bit unsigned int tag setter.
func (t TagSet) SetUint16(id uint16, val uint16) {
	var buf [2]byte
	binary.LittleEndian.PutUint16(buf[:], val)
	t[id] = buf[:]
}

// 32-bit unsigned int tag getter. Conversion can be used to get signed 32-bit integers.
func (t TagSet) Uint32(id uint16) (uint32, bool) {
	if data, ok := t[id]; ok && len(data) == 4 {
		return binary.LittleEndian.Uint32(data), ok
	}
	return 0, false
}

// 32-bit unsigned int tag setter.
func (t TagSet) SetUint32(id uint16, val uint32) {
	var buf [4]byte
	binary.LittleEndian.PutUint32(buf[:], val)
	t[id] = buf[:]
}

// 64-bit unsigned int tag getter. Conversion can be used to get signed 64-bit integers.
func (t TagSet) Uint64(id uint16) (uint64, bool) {
	if data, ok := t[id]; ok && len(data) == 8 {
		return binary.LittleEndian.Uint64(data), ok
	}
	return 0, false
}

// 64-bit unsigned int tag setter.
func (t TagSet) SetUint64(id uint16, val uint64) {
	var buf [8]byte
	binary.LittleEndian.PutUint64(buf[:], val)
	t[id] = buf[:]
}

// 64-bit float tag getter.
func (t TagSet) Number(id uint16) (float64, bool) {
	if data, ok := t[id]; ok && len(data) == 8 {
		return math.Float64frombits(binary.LittleEndian.Uint64(data)), ok
	}
	return 0, false
}

// 64-bit float tag setter.
func (t TagSet) SetNumber(id uint16, val float64) {
	var buf [8]byte
	binary.LittleEndian.PutUint64(buf[:], math.Float64bits(val))
	t[id] = buf[:]
}

// Reads tags set from stream.
func (t TagSet) Read(r io.Reader) (err error) {
	var num uint16
	if err = binary.Read(r, binary.LittleEndian, &num); err != nil {
		return
	}
	for i := uint16(0); i < num; i++ {
		var id, l uint16
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
func (t TagSet) Write(w io.Writer) (err error) {
	if err = binary.Write(w, binary.LittleEndian, uint16(len(t))); err != nil {
		return
	}
	for id, data := range t {
		if err = binary.Write(w, binary.LittleEndian, id); err != nil {
			return
		}
		if err = binary.Write(w, binary.LittleEndian, uint16(len(data))); err != nil {
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
	Tags map[string]TagSet // keys - package filenames in lower case
}

// Makes empty package structure ready to put new entries.
func (pack *Package) Init() {
	copy(pack.Signature[:], Prebuild)
	pack.FAT = []PackRec{}
	pack.Tags = map[string]TagSet{}
}

// Returns record associated with given filename.
func (pack *Package) NamedRecord(fname string) (*PackRec, error) {
	var key = strings.ToLower(filepath.ToSlash(fname))
	var tags, is = pack.Tags[key]
	if !is {
		return nil, ErrNotFound
	}
	var fid, _ = tags.Uint64(AID_FID)
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

// Error in tag set. Shows errors associated with any tags.
type TagError struct {
	Index int64  // index in tags table
	AID   uint16 // attribute ID
	What  string // error message
}

// Format error message of tag error.
func (e *TagError) Error() string {
	return fmt.Sprintf("tag index %d, attribute ID %d, %s", e.Index, e.AID, e.What)
}

// Opens package for reading. At first its checkup file signature, then
// reads records table, and reads file attributes table. Tags set
// for each file must contain at least file ID, file name and creation time.
func (pack *Package) Open(r io.ReadSeeker, filename string) (err error) {
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
	pack.Tags = make(map[string]TagSet, pack.TagNumber)

	// read records table
	if _, err = r.Seek(pack.RecOffset, io.SeekStart); err != nil {
		return
	}
	if err = binary.Read(r, binary.LittleEndian, &pack.FAT); err != nil {
		return
	}

	// read file attributes table
	if _, err = r.Seek(pack.TagOffset, io.SeekStart); err != nil {
		return
	}
	for i := int64(0); i < pack.TagNumber; i++ {
		var tags = TagSet{}
		if err = tags.Read(r); err != nil {
			return
		}
		// check tags fields
		var ok bool
		var fid uint64
		var fname string
		if fid, ok = tags.Uint64(AID_FID); !ok {
			return &TagError{i, AID_FID, "file ID is absent"}
		}
		if fid >= uint64(len(pack.FAT)) {
			return &TagError{i, AID_FID, "file ID is out of range"}
		}
		if fname, ok = tags.String(AID_name); !ok {
			return &TagError{i, AID_name, "file name is absent"}
		}
		var key = strings.ToLower(filepath.ToSlash(fname))
		if _, ok = pack.Tags[key]; ok {
			return &TagError{i, AID_name, fmt.Sprintf("file name '%s' is not unique", fname)}
		}
		if _, ok = tags.Uint64(AID_created); !ok {
			return &TagError{i, AID_created, "created time is absent"}
		}
		// insert file tags
		pack.Tags[key] = tags
	}

	return
}

func (pack *Package) PackData(w io.WriteSeeker, r io.Reader, fname string, crt int64) (tags TagSet, err error) {
	var key = strings.ToLower(filepath.ToSlash(fname))
	if _, ok := pack.Tags[key]; ok {
		err = ErrAlready
		return
	}

	var rec PackRec
	if rec.Offset, err = w.Seek(0, io.SeekCurrent); err != nil {
		return
	}
	if rec.Size, err = io.Copy(w, r); err != nil {
		return
	}

	var fid = uint64(len(pack.FAT))
	pack.FAT = append(pack.FAT, rec)
	tags = TagSet{}
	tags.SetUint64(AID_FID, fid)
	tags.SetString(AID_name, fname)
	tags.SetUint64(AID_created, uint64(crt))
	pack.Tags[key] = tags
	return
}

func (pack *Package) PackFile(w io.WriteSeeker, fname, fpath string) (tags TagSet, err error) {
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
	if tags, err = pack.PackData(w, file, fname, fi.ModTime().Unix()); err != nil {
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
