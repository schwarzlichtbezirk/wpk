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

	TIDoffset = 1  // required, defenid at TypeSize
	TIDsize   = 2  // required, defenid at TypeSize
	TIDpath   = 3  // required, unique, string
	TIDfid    = 4  // unique, defenid at TypeSize
	TIDmtime  = 5  // required for files, 8/12 bytes (mod-time)
	TIDatime  = 6  // 8/12 bytes (access-time)
	TIDctime  = 7  // 8/12 bytes (change-time)
	TIDbtime  = 8  // 8/12 bytes (birth-time)
	TIDattr   = 9  // uint32
	TIDmime   = 10 // string

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
type ErrTag struct {
	What error  // error message
	Key  string // normalized file name
	TID  uint   // tag ID
}

func (e *ErrTag) Error() string {
	return fmt.Sprintf("key '%s', tag ID %d: %s", e.Key, e.TID, e.What.Error())
}

func (e *ErrTag) Unwrap() error {
	return e.What
}

// Errors on WPK-API.
var (
	ErrSignPre = errors.New("package is not ready")
	ErrSignBad = errors.New("signature does not pass")
	ErrSignFTT = errors.New("header contains incorrect data")

	ErrSizeFOffset = errors.New("size of file offset type is not in set {4, 8}")
	ErrSizeFSize   = errors.New("size of file size type is not in set {4, 8}")
	ErrSizeFID     = errors.New("size of file ID type is not in set {2, 4, 8}")
	ErrSizeTID     = errors.New("size of tag ID type is not in set {1, 2, 4}")
	ErrSizeTSize   = errors.New("size of tag size type is not in set {1, 2, 4}")
	ErrSizeTSSize  = errors.New("size of tagset size type is not in set {2, 4}")
	ErrCondFSize   = errors.New("size of file size type should be not more than file offset size")
	ErrCondTID     = errors.New("size of tag ID type should be not more than tagset size")
	ErrCondTSize   = errors.New("size of tag size type should be not more than tagset size")
	ErrRangeTSSize = errors.New("tagset size value is exceeds out of the type dimension")

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
type Tagger interface {
	OpenTagset(*Tagset_t) (NestedFile, error)
	Tagset(string) (*Tagset_t, bool)
	Enum(func(string, *Tagset_t) bool)
}

// CompleteFS includes all FS interfaces.
type CompleteFS interface {
	io.Closer
	fs.SubFS
	fs.StatFS
	fs.GlobFS
	fs.ReadFileFS
	fs.ReadDirFS
}

// Packager refers to package data access management implementation.
type Packager interface {
	CompleteFS
	Tagger
}

const (
	SignSize   = 24 // SignSize - signature field size.
	HeaderSize = 64 // HeaderSize - package header size in bytes.
)

// TypeSize is set of package types sizes.
type TypeSize [8]byte

const (
	PTStidsz  = iota // Index of "tag ID" type size.
	PTStagsz         // Index of "tag size" type size.
	PTStssize        // Index of "tagset size" type size.
)

// Checkup performs check up sizes of all types in bytes
// used in current WPK-file.
func (pts TypeSize) Checkup() error {
	switch pts[PTStidsz] {
	case 1, 2, 4:
	default:
		return ErrSizeTID
	}

	switch pts[PTStagsz] {
	case 1, 2, 4:
	default:
		return ErrSizeTSize
	}

	switch pts[PTStssize] {
	case 2, 4:
	default:
		return ErrSizeTSSize
	}

	if pts[PTStidsz] > pts[PTStssize] {
		return ErrCondTID
	}
	if pts[PTStagsz] > pts[PTStssize] {
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
type FTT_t struct {
	sync.Map
	tidsz  byte
	tagsz  byte
	tssize byte
}

// NewTagset creates new empty tagset based on predefined
// TID type size and tag size type.
func (ftt *FTT_t) NewTagset() *Tagset_t {
	return &Tagset_t{nil, ftt.tidsz, ftt.tagsz}
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

// Info returns package information tagset,
// and stores if it not present before.
func (ftt *FTT_t) Info() *Tagset_t {
	var emptyinfo = ftt.NewTagset().
		Put(TIDpath, TagString(""))
	var val, _ = ftt.LoadOrStore("", emptyinfo)
	if val == nil {
		panic("can not obtain package info")
	}
	return val.(*Tagset_t)
}

type filepos struct {
	offset uint
	size   uint
}

func (ftt *FTT_t) checkTagset(ts *Tagset_t, lim *filepos) (fpath string, err error) {
	var ok bool
	var pos filepos

	// get file key
	if fpath, ok = ts.String(TIDpath); !ok {
		err = &ErrTag{ErrNoPath, "", TIDpath}
		return
	}
	if ftt.HasTagset(fpath) { // prevent same file from repeating
		err = &ErrTag{fs.ErrExist, fpath, TIDpath}
		return
	}

	// check system tags
	if pos.offset, ok = ts.Uint(TIDoffset); !ok && fpath != "" {
		err = &ErrTag{ErrNoOffset, fpath, TIDoffset}
		return
	}
	if pos.size, ok = ts.Uint(TIDsize); !ok && fpath != "" {
		err = &ErrTag{ErrNoSize, fpath, TIDsize}
		return
	}

	if fpath == "" { // setup whole package offset and size
		lim.offset, lim.size = pos.offset, pos.size
	} else if lim.size > 0 { // check up offset and tag if package info is provided
		if pos.offset < lim.offset || pos.offset > lim.offset+lim.size {
			err = &ErrTag{ErrOutOff, fpath, TIDoffset}
			return
		}
		if pos.offset+pos.size > lim.offset+lim.size {
			err = &ErrTag{ErrOutSize, fpath, TIDsize}
			return
		}
	}

	return
}

// ReadFrom reads file tags table whole content from the given stream.
func (ftt *FTT_t) ReadFrom(r io.Reader) (n int64, err error) {
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

		var ts = &Tagset_t{data, ftt.tidsz, ftt.tagsz}
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
func (ftt *FTT_t) WriteTo(w io.Writer) (n int64, err error) {
	// write tagset with package info at first
	if ts, ok := ftt.Tagset(""); ok {
		var tsl = uint(len(ts.Data()))
		if tsl > uint(1<<(ftt.tssize*8)-1) {
			err = ErrRangeTSSize
			return
		}

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
	ftt.Enum(func(fkey string, ts *Tagset_t) bool {
		var tsl = uint(len(ts.Data()))
		if tsl > uint(1<<(ftt.tssize*8)-1) {
			err = ErrRangeTSSize
			return false
		}

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
type Package struct {
	Header
	FTT_t
	mux sync.Mutex // writer mutex
}

// NewPackage returns pointer to new initialized Package structure.
func NewPackage(pts TypeSize) (pack *Package) {
	pack = &Package{}
	pack.Init(pts)
	return
}

// Init performs initialization for given Package structure.
func (pack *Package) Init(pts TypeSize) {
	pack.Header.typesize = pts
	pack.FTT_t.tidsz = pts[PTStidsz]
	pack.FTT_t.tagsz = pts[PTStagsz]
	pack.FTT_t.tssize = pts[PTStssize]
}

// BaseTagset returns new tagset based on predefined TID type size and tag size type,
// and puts file offset and file size into tagset with predefined sizes.
func (pack *Package) BaseTagset(offset, size uint, fpath string) *Tagset_t {
	var ts = pack.NewTagset()
	return ts.
		Put(TIDoffset, TagUint(offset)).
		Put(TIDsize, TagUint(size)).
		Put(TIDpath, TagString(ToSlash(fpath)))
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
	if err = pack.typesize.Checkup(); err != nil {
		return
	}
	// setup empty tags table
	pack.FTT_t = FTT_t{
		tidsz:  pack.Header.typesize[PTStidsz],
		tagsz:  pack.Header.typesize[PTStagsz],
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
func GetPackageInfo(r io.ReadSeeker, tidsz, tagsz byte) (ts *Tagset_t, err error) {
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
	if err = hdr.typesize.Checkup(); err != nil {
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

	ts = &Tagset_t{data, tidsz, tagsz}
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
