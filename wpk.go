package wpk

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"path/filepath"
	"sync"
)

// File format signatures.
const (
	Signature = "Whirlwind 3.4 Package   " // package is ready for use
	Prebuild  = "Whirlwind 3.4 Prebuild  " // package is in building progress
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

	TIDconst TID_t = 4 // marker of tags that should be unchanged
	TIDsys   TID_t = 8 // system protection marker

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

	TIDtmbimg   TID_t = 100 // []byte, thumbnail image (icon)
	TIDtmbmime  TID_t = 101 // string, MIME type of thumbnail image
	TIDlink     TID_t = 110 // string
	TIDkeywords TID_t = 111 // string
	TIDcategory TID_t = 112 // string
	TIDversion  TID_t = 113 // string
	TIDauthor   TID_t = 114 // string
	TIDcomment  TID_t = 115 // string
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
	Tagset(string) (*Tagset_t, bool)
	Enum(func(string, *Tagset_t) bool)
}

// Packager refers to package data access management implementation.
type Packager interface {
	DataSize() FSize_t
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
	SignSize = 24
	// NameSize - disk label field size.
	NameSize = 32
)

// Header - package header.
type Header struct {
	signature [SignSize]byte
	disklabel [NameSize]byte
	fttoffset uint64  // file tags table offset
	fttsize   uint64  // file tags table size
	typesize  [8]byte // sizes of package types
}

// PackageTypeSizes - list of type sizes used for package streaming.
var PackageTypeSizes = [8]byte{
	FOffset_l,
	FSize_l,
	FID_l,
	TID_l,
	TSize_l,
	TSSize_l,
	0,
	0,
}

// Reset initializes fields with zero values and sets
// prebuild signature. Label remains unchanged.
func (pack *Header) Reset() {
	copy(pack.signature[:], Prebuild)
	pack.fttoffset = HeaderSize
	pack.fttsize = 0
	pack.typesize = PackageTypeSizes
}

// Label returns string with disk label, copied from header fixed field.
// Maximum length of label is 24 bytes.
func (pack *Header) Label() string {
	var i int
	for ; i < NameSize && pack.disklabel[i] > 0; i++ {
	}
	return string(pack.disklabel[:i])
}

// SetLabel setups header fixed label field to given string.
// Maximum length of label is 24 bytes.
func (pack *Header) SetLabel(label string) {
	for i := copy(pack.disklabel[:], []byte(label)); i < NameSize; i++ {
		pack.disklabel[i] = 0 // make label zero-terminated
	}
}

// DataSize returns sum size of all real stored records in package.
func (pack *Header) DataSize() FSize_t {
	if pack.fttoffset > HeaderSize {
		return FSize_t(pack.fttoffset - HeaderSize)
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
	n += NameSize
	if err = binary.Read(r, binary.LittleEndian, &pack.fttoffset); err != nil {
		return
	}
	n += 8
	if err = binary.Read(r, binary.LittleEndian, &pack.fttsize); err != nil {
		return
	}
	n += 8
	if _, err = r.Read(pack.typesize[:]); err != nil {
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
	n += NameSize
	if err = binary.Write(w, binary.LittleEndian, &pack.fttoffset); err != nil {
		return
	}
	n += 8
	if err = binary.Write(w, binary.LittleEndian, &pack.fttsize); err != nil {
		return
	}
	n += 8
	if _, err = w.Write(pack.typesize[:]); err != nil {
		return
	}
	n += 8
	return
}

// File tags table.
// Keys - package filenames in lower case, values - tagset slices.
type FTT_t struct {
	sync.Map
}

// Tagset returns tagset with given filename key, if it found.
func (ftt *FTT_t) Tagset(fkey string) (ts *Tagset_t, ok bool) {
	var val interface{}
	if val, ok = ftt.Load(fkey); ok {
		ts = val.(*Tagset_t)
	}
	return
}

// Enum calls given closure for each tagset in package.
func (ftt *FTT_t) Enum(f func(string, *Tagset_t) bool) {
	ftt.Range(func(key, value interface{}) bool {
		return f(key.(string), value.(*Tagset_t))
	})
}

// HasTagset check up that tagset with given filename key is present.
func (ftt *FTT_t) HasTagset(fkey string) (ok bool) {
	_, ok = ftt.Load(fkey)
	return
}

// SetTagset puts tagset with given filename key.
func (ftt *FTT_t) SetTagset(fkey string, ts *Tagset_t) {
	ftt.Store(fkey, ts)
}

// DelTagset deletes tagset with given filename key.
func (ftt *FTT_t) DelTagset(fkey string) {
	ftt.Delete(fkey)
}

// Package structure contains all data needed for package representation.
type Package struct {
	Header
	FTT_t
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

	// read file tags table
	if _, err = r.Seek(int64(pack.fttoffset), io.SeekStart); err != nil {
		return
	}

	// setup empty tags table
	pack.FTT_t = FTT_t{}

	// read whole ftt slice at once
	var fttbulk = make([]byte, pack.fttsize)
	if _, err = r.Read(fttbulk); err != nil {
		return
	}

	var tspos int64
	for {
		if tspos == int64(pack.fttsize) {
			break // end bound was reached
		}
		if tspos += TSSize_l; tspos > int64(pack.fttsize) {
			return io.EOF
		}
		var tsl = int64(TSSize_r(fttbulk[tspos-TSSize_l : tspos]))
		if tsl == 0 {
			break // end marker was reached
		}

		var ts = MakeTagset(fttbulk[tspos : tspos+tsl])
		var tsi = ts.Iterator()
		for tsi.Next() {
		}
		if tsi.Failed() {
			return io.EOF
		}

		// get file key and check tags fields
		var (
			ok     bool
			fkey   string
			fpath  string
			offset FOffset_t
			size   FSize_t
		)
		if fpath, ok = ts.String(TIDpath); !ok {
			return &ErrTag{ErrNoPath, "", TIDpath}
		}
		fkey = Normalize(fpath)
		if pack.HasTagset(fkey) {
			return &ErrTag{fs.ErrExist, fkey, TIDpath}
		}
		if !ts.Has(TIDfid) {
			return &ErrTag{ErrNoFID, fkey, TIDfid}
		}
		if offset, ok = ts.FOffset(); !ok {
			return &ErrTag{ErrNoOffset, fkey, TIDoffset}
		}
		if size, ok = ts.FSize(); !ok {
			return &ErrTag{ErrNoSize, fkey, TIDsize}
		}
		if offset < HeaderSize || uint64(offset) >= pack.fttoffset {
			log.Printf("!!! offset=%d", offset)
			return &ErrTag{ErrOutOff, fkey, TIDoffset}
		}
		if uint64(offset)+uint64(size) > pack.fttoffset {
			return &ErrTag{ErrOutSize, fkey, TIDsize}
		}

		pack.SetTagset(fkey, ts)
		tspos += tsl
	}
	return
}

// The End.
