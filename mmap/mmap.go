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
type MappedFile struct {
	wpk.FileReader
	tags   *wpk.TagsetRaw // has fs.FileInfo interface
	region []byte
	mm.MMap
}

// NewMappedFile maps nested to package file based on given tags slice.
func NewMappedFile(pack *Package, ts *wpk.TagsetRaw) (f *MappedFile, err error) {
	// calculate paged size/offset
	var offset, size = ts.Pos()
	var pgoff = offset % pagesize
	var offsetx = offset - pgoff
	var sizex = size + pgoff
	// create mapped memory block
	var mmap mm.MMap
	if mmap, err = mm.MapRegion(pack.filewpk, int(sizex), mm.RDONLY, 0, int64(offsetx)); err != nil {
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

// Package is wrapper for package to get access to nested files as to memory mapped blocks.
// Gives access to pointed directory in package.
// fs.FS interface implementation.
type Package struct {
	*wpk.Package
	workspace string   // workspace directory in package
	filewpk   *os.File // open package file descriptor
}

// OpenTagset creates file object to give access to nested into package file by given tagset.
func (pack *Package) OpenTagset(ts *wpk.TagsetRaw) (wpk.NestedFile, error) {
	return NewMappedFile(pack, ts)
}

// OpenPackage opens WPK-file package by given file name.
func OpenPackage(fpath string) (pack *Package, err error) {
	pack = &Package{
		Package:   &wpk.Package{},
		workspace: ".",
	}

	var r io.ReadSeekCloser
	if r, err = os.Open(fpath); err != nil {
		return
	}
	defer r.Close()

	if err = pack.OpenFTT(r); err != nil {
		return
	}

	var dpath string
	if pack.IsSplitted() {
		dpath = wpk.MakeDataPath(fpath)
	} else {
		dpath = fpath
	}
	if pack.filewpk, err = os.Open(dpath); err != nil {
		return
	}
	return
}

// Close file handle. This function must be called only for root object,
// not subdirectories.
// io.Closer implementation.
func (pack *Package) Close() error {
	return pack.filewpk.Close()
}

// Sub clones object and gives access to pointed subdirectory.
// Copies file handle, so it must be closed only once for root object.
// fs.SubFS implementation.
func (pack *Package) Sub(dir string) (sub fs.FS, err error) {
	var fulldir = path.Join(pack.workspace, dir)
	var prefix string
	if fulldir != "." {
		prefix = wpk.Normalize(fulldir) + "/" // make prefix slash-terminated
	}
	pack.Enum(func(fkey string, ts *wpk.TagsetRaw) bool {
		if strings.HasPrefix(fkey, prefix) {
			sub = &Package{
				pack.Package,
				fulldir,
				pack.filewpk,
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
// fs.StatFS implementation.
func (pack *Package) Stat(name string) (fs.FileInfo, error) {
	var fullname = path.Join(pack.workspace, name)
	if ts, is := pack.Tagset(fullname); is {
		return ts, nil
	}
	return nil, &fs.PathError{Op: "stat", Path: name, Err: fs.ErrNotExist}
}

// ReadFile returns slice with nested into package file content.
// Makes content copy to prevent ambiguous access to closed mapped memory block.
// fs.ReadFileFS implementation.
func (pack *Package) ReadFile(name string) ([]byte, error) {
	var fullname = path.Join(pack.workspace, name)
	var ts *wpk.TagsetRaw
	var is bool
	if ts, is = pack.Tagset(fullname); !is {
		return nil, &fs.PathError{Op: "readfile", Path: name, Err: fs.ErrNotExist}
	}
	var f, err = NewMappedFile(pack, ts)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var size = ts.Size()
	var buf = make([]byte, size)
	_, err = f.Read(buf)
	return buf, err
}

// ReadDir reads the named directory
// and returns a list of directory entries sorted by filename.
func (pack *Package) ReadDir(dir string) ([]fs.DirEntry, error) {
	var fullname = path.Join(pack.workspace, dir)
	return pack.FTT.ReadDirN(fullname, -1)
}

// Open implements access to nested into package file or directory by keyname.
// fs.FS implementation.
func (pack *Package) Open(dir string) (fs.File, error) {
	var fullname = path.Join(pack.workspace, dir)
	if fullname == wpk.PackName {
		return pack.filewpk, nil
	}

	if ts, is := pack.Tagset(fullname); is {
		return NewMappedFile(pack, ts)
	}
	return pack.FTT.OpenDir(fullname)
}

// The End.
