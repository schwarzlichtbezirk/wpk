package mmap

import (
	"bytes"
	"io/fs"
	"os"

	mm "github.com/edsrzf/mmap-go"
	"github.com/schwarzlichtbezirk/wpk"
)

// System pages granulation for memory mapping system calls.
// The page size on most Unixes is 4KB, but on Windows it's 64KB.
// os.Getpagesize() returns incorrect value on Windows.
const pagesize = 64 * 1024

// MappedFile structure gives access to nested into package file by memory mapping.
// wpk.NestedFile interface implementation.
type MappedFile struct {
	wpk.FileReader
	tags   *wpk.TagsetRaw // has fs.FileInfo interface
	region []byte
	mm.MMap
}

// NewMappedFile maps nested to package file based on given tags slice.
func NewMappedFile(tgr *Tagger, ts *wpk.TagsetRaw) (f *MappedFile, err error) {
	// calculate paged size/offset
	var offset, size = ts.Pos()
	var pgoff = offset % pagesize
	var offsetx = offset - pgoff
	var sizex = size + pgoff
	// create mapped memory block
	var mmap mm.MMap
	if mmap, err = mm.MapRegion(tgr.fwpk, int(sizex), mm.RDONLY, 0, int64(offsetx)); err != nil {
		return
	}
	f = &MappedFile{
		FileReader: bytes.NewReader(mmap[pgoff : pgoff+size]),
		tags:       ts,
		region:     mmap[pgoff : pgoff+size],
		MMap:       mmap,
	}
	return
}

// Stat is for fs.File interface compatibility.
func (f *MappedFile) Stat() (fs.FileInfo, error) {
	return f.tags, nil
}

// Close unmaps memory and closes mapped memory handle.
func (f *MappedFile) Close() error {
	return f.Unmap()
}

// Tagger is object to get access to package nested files
// by memory mapping of wpk-file.
type Tagger struct {
	fwpk *os.File // open package file descriptor
}

// MakeTagger creates Tagger object to get access to package nested files.
func MakeTagger(pack *wpk.Package, fpath string) (wpk.Tagger, error) {
	var dpath string
	if pack.IsSplitted() {
		dpath = wpk.MakeDataPath(fpath)
	} else {
		dpath = fpath
	}

	var err error
	var tgr Tagger
	if tgr.fwpk, err = os.Open(dpath); err != nil {
		return nil, err
	}
	return &tgr, nil
}

// OpenTagset creates file object to give access to nested into package file by given tagset.
func (tgr *Tagger) OpenTagset(ts *wpk.TagsetRaw) (wpk.NestedFile, error) {
	return NewMappedFile(tgr, ts)
}

// Close file handle. This function must be called only for root object,
// not subdirectories. It has no effect otherwise.
// io.Closer implementation.
func (tgr *Tagger) Close() error {
	return tgr.fwpk.Close()
}

// The End.
