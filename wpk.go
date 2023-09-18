package wpk

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"strings"
	"sync"
)

const (
	SignSize   = 24 // signature field size.
	HeaderSize = 64 // package header size in bytes.

	SignReady = "Whirlwind 3.4 Package   " // package is ready for use
	SignBuild = "Whirlwind 3.4 Prebuild  " // package is in building progress
)

// List of predefined tags IDs.
const (
	TIDnone = 0

	TIDoffset = 1  // required, uint
	TIDsize   = 2  // required, uint
	TIDpath   = 3  // required, unique, string
	TIDfid    = 4  // unique, uint
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

	TIDtmbjpeg  = 100 // []byte, thumbnail image (icon) in JPEG format
	TIDtmbwebp  = 101 // []byte, thumbnail image (icon) in WebP format
	TIDlabel    = 110 // string
	TIDlink     = 111 // string
	TIDkeywords = 112 // string
	TIDcategory = 113 // string
	TIDversion  = 114 // string
	TIDauthor   = 115 // string
	TIDcomment  = 116 // string
)

type TID = uint16

// ErrTag is error on some field of tags set.
type ErrTag struct {
	What error  // error message
	Key  string // normalized file name
	TID  TID    // tag ID
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

	ErrRangeTSSize = errors.New("tagset size value is exceeds out of the type dimension")

	ErrNoTag    = errors.New("tag with given ID not found")
	ErrNoPath   = errors.New("file name is absent")
	ErrNoOffset = errors.New("file offset is absent")
	ErrNoSize   = errors.New("file size is absent")
	ErrOutOff   = errors.New("file offset is out of bounds")
	ErrOutSize  = errors.New("file size is out of bounds")

	ErrOtherSubdir = errors.New("directory refers to other workspace")
)

// FileReader is interface for nested package files access.
type FileReader interface {
	io.Reader
	io.ReaderAt
	io.Seeker
	Size() int64
}

// PkgFile is interface for access to nested into package files.
type PkgFile interface {
	fs.File
	FileReader
}

// Tagger provides acces to nested files by given tagset of this package.
type Tagger interface {
	OpenTagset(TagsetRaw) (PkgFile, error)
	io.Closer
}

// CompleteFS includes all FS interfaces.
type CompleteFS interface {
	fs.SubFS
	fs.StatFS
	fs.GlobFS
	fs.ReadDirFS
	fs.ReadFileFS
}

// Package types sizes.
const (
	PTStidsz  = 2 // "tag ID" type size.
	PTStagsz  = 2 // "tag size" type size.
	PTStssize = 2 // "tagset size" type size.

	tsmaxlen = 1<<(PTStssize*8) - 1 // tagset maximum length.
)

// Header - package header.
type Header struct {
	signature [SignSize]byte
	fttcount  uint64 // count of entries in file tags table
	fttoffset uint64 // file tags table offset
	fttsize   uint64 // file tags table size
	datoffset uint64 // files data offset
	datsize   uint64 // files data total size
}

// IsReady determines that package is ready for read the data.
func (hdr *Header) IsReady() error {
	// can not read file tags table for opened on write single-file package.
	if B2S(hdr.signature[:]) == SignBuild {
		if hdr.datoffset != 0 {
			return ErrSignPre
		}
		return nil
	}
	// can not read file tags table on any incorrect signature
	if B2S(hdr.signature[:]) != SignReady {
		return ErrSignBad
	}
	return nil
}

