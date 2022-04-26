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
	TIDattr       TID_t = 9 // uint32

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
	TIDlabel    TID_t = 110 // string
	TIDlink     TID_t = 111 // string
	TIDkeywords TID_t = 112 // string
	TIDcategory TID_t = 113 // string
	TIDversion  TID_t = 114 // string
	TIDauthor   TID_t = 115 // string
	TIDcomment  TID_t = 116 // string
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
	ErrSignFTT = errors.New("header contains incorrect data")

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
	OpenTagset(*Tagset_t) (NestedFile, error)
	Tagset(string) (*Tagset_t, bool)
	Enum(func(string, *Tagset_t) bool)
}

// Packager refers to package data access management implementation.
type Packager interface {
	Tagger
	io.Closer
	fs.SubFS
	fs.StatFS
	fs.GlobFS
	fs.ReadFileFS
	fs.ReadDirFS
}

const (
	// SignSize - signature field size.
	SignSize = 24
	// HeaderSize - package header size in bytes.
	HeaderSize = 64
)

// Header - package header.
type Header struct {
	signature [SignSize]byte
	typesize  [8]byte // sizes of package types
	fttoffset uint64  // file tags table offset
	fttsize   uint64  // file tags table size
	datoffset uint64  // files data offset
	datsize   uint64  // files data total size
}

const (
	PTSfoffset = iota
	PTSfsize
	PTSfid
	PTStid
	PTStsize
	PTStssize
)

// PackageTypeSizes - list of type sizes used for package streaming.
var PackageTypeSizes = [8]byte{
	FOffset_l, // can be: 4, 8
	FSize_l,   // can be: 4, 8
	FID_l,     // can be: 2, 4, 8
	TID_l,     // can be: 1, 2, 4
	TSize_l,   // can be: 1, 2, 4
	TSSize_l,  // can be: 2, 4
	0,
	0,
}

// DataPos returns data files block offset and size.
func (pack *Header) DataPos() (uint64, uint64) {
	return pack.datoffset, pack.datsize
}

// IsSplitted returns true if package is splitted on tags and data files.
func (pack *Header) IsSplitted() bool {
	return pack.datoffset == 0 && pack.datsize > 0
}

