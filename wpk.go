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
	TID  Uint   // tag ID
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

	ErrOtherSubdir = errors.New("directory refers to other workspace")
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

// Tagger provides acces to nested files by given tagset of this package.
type Tagger interface {
	OpenTagset(*TagsetRaw) (NestedFile, error)
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

const (
	InfoName = "@info" // package info tagset name
	PackName = "@pack" // package content reference
)

// File tags table.
type FTT struct {
	sync.Map      // keys - package filenames in lower case, values - tagset slices.
	tidsz    byte // tag ID size
	tagsz    byte // length of tag size
	tssize   byte // length of tagset size

	datoffset uint64 // files data offset
	datsize   uint64 // files data total size

	mux sync.Mutex // writer mutex
}

// Init performs initialization for given Package structure.
func (ftt *FTT) Init(pts TypeSize) {
	ftt.tidsz = pts[PTStidsz]
	ftt.tagsz = pts[PTStagsz]
	ftt.tssize = pts[PTStssize]
}

// NewTagset creates new empty tagset based on predefined
// TID type size and tag size type.
func (ftt *FTT) NewTagset() *TagsetRaw {
	return &TagsetRaw{nil, ftt.tidsz, ftt.tagsz}
}

// DataSize returns actual package data size from files tags table.
func (ftt *FTT) DataSize() Uint {
	return Uint(ftt.datsize)
}

// Info returns package information tagset if it present.
func (ftt *FTT) Info() (ts *TagsetRaw, ok bool) {
	var val interface{}
	if val, ok = ftt.Load(InfoName); ok {
		ts = val.(*TagsetRaw)
	}
	return
}

// SetInfo returns package information tagset,
// and stores if it not present before.
func (ftt *FTT) SetInfo() *TagsetRaw {
	var emptyinfo = ftt.NewTagset().
		Put(TIDpath, StrTag(InfoName))
	var val, _ = ftt.LoadOrStore(InfoName, emptyinfo)
	if val == nil {
		panic("can not obtain package info")
	}
	return val.(*TagsetRaw)
}

// IsSplitted returns true if package is splitted on tags and data files.
func (ftt *FTT) IsSplitted() bool {
	return ftt.datoffset == 0
}

// CheckTagset tests path & offset & size tags existence
// and checks that size & offset is are in the bounds.
func (ftt *FTT) CheckTagset(ts *TagsetRaw) (fpath string, err error) {
	var ok bool
	var offset, size Uint

	// get file key
	if fpath, ok = ts.TagStr(TIDpath); !ok {
		err = &ErrTag{ErrNoPath, "", TIDpath}
		return
	}
	if _, ok := ftt.Load(fpath); ok { // prevent same file from repeating
		err = &ErrTag{fs.ErrExist, fpath, TIDpath}
		return
	}

	// check system tags
	if offset, ok = ts.TagUint(TIDoffset); !ok && fpath != InfoName {
		err = &ErrTag{ErrNoOffset, fpath, TIDoffset}
		return
	}
	if size, ok = ts.TagUint(TIDsize); !ok && fpath != InfoName {
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

// ReadFrom reads file tags table whole content from the given stream.
func (ftt *FTT) ReadFrom(r io.Reader) (n int64, err error) {
	for {
		var tsl Uint
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

		var ts = &TagsetRaw{data, ftt.tidsz, ftt.tagsz}
		var tsi = ts.Iterator()
		for tsi.Next() {
		}
		if tsi.Failed() {
			err = io.ErrUnexpectedEOF
			return
		}

		var fpath string
		if fpath, err = ftt.CheckTagset(ts); err != nil {
			return
		}

		ftt.Store(Normalize(fpath), ts)
	}
	return
}

// WriteTo writes file tags table whole content to the given stream.
func (ftt *FTT) WriteTo(w io.Writer) (n int64, err error) {
	// write tagset with package info at first
	if ts, ok := ftt.Info(); ok {
		var tsl = Uint(len(ts.Data()))
		if tsl > Uint(1<<(ftt.tssize*8)-1) {
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
	ftt.Range(func(key, value interface{}) bool {
		var fkey, ts = key.(string), value.(*TagsetRaw)
		if fkey == InfoName {
			return true
		}

		var tsl = Uint(len(ts.Data()))
		if tsl > Uint(1<<(ftt.tssize*8)-1) {
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
	if err = hdr.typesize.Checkup(); err != nil {
		return
	}
	// setup empty tags table
	ftt.Map = sync.Map{}
	ftt.tidsz = hdr.typesize[PTStidsz]
	ftt.tagsz = hdr.typesize[PTStagsz]
	ftt.tssize = hdr.typesize[PTStssize]
	ftt.datoffset, ftt.datsize = hdr.datoffset, hdr.datsize
	// go to file tags table start
	if _, err = r.Seek(int64(hdr.fttoffset), io.SeekStart); err != nil {
		return
	}
	// read file tags table
	if hdr.fttsize == 0 {
		return
	}
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
func NewPackage(pts TypeSize) *Package {
	var ftt = &FTT{}
	ftt.Init(pts)
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
func (pkg *Package) BaseTagset(offset, size Uint, fpath string) *TagsetRaw {
	return pkg.NewTagset().
		Put(TIDoffset, UintTag(offset)).
		Put(TIDsize, UintTag(size)).
		Put(TIDpath, StrTag(ToSlash(pkg.FullPath(fpath))))
}

// Tagset returns tagset with given filename key, if it found.
func (pkg *Package) Tagset(fkey string) (ts *TagsetRaw, ok bool) {
	var val interface{}
	if val, ok = pkg.Load(Normalize(pkg.FullPath(fkey))); ok {
		ts = val.(*TagsetRaw)
	}
	return
}

// Enum calls given closure for each tagset in package. Skips package info.
func (pkg *Package) Enum(f func(string, *TagsetRaw) bool) {
	var prefix string
	if pkg.Workspace != "." && pkg.Workspace != "" {
		prefix = Normalize(pkg.Workspace) + "/" // make prefix slash-terminated
	}
	pkg.Range(func(key, value interface{}) bool {
		var fkey = key.(string)
		return fkey == InfoName ||
			!strings.HasPrefix(fkey, prefix) ||
			f(fkey[len(prefix):], value.(*TagsetRaw))
	})
}

// HasTagset check up that tagset with given filename key is present.
func (pkg *Package) HasTagset(fkey string) (ok bool) {
	_, ok = pkg.Load(Normalize(pkg.FullPath(fkey)))
	return
}

// SetTagset puts tagset with given filename key.
func (pkg *Package) SetTagset(fkey string, ts *TagsetRaw) {
	pkg.Store(Normalize(pkg.FullPath(fkey)), ts)
}

// DelTagset deletes tagset with given filename key.
func (pkg *Package) DelTagset(fkey string) {
	pkg.Delete(Normalize(pkg.FullPath(fkey)))
}

// GetDelTagset deletes the tagset for a key, returning the previous tagset if any.
func (pkg *Package) GetDelTagset(fkey string) (ts *TagsetRaw, ok bool) {
	var val interface{}
	if val, ok = pkg.LoadAndDelete(Normalize(pkg.FullPath(fkey))); ok {
		ts = val.(*TagsetRaw)
	}
	return
}

// Sub clones object and gives access to pointed subdirectory.
// fs.SubFS implementation.
func (pkg *Package) Sub(dir string) (sub fs.FS, err error) {
	var prefix string
	if dir != "." && dir != "" {
		prefix = Normalize(dir) + "/" // make prefix slash-terminated
	}
	pkg.Enum(func(fkey string, ts *TagsetRaw) bool {
		if strings.HasPrefix(fkey, prefix) {
			sub = &Package{
				FTT:       pkg.FTT,
				Tagger:    pkg.Tagger,
				Workspace: pkg.FullPath(dir),
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
	if ts, is := pkg.Tagset(fpath); is {
		return ts, nil
	}
	return nil, &fs.PathError{Op: "stat", Path: fpath, Err: fs.ErrNotExist}
}

// Glob returns the names of all files in package matching pattern or nil
// if there is no matching file.
// fs.GlobFS interface implementation.
func (pkg *Package) Glob(pattern string) (res []string, err error) {
	pattern = Normalize(pattern)
	if _, err = path.Match(pattern, ""); err != nil {
		return
	}
	pkg.Enum(func(fkey string, ts *TagsetRaw) bool {
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
	if ts, is := pkg.Tagset(fpath); is {
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

	if ts, is := pkg.Tagset(dir); is {
		return pkg.Tagger.OpenTagset(ts)
	}
	return pkg.OpenDir(fullname)
}

// GetPackageInfo returns tagset with package information.
// It's a quick function to get info from the file.
func GetPackageInfo(r io.ReadSeeker) (ts *TagsetRaw, err error) {
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
	var tsl Uint
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

	ts = &TagsetRaw{data, hdr.typesize[PTStidsz], hdr.typesize[PTStagsz]}
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