// ReadFrom reads header from stream as binary data of constant length in little endian order.
func (hdr *Header) ReadFrom(r io.Reader) (n int64, err error) {
	if err = binary.Read(r, binary.LittleEndian, hdr.signature[:]); err != nil {
		return
	}
	n += SignSize
	if err = binary.Read(r, binary.LittleEndian, &hdr.fttcount); err != nil {
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
	if err = binary.Write(w, binary.LittleEndian, &hdr.fttcount); err != nil {
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

const (
	InfoName = "@info" // package info tagset name
	PackName = "@pack" // package content reference
)

// File tags table.
type FTT struct {
	rwm RWMap[string, TagsetRaw] // keys - package filenames (case sensitive), values - tagset slices.

	datoffset uint64 // files data offset
	datsize   uint64 // files data total size

	mux sync.Mutex // writer mutex
}

// Init performs initialization for given Package structure.
func (ftt *FTT) Init(c int) {
	ftt.rwm.Init(c)
}

// DataSize returns actual package data size from files tags table.
func (ftt *FTT) DataSize() Uint {
	return Uint(ftt.datsize)
}

// TagsetNum returns number of entries at files tags table.
func (ftt *FTT) TagsetNum() int {
	return ftt.rwm.Len()
}

// GetInfo returns package information tagset if it present.
func (ftt *FTT) GetInfo() (TagsetRaw, bool) {
	return ftt.rwm.Get(InfoName)
}

// SetInfo puts given tagset as package information tagset with "@info" path tag.
func (ftt *FTT) SetInfo(ts TagsetRaw) {
	ts, _ = ts.Set(TIDpath, StrTag(InfoName))
	ftt.rwm.Set(InfoName, ts)
}

// IsSplitted returns true if package is splitted on tags and data files.
func (ftt *FTT) IsSplitted() bool {
	return ftt.datoffset == 0
}

// CheckTagset tests path & offset & size tags existence
// and checks that size & offset is are in the bounds.
func (ftt *FTT) CheckTagset(ts TagsetRaw) (fpath string, err error) {
	var offset, size Uint
	var ispath, isoffset, issize bool

	// find expected tags
	var tsi = ts.Iterator()
	for tsi.Next() {
		switch tsi.tid {
		case TIDoffset:
			offset, isoffset = TagRaw(tsi.TagsetRaw[tsi.tag:tsi.pos]).TagUint()
		case TIDsize:
			size, issize = TagRaw(tsi.TagsetRaw[tsi.tag:tsi.pos]).TagUint()
		case TIDpath:
			fpath, ispath = TagRaw(tsi.TagsetRaw[tsi.tag:tsi.pos]).TagStr()
		}
	}
	if tsi.Failed() {
		err = io.ErrUnexpectedEOF
		return
	}

	// check tags existence
	if !ispath {
		err = &ErrTag{ErrNoPath, "", TIDpath}
		return
	}
	if ftt.rwm.Has(fpath) { // prevent same file from repeating
		err = &ErrTag{fs.ErrExist, fpath, TIDpath}
		return
	}
	if !isoffset && fpath != InfoName {
		err = &ErrTag{ErrNoOffset, fpath, TIDoffset}
		return
	}
	if !issize && fpath != InfoName {
		err = &ErrTag{ErrNoSize, fpath, TIDsize}
		return
	}

	// check up offset and size
	if uint64(offset) < ftt.datoffset || uint64(offset) > ftt.datoffset+ftt.datsize {
		err = &ErrTag{ErrOutOff, fpath, TIDoffset}
		return
	}
	if uint64(offset+size) > ftt.datoffset+ftt.datsize {
		err = &ErrTag{ErrOutSize, fpath, TIDsize}
		return
	}

	return
}

// Parse makes table from given byte slice.
func (ftt *FTT) Parse(buf []byte) (n int64, err error) {
	for {
		var tsl uint16
		tsl = GetU16(buf[n : n+PTStssize])
		n += PTStssize

		if tsl == 0 {
			break // end marker was reached
		}

		var ts = TagsetRaw(buf[n : n+int64(tsl)])
		n += int64(tsl)

		var fpath string
		if fpath, err = ftt.CheckTagset(ts); err != nil {
			return
		}

		ftt.rwm.Set(ToSlash(fpath), ts)
	}
	return
}

// ReadFrom reads file tags table whole content from the given stream.
func (ftt *FTT) ReadFrom(r io.Reader) (n int64, err error) {
	for {
		var tsl uint16
		if tsl, err = ReadU16(r); err != nil {
			return
		}
		n += PTStssize

		if tsl == 0 {
			break // end marker was reached
		}

		var ts = make(TagsetRaw, tsl)
		if _, err = r.Read(ts); err != nil {
			return
		}
		n += int64(tsl)

		var fpath string
		if fpath, err = ftt.CheckTagset(ts); err != nil {
			return
		}

		ftt.rwm.Set(ToSlash(fpath), ts)
	}
	return
}

// WriteTo writes file tags table whole content to the given stream.
func (ftt *FTT) WriteTo(w io.Writer) (n int64, err error) {
	// write tagset with package info at first
	if ts, ok := ftt.GetInfo(); ok {
		var tsl = len(ts)
		if tsl > tsmaxlen {
			err = ErrRangeTSSize
			return
		}

		// write tagset length
		if err = WriteU16(w, uint16(tsl)); err != nil {
			return
		}
		n += PTStssize

		// write tagset content
		if _, err = w.Write(ts); err != nil {
			return
		}
		n += int64(tsl)
	}

	// write files tags table
	ftt.rwm.Range(func(fkey string, ts TagsetRaw) bool {
		if fkey == InfoName {
			return true
		}

		var tsl = len(ts)
		if tsl > tsmaxlen {
			err = ErrRangeTSSize
			return false
		}

		// write tagset length
		if err = WriteU16(w, uint16(tsl)); err != nil {
			return false
		}
		n += PTStssize

		// write tagset content
		if _, err = w.Write(ts); err != nil {
			return false
		}
		n += int64(tsl)
		return true
	})
	if err != nil {
		return
	}
	// write tags table end marker
	if err = WriteU16(w, 0); err != nil {
		return
	}
	n += PTStssize
	return
}

// ReadFTT opens package for reading. At first it checkups file signature,
// then reads records table, and reads file tagset table. Tags set for each
// file must contain at least file offset, file size, file ID and file name.
func (ftt *FTT) ReadFTT(r io.ReadSeeker) (err error) {
	// go to file start
	if _, err = r.Seek(0, io.SeekStart); err != nil {
		return
	}
	// read header
	var hdr Header
	if _, err = hdr.ReadFrom(r); err != nil {
		return
	}
	if err = hdr.IsReady(); err != nil {
		return
	}
	if hdr.fttcount == 0 || hdr.fttsize == 0 {
		return
	}
	// setup empty tags table with reserved map size
	ftt.Init(int(hdr.fttcount))
	ftt.datoffset, ftt.datsize = hdr.datoffset, hdr.datsize
	// go to file tags table start
	if _, err = r.Seek(int64(hdr.fttoffset), io.SeekStart); err != nil {
		return
	}
	// read file tags table
	var fttsize int64
	if fttsize, err = ftt.ReadFrom(r); err != nil {
		return
	}
	if fttsize != int64(hdr.fttsize) {
		err = ErrSignFTT
		return
	}
	return
}

// Package structure contains file tags table, tagger object
// to get access to nested files, and subdirectory workspace.
type Package struct {
	*FTT
	Tagger
	Workspace string
}

// NewPackage returns pointer to new initialized Package filesystem structure.
// Tagger should be set later if access to nested files is needed.
func NewPackage() *Package {
	var ftt = &FTT{}
	ftt.Init(0)
	return &Package{
		FTT:       ftt,
		Workspace: ".",
	}
}

// OpenPackage creates Package objects and reads package file tags table
// from the file with given name.
// Tagger should be set later if access to nested files is needed.
func OpenPackage(fpath string) (pkg *Package, err error) {
	var r io.ReadSeekCloser
	if r, err = os.Open(fpath); err != nil {
		return
	}
	defer r.Close()

	pkg = &Package{
		FTT:       &FTT{},
		Workspace: ".",
	}
	err = pkg.ReadFTT(r)
	return
}

// FullPath returns concatenation of workspace and relative path.
func (pkg *Package) FullPath(fpath string) string {
	return path.Join(pkg.Workspace, fpath)
}

// TrimPath returns trimmed path without workspace prefix.
func (pkg *Package) TrimPath(fpath string) string {
	if pkg.Workspace == "." || pkg.Workspace == "" {
		return fpath
	}
	if !strings.HasPrefix(fpath, pkg.Workspace) {
		return ""
	}
	fpath = fpath[len(pkg.Workspace):]
	if fpath == "" || fpath == "/" {
		return "."
	}
	if fpath[0] != '/' {
		return ""
	}
	return fpath[1:]
}

// BaseTagset returns new tagset based on predefined TID type size and tag size type,
// and puts file offset and file size into tagset with predefined sizes.
func (pkg *Package) BaseTagset(offset, size Uint, fpath string) TagsetRaw {
	return TagsetRaw{}.
		Put(TIDoffset, UintTag(offset)).
		Put(TIDsize, UintTag(size)).
		Put(TIDpath, StrTag(pkg.FullPath(ToSlash(fpath))))
}

// HasTagset check up that tagset with given filename key is present.
func (pkg *Package) HasTagset(fkey string) bool {
	return pkg.rwm.Has(pkg.FullPath(ToSlash(fkey)))
}

// GetTagset returns tagset with given filename key, if it found.
func (pkg *Package) GetTagset(fkey string) (TagsetRaw, bool) {
	return pkg.rwm.Get(pkg.FullPath(ToSlash(fkey)))
}

// SetTagset puts tagset with given filename key.
func (pkg *Package) SetTagset(fkey string, ts TagsetRaw) {
	pkg.rwm.Set(pkg.FullPath(ToSlash(fkey)), ts)
}

// SetupTagset puts tagset with filename key stored at tagset.
func (pkg *Package) SetupTagset(ts TagsetRaw) {
	pkg.rwm.Set(ts.Path(), ts)
}

// DelTagset deletes tagset with given filename key.
func (pkg *Package) DelTagset(fkey string) {
	pkg.rwm.Delete(pkg.FullPath(ToSlash(fkey)))
}

// GetDelTagset deletes the tagset for a key, returning the previous tagset if any.
func (pkg *Package) GetDelTagset(fkey string) (TagsetRaw, bool) {
	return pkg.rwm.GetAndDelete(pkg.FullPath(ToSlash(fkey)))
}

// Enum calls given closure for each tagset in package. Skips package info.
func (pkg *Package) Enum(f func(string, TagsetRaw) bool) {
	var prefix string
	if pkg.Workspace != "." && pkg.Workspace != "" {
		prefix = pkg.Workspace + "/" // make prefix path slash-terminated
	}
	pkg.rwm.Range(func(fkey string, ts TagsetRaw) bool {
		return fkey == InfoName ||
			!strings.HasPrefix(fkey, prefix) ||
			f(fkey[len(prefix):], ts)
	})
}

// Sub clones object and gives access to pointed subdirectory.
// fs.SubFS implementation.
func (pkg *Package) Sub(dir string) (sub fs.FS, err error) {
	var prefix string
	if dir != "." && dir != "" {
		prefix = ToSlash(dir) + "/" // make prefix slash-terminated
	}
	pkg.Enum(func(fkey string, ts TagsetRaw) bool {
		if strings.HasPrefix(fkey, prefix) {
			sub = &Package{
				FTT:       pkg.FTT,
				Tagger:    pkg.Tagger,
				Workspace: pkg.FullPath(ToSlash(dir)),
			}
			return false
		}
		return true
	})
	if sub == nil { // on case if not found
		err = &fs.PathError{Op: "sub", Path: dir, Err: fs.ErrNotExist}
	}
	return
}

// Stat returns a fs.FileInfo describing the file.
// fs.StatFS interface implementation.
func (pkg *Package) Stat(fpath string) (fs.FileInfo, error) {
	if ts, is := pkg.GetTagset(fpath); is {
		return ts, nil
	}
	return nil, &fs.PathError{Op: "stat", Path: fpath, Err: fs.ErrNotExist}
}

// Glob returns the names of all files in package matching pattern or nil
// if there is no matching file.
// fs.GlobFS interface implementation.
func (pkg *Package) Glob(pattern string) (res []string, err error) {
	pattern = ToSlash(pattern)
	if _, err = path.Match(pattern, ""); err != nil {
		return
	}
	pkg.Enum(func(fkey string, ts TagsetRaw) bool {
		if matched, _ := path.Match(pattern, fkey); matched {
			res = append(res, fkey)
		}
		return true
	})
	return
}

// ReadDir reads the named directory
// and returns a list of directory entries sorted by filename.
// fs.ReadDirFS interface implementation.
func (pkg *Package) ReadDir(dir string) ([]fs.DirEntry, error) {
	var fulldir = pkg.FullPath(dir)
	return pkg.FTT.ReadDirN(fulldir, -1)
}

// ReadFile returns slice with nested into package file content.
// Makes content copy to prevent ambiguous access to closed mapped memory block.
// fs.ReadFileFS implementation.
func (pkg *Package) ReadFile(fpath string) ([]byte, error) {
	if ts, is := pkg.GetTagset(fpath); is {
		var f, err = pkg.Tagger.OpenTagset(ts)
		if err != nil {
			return nil, err
		}
		defer f.Close()

		var size = ts.Size()
		var buf = make([]byte, size)
		_, err = f.Read(buf)
		return buf, err
	}
	return nil, &fs.PathError{Op: "readfile", Path: fpath, Err: fs.ErrNotExist}
}

// Open implements access to nested into package file or directory by filename.
// fs.FS implementation.
func (pkg *Package) Open(dir string) (fs.File, error) {
	var fullname = pkg.FullPath(dir)
	if fullname == PackName {
		var ts = pkg.BaseTagset(0, Uint(pkg.datoffset+pkg.datsize), "wpk")
		return pkg.Tagger.OpenTagset(ts)
	}

	if ts, is := pkg.GetTagset(dir); is {
		return pkg.Tagger.OpenTagset(ts)
	}
	return pkg.OpenDir(fullname)
}

// GetPackageInfo returns tagset with package information.
// It's a quick function to get info from the file.
func GetPackageInfo(r io.ReadSeeker) (ts TagsetRaw, err error) {
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
	var tsl uint16
	if tsl, err = ReadU16(r); err != nil {
		return
	}
	if tsl == 0 {
		return // end marker was reached
	}

	ts = make(TagsetRaw, tsl)
	if _, err = r.Read(ts); err != nil {
		return
	}

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
	if fpath, ok = ts.TagStr(TIDpath); !ok {
		err = ErrNoPath
		return
	}
	if fpath != InfoName {
		ts = nil // info is not found, returns (nil, nil)
		return
	}
	return
}

// The End.
