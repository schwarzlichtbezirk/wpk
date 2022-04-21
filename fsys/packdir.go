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
func NewChunkFile(fname string, ts *wpk.Tagset_t) (f *ChunkFile, err error) {
	var wpkf *os.File
	if wpkf, err = os.Open(fname); err != nil {
		return
	}
	var offset, _ = ts.FOffset()
	var size, _ = ts.FSize()
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

// PackDir is wrapper for package to get access to nested files as to memory mapped blocks.
// Gives access to pointed directory in package.
// fs.FS interface implementation.
type PackDir struct {
	*wpk.Package
	workspace string // workspace directory in package
	fname     string // package filename
}

// OpenTagset creates file object to give access to nested into package file by given tagset.
func (pack *PackDir) OpenTagset(ts *wpk.Tagset_t) (wpk.NestedFile, error) {
	return NewChunkFile(pack.fname, ts)
}

// OpenPackage opens WPK-file package by given file name.
func OpenPackage(fname string) (pack *PackDir, err error) {
	pack = &PackDir{Package: &wpk.Package{}}
	pack.workspace = "."
	pack.fname = fname

	var filewpk *os.File
	if filewpk, err = os.Open(fname); err != nil {
		return
	}
	defer filewpk.Close()

	if err = pack.OpenFTT(filewpk); err != nil {
		return
	}
	return
}

// Close file handle. This function must be called only for root object,
// not subdirectories.
// io.Closer implementation.
func (pack *PackDir) Close() error {
	return nil
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
				pack.fname,
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
	if ts, is = pack.Tagset(path.Join(pack.workspace, name)); !is {
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
	if ts, is = pack.Tagset(path.Join(pack.workspace, name)); !is {
		return nil, &fs.PathError{Op: "readfile", Path: name, Err: fs.ErrNotExist}
	}
	var f, err = NewChunkFile(pack.fname, ts)
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
		return os.Open(pack.fname)
	}

	var fullname = path.Join(pack.workspace, dir)
	if ts, is := pack.Tagset(fullname); is {
		return NewChunkFile(pack.fname, ts)
	}
	return wpk.OpenDir(pack, fullname)
}

// The End.