// IsReady determines that package is ready for read the data.
func (pack *Header) IsReady() (err error) {
	// can not read file tags table for opened on write single-file package.
	if string(pack.signature[:]) == Prebuild && !pack.IsSplitted() {
		return ErrSignPre
	}
	// can not read file tags table on any incorrect signature
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
	if _, err = r.Read(pack.typesize[:]); err != nil {
		return
	}
	n += 8
	if err = binary.Read(r, binary.LittleEndian, &pack.fttoffset); err != nil {
		return
	}
	n += 8
	if err = binary.Read(r, binary.LittleEndian, &pack.fttsize); err != nil {
		return
	}
	n += 8
	if err = binary.Read(r, binary.LittleEndian, &pack.datoffset); err != nil {
		return
	}
	n += 8
	if err = binary.Read(r, binary.LittleEndian, &pack.datsize); err != nil {
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
	if _, err = w.Write(pack.typesize[:]); err != nil {
		return
	}
	n += 8
	if err = binary.Write(w, binary.LittleEndian, &pack.fttoffset); err != nil {
		return
	}
	n += 8
	if err = binary.Write(w, binary.LittleEndian, &pack.fttsize); err != nil {
		return
	}
	n += 8
	if err = binary.Write(w, binary.LittleEndian, &pack.datoffset); err != nil {
		return
	}
	n += 8
	if err = binary.Write(w, binary.LittleEndian, &pack.datsize); err != nil {
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
	if val, ok = ftt.Load(Normalize(fkey)); ok {
		ts = val.(*Tagset_t)
	}
	return
}

// Enum calls given closure for each tagset in package. Skips package info.
func (ftt *FTT_t) Enum(f func(string, *Tagset_t) bool) {
	ftt.Range(func(key, value interface{}) bool {
		return key.(string) == "" || f(key.(string), value.(*Tagset_t))
	})
}

// HasTagset check up that tagset with given filename key is present.
func (ftt *FTT_t) HasTagset(fkey string) (ok bool) {
	_, ok = ftt.Load(Normalize(fkey))
	return
}

// SetTagset puts tagset with given filename key.
func (ftt *FTT_t) SetTagset(fkey string, ts *Tagset_t) {
	ftt.Store(Normalize(fkey), ts)
}

// DelTagset deletes tagset with given filename key.
func (ftt *FTT_t) DelTagset(fkey string) {
	ftt.Delete(Normalize(fkey))
}

// GetDelTagset deletes the tagset for a key, returning the previous tagset if any.
func (ftt *FTT_t) GetDelTagset(fkey string) (ts *Tagset_t, ok bool) {
	var val interface{}
	if val, ok = ftt.LoadAndDelete(Normalize(fkey)); ok {
		ts = val.(*Tagset_t)
	}
	return
}

var emptyinfo = (&Tagset_t{}).
	Put(TIDfid, TagFID(0)).
	Put(TIDpath, TagString(""))

// Info returns package information tagset,
// and stores if it not present before.
func (ftt *FTT_t) Info() *Tagset_t {
	var val, _ = ftt.LoadOrStore("", &Tagset_t{emptyinfo.data})
	if val == nil {
		panic("can not obtain package info")
	}
	return val.(*Tagset_t)
}

// ReadFrom reads file tags table whole content from the given stream.
func (ftt *FTT_t) ReadFrom(r io.Reader) (n int64, err error) {
	var dataoffset FOffset_t
	var datasize FSize_t
	for {
		var tsl TSSize_t
		if err = binary.Read(r, binary.LittleEndian, &tsl); err != nil {
			return
		}
		n += TSSize_l

		if tsl == 0 {
			break // end marker was reached
		}

		var data = make([]byte, tsl)
		if _, err = r.Read(data); err != nil {
			return
		}
		n += int64(tsl)

		var ts = &Tagset_t{data}
		var tsi = ts.Iterator()
		for tsi.Next() {
		}
		if tsi.Failed() {
			err = io.ErrUnexpectedEOF
			return
		}

		var (
			ok     bool
			offset FOffset_t
			size   FSize_t
			fpath  string
		)

		// get file key
		if fpath, ok = ts.String(TIDpath); !ok {
			err = &ErrTag{ErrNoPath, "", TIDpath}
			return
		}
		if ftt.HasTagset(fpath) {
			err = &ErrTag{fs.ErrExist, fpath, TIDpath}
			return
		}

		// check system tags
		if offset, ok = ts.FOffset(); !ok && fpath != "" {
			err = &ErrTag{ErrNoOffset, fpath, TIDoffset}
			return
		}
		if size, ok = ts.FSize(); !ok && fpath != "" {
			err = &ErrTag{ErrNoSize, fpath, TIDsize}
			return
		}
		if !ts.Has(TIDfid) {
			err = &ErrTag{ErrNoFID, fpath, TIDfid}
			return
		}

		// setup whole package offset and size
		if fpath == "" {
			dataoffset, datasize = offset, size
		}

		// check up offset and tag if package info is provided
		if datasize > 0 {
			if offset < dataoffset || offset > dataoffset+FOffset_t(datasize) {
				err = &ErrTag{ErrOutOff, fpath, TIDoffset}
				return
			}
			if offset+FOffset_t(size) > dataoffset+FOffset_t(datasize) {
				err = &ErrTag{ErrOutSize, fpath, TIDsize}
				return
			}
		}

		ftt.SetTagset(fpath, ts)
	}
	return
}

// WriteTo writes file tags table whole content to the given stream.
func (ftt *FTT_t) WriteTo(w io.Writer) (n int64, err error) {
	// write tagset with package info at first
	if ts, ok := ftt.Tagset(""); ok {
		var tsl = len(ts.Data())

		// write tagset length
		if err = binary.Write(w, binary.LittleEndian, TSSize_t(tsl)); err != nil {
			return
		}
		n += TSSize_l

		// write tagset content
		if _, err = w.Write(ts.Data()); err != nil {
			return
		}
		n += int64(tsl)
	}

	// write files tags table
	ftt.Enum(func(fkey string, ts *Tagset_t) bool {
		var tsl = len(ts.Data())

		// write tagset length
		if err = binary.Write(w, binary.LittleEndian, TSSize_t(tsl)); err != nil {
			return false
		}
		n += TSSize_l

		// write tagset content
		if _, err = w.Write(ts.Data()); err != nil {
			return false
		}
		n += int64(tsl)
		return true
	})
	if err != nil {
		return
	}
	// write tags table end marker
	if err = binary.Write(w, binary.LittleEndian, TSSize_t(0)); err != nil {
		return
	}
	n += TSSize_l
	return
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

// Opens package for reading. At first it checkups file signature, then
// reads records table, and reads file tagset table. Tags set for each
// file must contain at least file offset, file size, file ID and file name.
func (pack *Package) OpenFTT(r io.ReadSeeker) (err error) {
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

	// go to file tags table start
	if _, err = r.Seek(int64(pack.fttoffset), io.SeekStart); err != nil {
		return
	}
	// setup empty tags table
	pack.FTT_t = FTT_t{}
	// read file tags table
	var fttsize int64
	if fttsize, err = pack.FTT_t.ReadFrom(r); err != nil {
		return
	}
	if fttsize != int64(pack.fttsize) {
		err = ErrSignFTT
		return
	}
	return
}

// GetPackageInfo returns tagset with package information.
// It's a quick function to get info from the file.
func GetPackageInfo(r io.ReadSeeker) (ts *Tagset_t, err error) {
	var hdr Header
	// go to file start
	if _, err = r.Seek(0, io.SeekStart); err != nil {
		return
	}
	// read header
	if _, err = hdr.ReadFrom(r); err != nil {
		return
	}
	if err = hdr.IsReady(); err != nil {
		return
	}

	// go to file tags table start
	if _, err = r.Seek(int64(hdr.fttoffset), io.SeekStart); err != nil {
		return
	}

	// read first tagset that can be package info,
	// or some file tagset if info is absent
	var tsl TSSize_t
	if err = binary.Read(r, binary.LittleEndian, &tsl); err != nil {
		return
	}
	if tsl == 0 {
		return // end marker was reached
	}

	var data = make([]byte, tsl)
	if _, err = r.Read(data); err != nil {
		return
	}

	ts = &Tagset_t{data}
	var tsi = ts.Iterator()
	for tsi.Next() {
	}
	if tsi.Failed() {
		err = io.ErrUnexpectedEOF
		return
	}

	// get file key
	var ok bool
	var fpath string
	if fpath, ok = ts.String(TIDpath); !ok {
		err = ErrNoPath
		return
	}
	if fpath != "" {
		ts = nil // info is not found
		return
	}
	return
}

// The End.
