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
	TIDnone = 0

	TIDoffset = 1  // required, uint64
	TIDsize   = 2  // required, uint64
	TIDfid    = 3  // required, uint32
	TIDpath   = 4  // required, unique, string
	TIDmtime  = 5  // required for files, 8/12 bytes (mod-time)
	TIDatime  = 6  // 8/12 bytes (access-time)
	TIDctime  = 7  // 8/12 bytes (change-time)
	TIDbtime  = 8  // 8/12 bytes (birth-time)
	TIDattr   = 9  // uint32
	TIDmime   = 10 // string

	TIDconst = 4 // marker of tags that should be unchanged
	TIDsys   = 8 // system protection marker

	TIDcrc32ieee = 11 // uint32, CRC-32-IEEE 802.3, poly = 0x04C11DB7, init = -1
	TIDcrc32c    = 12 // uint32, (Castagnoli), poly = 0x1EDC6F41, init = -1
	TIDcrc32k    = 13 // uint32, (Koopman), poly = 0x741B8CD7, init = -1
	TIDcrc64iso  = 14 // uint64, poly = 0xD800000000000000, init = -1

	TIDmd5    = 20 // [16]byte
	TIDsha1   = 21 // [20]byte
	TIDsha224 = 22 // [28]byte
	TIDsha256 = 23 // [32]byte
	TIDsha384 = 24 // [48]byte
	TIDsha512 = 25 // [64]byte

	TIDtmbimg   = 100 // []byte, thumbnail image (icon)
	TIDtmbmime  = 101 // string, MIME type of thumbnail image
	TIDlabel    = 110 // string
	TIDlink     = 111 // string
	TIDkeywords = 112 // string
	TIDcategory = 113 // string
	TIDversion  = 114 // string
	TIDauthor   = 115 // string
	TIDcomment  = 116 // string
)

// ErrTag is error on some field of tags set.
type ErrTag[TID_t TID_i] struct {
	What error  // error message
	Key  string // normalized file name
	TID  TID_t  // tag ID
}

func (e *ErrTag[TID_t]) Error() string {
	return fmt.Sprintf("key '%s', tag ID %d: %s", e.Key, e.TID, e.What.Error())
}

func (e *ErrTag[TID_t]) Unwrap() error {
	return e.What
}

