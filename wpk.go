package wpk

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"path/filepath"
	"sync"
)

// File format signatures.
const (
	Signature = "Whirlwind 3.4 Package   " // package is ready for use
	Prebuild  = "Whirlwind 3.4 Prebuild  " // package is in building progress
)

type (
	// TID_t - tag identifier type.
	TID_t uint16
	// TSSize_t - tagset size type.
	TSSize_t uint16
	// FID_t - file index/identifier type.
	FID_t uint32
	// Offset_t - data block offset type.
	Offset_t uint64
	// Size_t - data block size type.
	Size_t uint64
)

// List of predefined tags IDs.
const (
	TIDnone TID_t = 0

	TIDoffset     TID_t = 1 // required, uint64
	TIDsize       TID_t = 2 // required, uint64
	TIDfid        TID_t = 3 // required, uint32
	TIDpath       TID_t = 4 // required, unique, string
	TIDcreated    TID_t = 5 // required for files, uint64
	TIDlastwrite  TID_t = 6 // uint64
	TIDlastaccess TID_t = 7 // uint64
	TIDmime       TID_t = 8 // string
	TIDfileattr   TID_t = 9 // uint32

	TIDconst TID_t = 4  // marker of tags that should be unchanged
	TIDsys   TID_t = 10 // system protection marker

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

	TIDlink     TID_t = 100 // string
	TIDkeywords TID_t = 101 // string
	TIDcategory TID_t = 102 // string
	TIDversion  TID_t = 103 // string
	TIDauthor   TID_t = 104 // string
	TIDcomment  TID_t = 105 // string
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
}

// Reset initializes fields with zero values and sets
// prebuild signature. Label remains unchanged.
func (pack *Header) Reset() {
	copy(pack.signature[:], Prebuild)
	pack.fttoffset = HeaderSize
	pack.fttsize = 0
}

// Label returns string with disk label, copied from header fixed field.
// Maximum length of label is 24 bytes.
func (pack *Header) Label() string {
	var i int
	for ; i < LabelSize && pack.disklabel[i] > 0; i++ {
	}
	return string(pack.disklabel[:i])
}

// SetLabel setups header fixed label field to given string.
// Maximum length of label is 24 bytes.
func (pack *Header) SetLabel(label string) {
	for i := copy(pack.disklabel[:], []byte(label)); i < LabelSize; i++ {
		pack.disklabel[i] = 0 // make label zero-terminated
	}
}

// DataSize returns sum size of all real stored records in package.
func (pack *Header) DataSize() Size_t {
	if pack.fttoffset > HeaderSize {
		return Size_t(pack.fttoffset - HeaderSize)
	}
	return 0
}

// IsReady determines that package is ready for read the data.
func (pack *Header) IsReady() (err error) {
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
		var tsi TagsetIterator
		tsi.data = data
		tsi.Reset()
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
