package mmap

import (
	"bytes"
	"io/fs"
	"os"
	"path"
	"strings"

	mm "github.com/schwarzlichtbezirk/mmap-go"
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
	tags   *wpk.Tagset_t // has fs.FileInfo interface
	region []byte
	mm.MMap
}

// NewMappedFile maps nested to package file based on given tags slice.
func NewMappedFile(pack *PackDir, ts *wpk.Tagset_t) (f *MappedFile, err error) {
	// calculate paged size/offset
	var offset, _ = ts.FOffset()
	var size, _ = ts.FSize()
	var pgoff = offset % pagesize
	var offsetx = offset - pgoff
	var sizex = size + wpk.FSize_t(pgoff)
	// create mapped memory block
	var mmap mm.MMap
	if mmap, err = mm.MapRegion(pack.filewpk, int64(offsetx), int64(sizex), mm.RDONLY, 0); err != nil {
		return
	}
	f = &MappedFile{
		tags:       ts,
		FileReader: bytes.NewReader(mmap[pgoff : pgoff+wpk.FOffset_t(size)]),
		region:     mmap[pgoff : pgoff+wpk.FOffset_t(size)],
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

// PackDir is wrapper for package to get access to nested files as to memory mapped blocks.
// Gives access to pointed directory in package.
// fs.FS interface implementation.
type PackDir struct {
	*wpk.Package
	workspace string   // workspace directory in package
	filewpk   *os.File // open package file descriptor
}

// OpenTagset creates file object to give access to nested into package file by given tagset.
func (pack *PackDir) OpenTagset(ts *wpk.Tagset_t) (wpk.NestedFile, error) {
	return NewMappedFile(pack, ts)
}

// OpenImage opens WPK-file package by given file name.
func OpenImage(fname string) (pack *PackDir, err error) {
	pack = &PackDir{Package: &wpk.Package{}}
	pack.workspace = "."

	if pack.filewpk, err = os.Open(fname); err != nil {
		return
	}
	if err = pack.Read(pack.filewpk); err != nil {
		return
	}
	return
}

// Close file handle. This function must be called only for root object,
// not subdirectories.
// io.Closer implementation.
func (pack *PackDir) Close() error {
	return pack.filewpk.Close()
}

// Sub clones object and gives access to pointed subdirectory.
// Copies file handle, so it must be closed only once for root object.
// fs.SubFS implementation.
func (pack *PackDir) Sub(dir string) (df fs.FS, err error) {
	if !fs.ValidPath(dir) {
		err = &fs.PathError{Op: "sub", Path: dir, Err: fs.ErrInvalid}
		return
	}
	var workspace = path.Join(pack.workspace, dir)
	var prefixdir string
	if workspace != "." {
		prefixdir = workspace + "/" // make prefix slash-terminated
	}
	pack.Enum(func(fkey string, ts *wpk.Tagset_t) bool {
		if strings.HasPrefix(fkey, prefixdir) {
			df, err = &PackDir{
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
func (pack *PackDir) Stat(name string) (fs.FileInfo, error) {
	if !fs.ValidPath(name) {
		return nil, &fs.PathError{Op: "stat", Path: name, Err: fs.ErrInvalid}
	}
	var ts *wpk.Tagset_t
	var is bool
	if ts, is = pack.Tagset(wpk.Normalize(path.Join(pack.workspace, name))); !is {
		return nil, &fs.PathError{Op: "stat", Path: name, Err: fs.ErrNotExist}
	}
	return ts, nil
}

// ReadFile returns slice with nested into package file content.
// Makes content copy to prevent ambiguous access to closed mapped memory block.
// fs.ReadFileFS implementation.
func (pack *PackDir) ReadFile(name string) ([]byte, error) {
	if !fs.ValidPath(name) {
		return nil, &fs.PathError{Op: "readfile", Path: name, Err: fs.ErrInvalid}
	}
	var ts *wpk.Tagset_t
	var is bool
	if ts, is = pack.Tagset(wpk.Normalize(path.Join(pack.workspace, name))); !is {
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
func (pack *PackDir) ReadDir(dir string) ([]fs.DirEntry, error) {
	return wpk.ReadDir(pack, path.Join(pack.workspace, dir), -1)
}

// Open implements access to nested into package file or directory by keyname.
// fs.FS implementation.
func (pack *PackDir) Open(dir string) (fs.File, error) {
	if dir == "wpk" && pack.workspace == "." {
		return pack.filewpk, nil
	}

	var fullname = path.Join(pack.workspace, dir)
	if ts, is := pack.Tagset(wpk.Normalize(fullname)); is {
		return NewMappedFile(pack, ts)
	}
	return wpk.OpenDir(pack, fullname)
}

// The End.
