package mmap

import (
	"bytes"
	"io"
	"io/fs"
	"os"
	"path"
	"strings"

	mm "github.com/edsrzf/mmap-go"
	"github.com/schwarzlichtbezirk/wpk"
)

// System pages granulation for memory mapping system calls.
// The page size on most Unixes is 4KB, but on Windows it's 64KB.
// os.Getpagesize() returns incorrect value on Windows.
const pagesize = 64 * 1024

// MappedFile structure gives access to nested into package file by memory mapping.
// wpk.NestedFile interface implementation.
type MappedFile[TID_t wpk.TID_i, TSize_t wpk.TSize_i] struct {
	wpk.FileReader
	tags   *wpk.Tagset_t[TID_t, TSize_t] // has fs.FileInfo interface
	region []byte
	mm.MMap
}

// NewMappedFile maps nested to package file based on given tags slice.
func NewMappedFile[TID_t wpk.TID_i, TSize_t wpk.TSize_i, TSSize_t wpk.TSSize_i](pack *Package[TID_t, TSize_t, TSSize_t], ts *wpk.Tagset_t[TID_t, TSize_t]) (f *MappedFile[TID_t, TSize_t], err error) {
	// calculate paged size/offset
	var offset, _ = ts.FOffset()
	var size, _ = ts.FSize()
	var pgoff = offset % pagesize
	var offsetx = offset - pgoff
	var sizex = size + wpk.FSize_t(pgoff)
	// create mapped memory block
	var mmap mm.MMap
	if mmap, err = mm.MapRegion(pack.filewpk, int(sizex), mm.RDONLY, 0, int64(offsetx)); err != nil {
		return
	}
	f = &MappedFile[TID_t, TSize_t]{
		tags:       ts,
		FileReader: bytes.NewReader(mmap[pgoff : pgoff+wpk.FOffset_t(size)]),
		region:     mmap[pgoff : pgoff+wpk.FOffset_t(size)],
		MMap:       mmap,
	}
	return
}

// Stat is for fs.File interface compatibility.
func (f *MappedFile[TID_t, TSize_t]) Stat() (fs.FileInfo, error) {
	return f.tags, nil
}

// Close unmaps memory and closes mapped memory handle.
func (f *MappedFile[TID_t, TSize_t]) Close() error {
	return f.Unmap()
}

// Package is wrapper for package to get access to nested files as to memory mapped blocks.
// Gives access to pointed directory in package.
// fs.FS interface implementation.
type Package[TID_t wpk.TID_i, TSize_t wpk.TSize_i, TSSize_t wpk.TSSize_i] struct {
	*wpk.Package[TID_t, TSize_t, TSSize_t]
	workspace string   // workspace directory in package
	filewpk   *os.File // open package file descriptor
}

// OpenTagset creates file object to give access to nested into package file by given tagset.
func (pack *Package[TID_t, TSize_t, TSSize_t]) OpenTagset(ts *wpk.Tagset_t[TID_t, TSize_t]) (wpk.NestedFile, error) {
	return NewMappedFile(pack, ts)
}

// OpenPackage opens WPK-file package by given file name.
func OpenPackage[TID_t wpk.TID_i, TSize_t wpk.TSize_i, TSSize_t wpk.TSSize_i](fname string) (pack *Package[TID_t, TSize_t, TSSize_t], err error) {
	pack = &Package[TID_t, TSize_t, TSSize_t]{
		Package:   &wpk.Package[TID_t, TSize_t, TSSize_t]{},
		workspace: ".",
	}

	var r io.ReadSeekCloser
	if r, err = os.Open(fname); err != nil {
		return
	}
	defer r.Close()

	if err = pack.OpenFTT(r); err != nil {
		return
	}

	if pack.IsSplitted() {
		if pack.filewpk, err = os.Open(wpk.MakeDataPath(fname)); err != nil {
			return
		}
	} else {
		if pack.filewpk, err = os.Open(fname); err != nil {
			return
		}
	}
	return
}

// Close file handle. This function must be called only for root object,
// not subdirectories.
// io.Closer implementation.
func (pack *Package[TID_t, TSize_t, TSSize_t]) Close() error {
	return pack.filewpk.Close()
}

// Sub clones object and gives access to pointed subdirectory.
// Copies file handle, so it must be closed only once for root object.
// fs.SubFS implementation.
func (pack *Package[TID_t, TSize_t, TSSize_t]) Sub(dir string) (df fs.FS, err error) {
	if !fs.ValidPath(dir) {
		err = &fs.PathError{Op: "sub", Path: dir, Err: fs.ErrInvalid}
		return
	}
	var workspace = path.Join(pack.workspace, dir)
	var prefixdir string
	if workspace != "." {
		prefixdir = workspace + "/" // make prefix slash-terminated
	}
	pack.Enum(func(fkey string, ts *wpk.Tagset_t[TID_t, TSize_t]) bool {
		if strings.HasPrefix(fkey, prefixdir) {
			df, err = &Package[TID_t, TSize_t, TSSize_t]{
				pack.Package,
				workspace,
				pack.filewpk,
			}, nil
			return false
		}
		return true
	})
	if df == nil { // on case if not found
		err = &fs.PathError{Op: "sub", Path: dir, Err: fs.ErrNotExist}
	}
	return
}

// Stat returns a fs.FileInfo describing the file.
// fs.StatFS implementation.
func (pack *Package[TID_t, TSize_t, TSSize_t]) Stat(name string) (fs.FileInfo, error) {
	if !fs.ValidPath(name) {
		return nil, &fs.PathError{Op: "stat", Path: name, Err: fs.ErrInvalid}
	}
	var ts *wpk.Tagset_t[TID_t, TSize_t]
	var is bool
	if ts, is = pack.Tagset(path.Join(pack.workspace, name)); !is {
		return nil, &fs.PathError{Op: "stat", Path: name, Err: fs.ErrNotExist}
	}
	return ts, nil
}

// ReadFile returns slice with nested into package file content.
// Makes content copy to prevent ambiguous access to closed mapped memory block.
// fs.ReadFileFS implementation.
func (pack *Package[TID_t, TSize_t, TSSize_t]) ReadFile(name string) ([]byte, error) {
	if !fs.ValidPath(name) {
		return nil, &fs.PathError{Op: "readfile", Path: name, Err: fs.ErrInvalid}
	}
	var ts *wpk.Tagset_t[TID_t, TSize_t]
	var is bool
	if ts, is = pack.Tagset(path.Join(pack.workspace, name)); !is {
		return nil, &fs.PathError{Op: "readfile", Path: name, Err: fs.ErrNotExist}
	}
	var f, err = NewMappedFile(pack, ts)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var size, _ = ts.FSize()
	var buf = make([]byte, size)
	_, err = f.Read(buf)
	return buf, err
}

// ReadDir reads the named directory
// and returns a list of directory entries sorted by filename.
func (pack *Package[TID_t, TSize_t, TSSize_t]) ReadDir(dir string) ([]fs.DirEntry, error) {
	return wpk.ReadDir[TID_t, TSize_t](pack, path.Join(pack.workspace, dir), -1)
}

// Open implements access to nested into package file or directory by keyname.
// fs.FS implementation.
func (pack *Package[TID_t, TSize_t, TSSize_t]) Open(dir string) (fs.File, error) {
	if dir == "wpk" && pack.workspace == "." {
		return pack.filewpk, nil
	}

	var fullname = path.Join(pack.workspace, dir)
	if ts, is := pack.Tagset(fullname); is {
		return NewMappedFile(pack, ts)
	}
	return wpk.OpenDir[TID_t, TSize_t](pack, fullname)
}

// The End.
