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
	tags *wpk.Tagset_t // has fs.FileInfo interface
}

// NewSliceFile creates SliceFile file structure based on given tags slice.
func NewSliceFile(pack *Package, ts *wpk.Tagset_t) (f *SliceFile, err error) {
	var offset, _ = ts.FOffset()
	var size, _ = ts.FSize()
	f = &SliceFile{
		tags:       ts,
		FileReader: bytes.NewReader(pack.bulk[offset : offset+wpk.FOffset_t(size)]),
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
func (pack *Package) OpenTagset(ts *wpk.Tagset_t) (wpk.NestedFile, error) {
	return NewSliceFile(pack, ts)
}

// OpenPackage opens WPK-file package by given file name.
func OpenPackage(fname string) (pack *Package, err error) {
	pack = &Package{
		Package:   &wpk.Package{},
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

	var bulk []byte
	if pack.IsSplitted() {
		if bulk, err = os.ReadFile(wpk.MakeDataPath(fname)); err != nil {
			return
		}
	} else {
		var offset, size = pack.DataPos()
		bulk = make([]byte, offset+size)
		if _, err = r.Seek(0, io.SeekStart); err != nil {
			return
		}
		if _, err = r.Read(bulk); err != nil {
			return
		}
	}
	pack.bulk = bulk
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
func (pack *Package) Sub(dir string) (df fs.FS, err error) {
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
			df, err = &Package{
				pack.Package,
				workspace,
				pack.bulk,
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
func (pack *Package) Stat(name string) (fs.FileInfo, error) {
	if !fs.ValidPath(name) {
		return nil, &fs.PathError{Op: "stat", Path: name, Err: fs.ErrInvalid}
	}
	var ts *wpk.Tagset_t
	var is bool
	if ts, is = pack.Tagset(path.Join(pack.workspace, name)); !is {
		return nil, &fs.PathError{Op: "stat", Path: name, Err: fs.ErrNotExist}
	}
	return ts, nil
}

// ReadFile returns slice with nested into package file content.
// fs.ReadFileFS implementation.
func (pack *Package) ReadFile(name string) ([]byte, error) {
	if !fs.ValidPath(name) {
		return nil, &fs.PathError{Op: "readfile", Path: name, Err: fs.ErrInvalid}
	}
	var ts *wpk.Tagset_t
	var is bool
	if ts, is = pack.Tagset(path.Join(pack.workspace, name)); !is {
		return nil, &fs.PathError{Op: "readfile", Path: name, Err: fs.ErrNotExist}
	}
	var offset, _ = ts.FOffset()
	var size, _ = ts.FSize()
	return pack.bulk[offset : offset+wpk.FOffset_t(size)], nil
}

// ReadDir reads the named directory
// and returns a list of directory entries sorted by filename.
func (pack *Package) ReadDir(dir string) ([]fs.DirEntry, error) {
	return wpk.ReadDir(pack, path.Join(pack.workspace, dir), -1)
}

// Open implements access to nested into package file or directory by keyname.
// fs.FS implementation.
func (pack *Package) Open(dir string) (fs.File, error) {
	if dir == "wpk" && pack.workspace == "." {
		var ts = (&wpk.Tagset_t{}).
			Put(wpk.TIDfid, wpk.TagFID(0)).
			Put(wpk.TIDoffset, wpk.TagFOffset(0)).
			Put(wpk.TIDsize, wpk.TagFSize(wpk.FSize_t(len(pack.bulk))))
		return NewSliceFile(pack, ts)
	}

	var fullname = path.Join(pack.workspace, dir)
	if ts, is := pack.Tagset(fullname); is {
		return NewSliceFile(pack, ts)
	}
	return wpk.OpenDir(pack, fullname)
}

// The End.
