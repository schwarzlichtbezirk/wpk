package fsys

import (
	"io"
	"io/fs"
	"os"
	"path"
	"strings"

	"github.com/schwarzlichtbezirk/wpk"
)

// ChunkFile structure gives access to nested into package file.
// wpk.NestedFile interface implementation.
type ChunkFile struct {
	wpk.FileReader
	tags *wpk.Tagset_t // has fs.FileInfo interface
	wpkf *os.File
}

// NewChunkFile creates ChunkFile file structure based on given tags slice.
func NewChunkFile(fpath string, ts *wpk.Tagset_t) (f *ChunkFile, err error) {
	var wpkf *os.File
	if wpkf, err = os.Open(fpath); err != nil {
		return
	}
	var offset, size = ts.Pos()
	f = &ChunkFile{
		tags:       ts,
		FileReader: io.NewSectionReader(wpkf, int64(offset), int64(size)),
		wpkf:       wpkf,
	}
	return
}

// Stat is for fs.File interface compatibility.
func (f *ChunkFile) Stat() (fs.FileInfo, error) {
	return f.tags, nil
}

// Close closes associated wpk-file handle.
func (f *ChunkFile) Close() error {
	return f.wpkf.Close()
}

// Package is wrapper for package to get access to nested files as to memory mapped blocks.
// Gives access to pointed directory in package. This type of package can be used for write.
// fs.FS interface implementation.
type Package struct {
	*wpk.Package
	workspace string // workspace directory in package
	fpath     string // package filename
}

// OpenTagset creates file object to give access to nested into package file by given tagset.
func (pack *Package) OpenTagset(ts *wpk.Tagset_t) (wpk.NestedFile, error) {
	return NewChunkFile(pack.fpath, ts)
}

// NewPackage creates new package with given data-part file.
func NewPackage(datpath string, pts wpk.TypeSize) *Package {
	return &Package{
		Package:   wpk.NewPackage(pts),
		workspace: ".",
		fpath:     datpath,
	}
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

	if pack.IsSplitted() {
		pack.fpath = wpk.MakeDataPath(fpath)
	} else {
		pack.fpath = fpath
	}
	return
}

// Close file handle. This function must be called only for root object,
// not subdirectories.
// io.Closer implementation.
func (pack *Package) Close() error {
	return nil
}

// Sub clones object and gives access to pointed subdirectory.
// Copies file handle, so it must be closed only once for root object.
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
				pack.fpath,
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
// Makes content copy to prevent ambiguous access to closed mapped memory block.
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
	var f, err = NewChunkFile(pack.fpath, ts)
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
	return pack.FTT_t.ReadDir(fullname, -1)
}

// Open implements access to nested into package file or directory by keyname.
// fs.FS implementation.
func (pack *Package) Open(dir string) (fs.File, error) {
	if dir == "wpk" && pack.workspace == "." {
		return os.Open(pack.fpath)
	}

	var fullname = path.Join(pack.workspace, dir)
	if ts, is := pack.Tagset(fullname); is {
		return NewChunkFile(pack.fpath, ts)
	}
	return pack.FTT_t.Open(fullname)
}

// The End.
