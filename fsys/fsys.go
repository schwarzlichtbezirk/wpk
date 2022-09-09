package fsys

import (
	"io"
	"io/fs"
	"os"
	"path"
	"strings"

	"github.com/schwarzlichtbezirk/wpk"
)

type ReaserAtCloser interface {
	io.ReaderAt
	io.Closer
}

// ChunkFile structure gives access to nested into package file.
// wpk.NestedFile interface implementation.
type ChunkFile struct {
	wpk.FileReader
	wpkf io.Closer
	tags *wpk.TagsetRaw // has fs.FileInfo interface
}

// NewChunkFile creates ChunkFile file structure based on given tags slice.
func NewChunkFile(fpath string, ts *wpk.TagsetRaw) (f *ChunkFile, err error) {
	var wpkf *os.File
	if wpkf, err = os.Open(fpath); err != nil {
		return
	}
	var offset, size = ts.Pos()
	f = &ChunkFile{
		FileReader: io.NewSectionReader(wpkf, int64(offset), int64(size)),
		wpkf:       wpkf,
		tags:       ts,
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
	dpath     string // package filename
}

// OpenTagset creates file object to give access to nested into package file by given tagset.
func (pack *Package) OpenTagset(ts *wpk.TagsetRaw) (wpk.NestedFile, error) {
	return NewChunkFile(pack.dpath, ts)
}

// NewPackage creates new package with given data-part file.
func NewPackage(datpath string, pts wpk.TypeSize) *Package {
	return &Package{
		Package:   wpk.NewPackage(pts),
		workspace: ".",
		dpath:     datpath,
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
		pack.dpath = wpk.MakeDataPath(fpath)
	} else {
		pack.dpath = fpath
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
				pack.dpath,
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
	var f, err = NewChunkFile(pack.dpath, ts)
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
		return os.Open(pack.dpath)
	}

	if ts, is := pack.Tagset(fullname); is {
		return NewChunkFile(pack.dpath, ts)
	}
	return pack.FTT.OpenDir(fullname)
}

// The End.
