package bulk

import (
	"bytes"
	"io"
	"io/fs"
	"os"
	"path"
	"strings"

	"github.com/schwarzlichtbezirk/wpk"
)

// SliceFile structure gives access to nested into package file.
// wpk.NestedFile interface implementation.
type SliceFile struct {
	wpk.FileReader
	tags *wpk.TagsetRaw // has fs.FileInfo interface
}

// NewSliceFile creates SliceFile file structure based on given tags slice.
func NewSliceFile(pack *Package, ts *wpk.TagsetRaw) (f *SliceFile, err error) {
	var offset, size = ts.Pos()
	f = &SliceFile{
		FileReader: bytes.NewReader(pack.bulk[offset : offset+size]),
		tags:       ts,
	}
	return
}

// Stat is for fs.File interface compatibility.
func (f *SliceFile) Stat() (fs.FileInfo, error) {
	return f.tags, nil
}

// Close is for fs.File interface compatibility.
func (f *SliceFile) Close() error {
	return nil
}

// Package is wrapper for package to hold WPK-file whole content as a slice.
// Gives access to pointed directory in package.
// fs.FS interface implementation.
type Package struct {
	*wpk.Package
	workspace string // workspace directory in package
	bulk      []byte // slice with whole package content
}

// OpenTagset creates file object to give access to nested into package file by given tagset.
func (pack *Package) OpenTagset(ts *wpk.TagsetRaw) (wpk.NestedFile, error) {
	return NewSliceFile(pack, ts)
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
	if pack.bulk, err = os.ReadFile(dpath); err != nil {
		return
	}
	return
}

// Close does nothing, there is no any opened handles.
// Useful for interface compatibility.
// io.Closer implementation.
func (pack *Package) Close() error {
	return nil
}

// Sub clones object and gives access to pointed subdirectory.
// fs.SubFS implementation.
func (pack *Package) Sub(dir string) (sub fs.FS, err error) {
	var fulldir = path.Join(pack.workspace, dir)
	var prefix string
	if fulldir != "." {
		prefix = fulldir + "/" // make prefix slash-terminated
	}
	pack.Enum(func(fkey string, ts *wpk.TagsetRaw) bool {
		if strings.HasPrefix(fkey, prefix) {
			sub = &Package{
				pack.Package,
				fulldir,
				pack.bulk,
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
// fs.ReadFileFS implementation.
func (pack *Package) ReadFile(name string) ([]byte, error) {
	var fullname = path.Join(pack.workspace, name)
	var ts *wpk.TagsetRaw
	var is bool
	if ts, is = pack.Tagset(fullname); !is {
		return nil, &fs.PathError{Op: "readfile", Path: name, Err: fs.ErrNotExist}
	}
	var offset, size = ts.Pos()
	return pack.bulk[offset : offset+size], nil
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
	if dir == "wpk" && pack.workspace == "." {
		var ts = pack.BaseTagset(0, uint(len(pack.bulk)), "wpk")
		return NewSliceFile(pack, ts)
	}

	var fullname = path.Join(pack.workspace, dir)
	if ts, is := pack.Tagset(fullname); is {
		return NewSliceFile(pack, ts)
	}
	return pack.FTT.OpenDir(fullname)
}

// The End.
