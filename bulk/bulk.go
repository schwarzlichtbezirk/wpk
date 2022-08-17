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
type SliceFile[TID_t wpk.TID_i, TSize_t wpk.TSize_i] struct {
	wpk.FileReader
	tags *wpk.Tagset_t[TID_t, TSize_t] // has fs.FileInfo interface
}

// NewSliceFile creates SliceFile file structure based on given tags slice.
func NewSliceFile[TID_t wpk.TID_i, TSize_t wpk.TSize_i](pack *Package[TID_t, TSize_t], ts *wpk.Tagset_t[TID_t, TSize_t]) (f *SliceFile[TID_t, TSize_t], err error) {
	var offset, _ = ts.Uint(wpk.TIDoffset)
	var size, _ = ts.Uint(wpk.TIDsize)
	f = &SliceFile[TID_t, TSize_t]{
		tags:       ts,
		FileReader: bytes.NewReader(pack.bulk[offset : offset+size]),
	}
	return
}

// Stat is for fs.File interface compatibility.
func (f *SliceFile[TID_t, TSize_t]) Stat() (fs.FileInfo, error) {
	return f.tags, nil
}

// Close is for fs.File interface compatibility.
func (f *SliceFile[TID_t, TSize_t]) Close() error {
	return nil
}

// Package is wrapper for package to hold WPK-file whole content as a slice.
// Gives access to pointed directory in package.
// fs.FS interface implementation.
type Package[TID_t wpk.TID_i, TSize_t wpk.TSize_i] struct {
	*wpk.Package[TID_t, TSize_t]
	workspace string // workspace directory in package
	bulk      []byte // slice with whole package content
}

// OpenTagset creates file object to give access to nested into package file by given tagset.
func (pack *Package[TID_t, TSize_t]) OpenTagset(ts *wpk.Tagset_t[TID_t, TSize_t]) (wpk.NestedFile, error) {
	return NewSliceFile(pack, ts)
}

// OpenPackage opens WPK-file package by given file name.
func OpenPackage[TID_t wpk.TID_i, TSize_t wpk.TSize_i](fname string, foffset, fsize, fidsz, tssize byte) (pack *Package[TID_t, TSize_t], err error) {
	pack = &Package[TID_t, TSize_t]{
		Package:   wpk.NewPackage[TID_t, TSize_t](foffset, fsize, fidsz, tssize),
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
func (pack *Package[TID_t, TSize_t]) Close() error {
	return nil
}

// Sub clones object and gives access to pointed subdirectory.
// fs.SubFS implementation.
func (pack *Package[TID_t, TSize_t]) Sub(dir string) (df fs.FS, err error) {
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
			df, err = &Package[TID_t, TSize_t]{
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
func (pack *Package[TID_t, TSize_t]) Stat(name string) (fs.FileInfo, error) {
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
// fs.ReadFileFS implementation.
func (pack *Package[TID_t, TSize_t]) ReadFile(name string) ([]byte, error) {
	if !fs.ValidPath(name) {
		return nil, &fs.PathError{Op: "readfile", Path: name, Err: fs.ErrInvalid}
	}
	var ts *wpk.Tagset_t[TID_t, TSize_t]
	var is bool
	if ts, is = pack.Tagset(path.Join(pack.workspace, name)); !is {
		return nil, &fs.PathError{Op: "readfile", Path: name, Err: fs.ErrNotExist}
	}
	var offset, _ = ts.Uint(wpk.TIDoffset)
	var size, _ = ts.Uint(wpk.TIDsize)
	return pack.bulk[offset : offset+size], nil
}

// ReadDir reads the named directory
// and returns a list of directory entries sorted by filename.
func (pack *Package[TID_t, TSize_t]) ReadDir(dir string) ([]fs.DirEntry, error) {
	return wpk.ReadDir[TID_t, TSize_t](pack, path.Join(pack.workspace, dir), -1)
}

// Open implements access to nested into package file or directory by keyname.
// fs.FS implementation.
func (pack *Package[TID_t, TSize_t]) Open(dir string) (fs.File, error) {
	if dir == "wpk" && pack.workspace == "." {
		var ts = (&wpk.Tagset_t[TID_t, TSize_t]{}).
			Put(wpk.TIDoffset, wpk.TagUintLen(0, pack.PTS(wpk.PTSfoffset))).
			Put(wpk.TIDsize, wpk.TagUintLen(uint(len(pack.bulk)), pack.PTS(wpk.PTSfsize)))
		return NewSliceFile(pack, ts)
	}

	var fullname = path.Join(pack.workspace, dir)
	if ts, is := pack.Tagset(fullname); is {
		return NewSliceFile(pack, ts)
	}
	return wpk.OpenDir[TID_t, TSize_t](pack, fullname)
}

// The End.