// Errors on WPK-API.
var (
	ErrSignPre = errors.New("package is not ready")
	ErrSignBad = errors.New("signature does not pass")
	ErrSignFTT = errors.New("header contains incorrect data")

	ErrSizeFOffset = errors.New("size of file offset type is differs from expected")
	ErrSizeFSize   = errors.New("size of file size type is differs from expected")
	ErrSizeFID     = errors.New("size of file ID type is differs from expected")
	ErrSizeTID     = errors.New("size of tag ID type is differs from expected")
	ErrSizeTSize   = errors.New("size of tag size type is differs from expected")
	ErrSizeTSSize  = errors.New("size of tagset size type is differs from expected")
	ErrCondFSize   = errors.New("size of file size type should be not more than file offset size")
	ErrCondTID     = errors.New("size of tag ID type should be not more than tagset size")
	ErrCondTSize   = errors.New("size of tag size type should be not more than tagset size")

	ErrNoTag    = errors.New("tag with given ID not found")
	ErrNoPath   = errors.New("file name is absent")
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
type Tagger[TID_t TID_i, TSize_t TSize_i] interface {
	OpenTagset(*Tagset_t[TID_t, TSize_t]) (NestedFile, error)
	Tagset(string) (*Tagset_t[TID_t, TSize_t], bool)
	Enum(func(string, *Tagset_t[TID_t, TSize_t]) bool)
}

// Packager refers to package data access management implementation.
type Packager[TID_t TID_i, TSize_t TSize_i] interface {
	Tagger[TID_t, TSize_t]
	io.Closer
	fs.SubFS
	fs.StatFS
	fs.GlobFS
	fs.ReadFileFS
	fs.ReadDirFS
}

const (
	SignSize   = 24 // SignSize - signature field size.
	HeaderSize = 64 // HeaderSize - package header size in bytes.
)

// TypeSize is set of package types sizes.
type TypeSize [8]byte

const (
	PTSfoffset = iota // Index of "file offset" type size.
	PTSfsize          // Index of "file size" type size.
	PTSfid            // Index of "file ID" type size.
	PTStid            // Index of "tag ID" type size.
	PTStsize          // Index of "tag size" type size.
	PTStssize         // Index of "tagset size" type size.
)

// Checkup performs check up sizes of all types in bytes
// used in current WPK-file.
func (pts TypeSize) Checkup(FOffset_l, FSize_l, TID_l, TSize_l byte) error {
	if pts[PTSfoffset] != FOffset_l {
		return ErrSizeFOffset
	}
	if pts[PTSfsize] != FSize_l {
		return ErrSizeFSize
	}

	switch pts[PTSfid] {
	case 2, 4, 8:
	default:
		return ErrSizeFID
	}

	if pts[PTStid] != TID_l {
		return ErrSizeTID
	}
	if pts[PTStsize] != TSize_l {
		return ErrSizeTSize
	}

	switch pts[PTStssize] {
	case 2, 4:
	default:
		return ErrSizeTSSize
	}

	if pts[PTSfsize] > pts[PTSfoffset] {
		return ErrCondFSize
	}
	if pts[PTStid] > pts[PTStssize] {
		return ErrCondTID
	}
	if pts[PTStsize] > pts[PTStssize] {
		return ErrCondTSize
	}
	return nil
}

// Header - package header.
type Header struct {
	signature [SignSize]byte
	typesize  TypeSize // sizes of package types
	fttoffset uint64   // file tags table offset
	fttsize   uint64   // file tags table size
	datoffset uint64   // files data offset
	datsize   uint64   // files data total size
}

// PTS is getter for sizes of package types.
func (hdr *Header) PTS(idx int) byte {
	return hdr.typesize[idx]
}

// DataPos returns data files block offset and size.
func (hdr *Header) DataPos() (uint64, uint64) {
	return hdr.datoffset, hdr.datsize
}

// IsSplitted returns true if package is splitted on tags and data files.
func (hdr *Header) IsSplitted() bool {
	return hdr.datoffset == 0 && hdr.datsize > 0
}

// IsReady determines that package is ready for read the data.
func (hdr *Header) IsReady() (err error) {
	// can not read file tags table for opened on write single-file package.
	if string(hdr.signature[:]) == Prebuild && !hdr.IsSplitted() {
		return ErrSignPre
	}
	// can not read file tags table on any incorrect signature
	if string(hdr.signature[:]) != Signature {
		return ErrSignBad
	}
	return
}

// ReadFrom reads header from stream as binary data of constant length in little endian order.
func (hdr *Header) ReadFrom(r io.Reader) (n int64, err error) {
	if err = binary.Read(r, binary.LittleEndian, hdr.signature[:]); err != nil {
		return
	}
	n += SignSize
	if _, err = r.Read(hdr.typesize[:]); err != nil {
		return
	}
	n += 8
	if err = binary.Read(r, binary.LittleEndian, &hdr.fttoffset); err != nil {
		return
	}
	n += 8
	if err = binary.Read(r, binary.LittleEndian, &hdr.fttsize); err != nil {
		return
	}
	n += 8
	if err = binary.Read(r, binary.LittleEndian, &hdr.datoffset); err != nil {
		return
	}
	n += 8
	if err = binary.Read(r, binary.LittleEndian, &hdr.datsize); err != nil {
		return
	}
	n += 8
	return
}

// WriteTo writes header to stream as binary data of constant length in little endian order.
func (hdr *Header) WriteTo(w io.Writer) (n int64, err error) {
	if err = binary.Write(w, binary.LittleEndian, hdr.signature[:]); err != nil {
		return
	}
	n += SignSize
	if _, err = w.Write(hdr.typesize[:]); err != nil {
		return
	}
	n += 8
	if err = binary.Write(w, binary.LittleEndian, &hdr.fttoffset); err != nil {
		return
	}
	n += 8
	if err = binary.Write(w, binary.LittleEndian, &hdr.fttsize); err != nil {
		return
	}
	n += 8
	if err = binary.Write(w, binary.LittleEndian, &hdr.datoffset); err != nil {
		return
	}
	n += 8
	if err = binary.Write(w, binary.LittleEndian, &hdr.datsize); err != nil {
		return
	}
	n += 8
	return
}

// File tags table.
// Keys - package filenames in lower case, values - tagset slices.
type FTT_t[TID_t, TSize_t TSize_i] struct {
	sync.Map
	tssize byte
}

// Tagset returns tagset with given filename key, if it found.
func (ftt *FTT_t[TID_t, TSize_t]) Tagset(fkey string) (ts *Tagset_t[TID_t, TSize_t], ok bool) {
	var val interface{}
	if val, ok = ftt.Load(Normalize(fkey)); ok {
		ts = val.(*Tagset_t[TID_t, TSize_t])
	}
	return
}

// Enum calls given closure for each tagset in package. Skips package info.
func (ftt *FTT_t[TID_t, TSize_t]) Enum(f func(string, *Tagset_t[TID_t, TSize_t]) bool) {
	ftt.Range(func(key, value interface{}) bool {
		return key.(string) == "" || f(key.(string), value.(*Tagset_t[TID_t, TSize_t]))
	})
}

// HasTagset check up that tagset with given filename key is present.
func (ftt *FTT_t[TID_t, TSize_t]) HasTagset(fkey string) (ok bool) {
	_, ok = ftt.Load(Normalize(fkey))
	return
}

// SetTagset puts tagset with given filename key.
func (ftt *FTT_t[TID_t, TSize_t]) SetTagset(fkey string, ts *Tagset_t[TID_t, TSize_t]) {
	ftt.Store(Normalize(fkey), ts)
}

// DelTagset deletes tagset with given filename key.
func (ftt *FTT_t[TID_t, TSize_t]) DelTagset(fkey string) {
	ftt.Delete(Normalize(fkey))
}

// GetDelTagset deletes the tagset for a key, returning the previous tagset if any.
func (ftt *FTT_t[TID_t, TSize_t]) GetDelTagset(fkey string) (ts *Tagset_t[TID_t, TSize_t], ok bool) {
	var val interface{}
	if val, ok = ftt.LoadAndDelete(Normalize(fkey)); ok {
		ts = val.(*Tagset_t[TID_t, TSize_t])
	}
	return
}

// Info returns package information tagset,
// and stores if it not present before.
func (ftt *FTT_t[TID_t, TSize_t]) Info() *Tagset_t[TID_t, TSize_t] {
	var emptyinfo = (&Tagset_t[TID_t, TSize_t]{}).
		Put(TIDpath, TagString(""))
	var val, _ = ftt.LoadOrStore("", &Tagset_t[TID_t, TSize_t]{emptyinfo.data})
	if val == nil {
		panic("can not obtain package info")
	}
	return val.(*Tagset_t[TID_t, TSize_t])
}

type filepos struct {
	offset uint
	size   uint
}

func (ftt *FTT_t[TID_t, TSize_t]) checkTagset(ts *Tagset_t[TID_t, TSize_t], lim *filepos) (fpath string, err error) {
	var ok bool
	var pos filepos

	// get file key
	if fpath, ok = ts.String(TIDpath); !ok {
		err = &ErrTag[TID_t]{ErrNoPath, "", TIDpath}
		return
	}
	if ftt.HasTagset(fpath) { // prevent same file from repeating
		err = &ErrTag[TID_t]{fs.ErrExist, fpath, TIDpath}
		return
	}

	// check system tags
	if pos.offset, ok = ts.Uint(TIDoffset); !ok && fpath != "" {
		err = &ErrTag[TID_t]{ErrNoOffset, fpath, TIDoffset}
		return
	}
	if pos.size, ok = ts.Uint(TIDsize); !ok && fpath != "" {
		err = &ErrTag[TID_t]{ErrNoSize, fpath, TIDsize}
		return
	}

	if fpath == "" { // setup whole package offset and size
		lim.offset, lim.size = pos.offset, pos.size
	} else if lim.size > 0 { // check up offset and tag if package info is provided
		if pos.offset < lim.offset || pos.offset > lim.offset+lim.size {
			err = &ErrTag[TID_t]{ErrOutOff, fpath, TIDoffset}
			return
		}
		if pos.offset+pos.size > lim.offset+lim.size {
			err = &ErrTag[TID_t]{ErrOutSize, fpath, TIDsize}
			return
		}
	}

	return
}

// ReadFrom reads file tags table whole content from the given stream.
func (ftt *FTT_t[TID_t, TSize_t]) ReadFrom(r io.Reader) (n int64, err error) {
	var limits filepos
	for {
		var tsl uint
		if tsl, err = ReadUint(r, ftt.tssize); err != nil {
			return
		}
		n += int64(ftt.tssize)

		if tsl == 0 {
			break // end marker was reached
		}

		var data = make([]byte, tsl)
		if _, err = r.Read(data); err != nil {
			return
		}
		n += int64(tsl)

		var ts = &Tagset_t[TID_t, TSize_t]{data}
		var tsi = ts.Iterator()
		for tsi.Next() {
		}
		if tsi.Failed() {
			err = io.ErrUnexpectedEOF
			return
		}

		var fpath string
		if fpath, err = ftt.checkTagset(ts, &limits); err != nil {
			return
		}

		ftt.SetTagset(fpath, ts)
	}
	return
}

// WriteTo writes file tags table whole content to the given stream.
func (ftt *FTT_t[TID_t, TSize_t]) WriteTo(w io.Writer) (n int64, err error) {
	// write tagset with package info at first
	if ts, ok := ftt.Tagset(""); ok {
		var tsl = uint(len(ts.Data()))

		// write tagset length
		if err = WriteUint(w, tsl, ftt.tssize); err != nil {
			return
		}
		n += int64(ftt.tssize)

		// write tagset content
		if _, err = w.Write(ts.Data()); err != nil {
			return
		}
		n += int64(tsl)
	}

	// write files tags table
	ftt.Enum(func(fkey string, ts *Tagset_t[TID_t, TSize_t]) bool {
		var tsl = uint(len(ts.Data()))

		// write tagset length
		if err = WriteUint(w, tsl, ftt.tssize); err != nil {
			return false
		}
		n += int64(ftt.tssize)

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
	if err = WriteUint(w, 0, ftt.tssize); err != nil {
		return
	}
	n += int64(ftt.tssize)
	return
}

// Package structure contains all data needed for package representation.
type Package[TID_t TID_i, TSize_t TSize_i] struct {
	Header
	FTT_t[TID_t, TSize_t]
	mux sync.Mutex // writer mutex
}

// NewPackage returns pointer to new initialized Package structure.
func NewPackage[TID_t TID_i, TSize_t TSize_i](fidsz, tssize byte) (pack *Package[TID_t, TSize_t]) {
	pack = &Package[TID_t, TSize_t]{}
	pack.Init(fidsz, tssize)
	return
}

// Init performs initialization for given Package structure.
func (pack *Package[TID_t, TSize_t]) Init(fidsz, tssize byte) {
	pack.Header.typesize = TypeSize{
		Uint_l[FOffset_t](), // can be: 4, 8
		Uint_l[FSize_t](),   // can be: 4, 8
		fidsz,               // can be: 2, 4, 8
		Uint_l[TID_t](),     // can be: 1, 2, 4
		Uint_l[TSize_t](),   // can be: 1, 2, 4
		tssize,              // can be: 2, 4
		0,
		0,
	}
	pack.FTT_t.tssize = tssize
}

// Glob returns the names of all files in package matching pattern or nil
// if there is no matching file.
func (pack *Package[TID_t, TSize_t]) Glob(pattern string) (res []string, err error) {
	pattern = Normalize(pattern)
	if _, err = filepath.Match(pattern, ""); err != nil {
		return
	}
	pack.Enum(func(fkey string, ts *Tagset_t[TID_t, TSize_t]) bool {
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
func (pack *Package[TID_t, TSize_t]) OpenFTT(r io.ReadSeeker) (err error) {
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
	if err = pack.typesize.Checkup(
		Uint_l[FOffset_t](),
		Uint_l[FSize_t](),
		Uint_l[TID_t](),
		Uint_l[TSize_t](),
	); err != nil {
		return
	}
	// setup empty tags table
	pack.FTT_t = FTT_t[TID_t, TSize_t]{
		tssize: pack.Header.typesize[PTStssize],
	}
	// go to file tags table start
	if _, err = r.Seek(int64(pack.fttoffset), io.SeekStart); err != nil {
		return
	}
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
func GetPackageInfo[TID_t TID_i, TSize_t TSize_i](r io.ReadSeeker) (ts *Tagset_t[TID_t, TSize_t], err error) {
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
	if err = hdr.typesize.Checkup(
		Uint_l[FOffset_t](),
		Uint_l[FSize_t](),
		Uint_l[TID_t](),
		Uint_l[TSize_t](),
	); err != nil {
		return
	}

	// go to file tags table start
	if _, err = r.Seek(int64(hdr.fttoffset), io.SeekStart); err != nil {
		return
	}

	// read first tagset that can be package info,
	// or some file tagset if info is absent
	var tsl uint
	if tsl, err = ReadUint(r, hdr.typesize[PTStssize]); err != nil {
		return
	}
	if tsl == 0 {
		return // end marker was reached
	}

	var data = make([]byte, tsl)
	if _, err = r.Read(data); err != nil {
		return
	}

	ts = &Tagset_t[TID_t, TSize_t]{data}
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
