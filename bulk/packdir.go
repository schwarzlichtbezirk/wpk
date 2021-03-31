package bulk

import (
	"bytes"
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
	tags wpk.TagSlice // has fs.FileInfo interface
}

// NewSliceFile creates SliceFile file structure based on given tags slice.
func NewSliceFile(pack *PackDir, ts wpk.TagSlice) (f *SliceFile, err error) {
	var offset, size = ts.Offset(), ts.Size()
	f = &SliceFile{
		tags:       ts,
		FileReader: bytes.NewReader(pack.bulk[offset : offset+size]),
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

// PackDir is wrapper for package to hold WPK-file whole content as a slice.
// Gives access to pointed directory in package.
// fs.FS interface implementation.
type PackDir struct {
	*wpk.Package
	workspace string // workspace directory in package
	bulk      []byte
}

// OpenTags creates file object to give access to nested into package file by given tagset.
func (pack *PackDir) OpenTags(ts wpk.TagSlice) (wpk.NestedFile, error) {
	return NewSliceFile(pack, ts)
}

// NamedTags returns tags set referred by offset at named file tags map field.
// Function receives normalized full path of file.
func (pack *PackDir) NamedTags(key string) (wpk.TagSlice, bool) {
	if tagpos, is := pack.Tags[key]; is {
		return pack.bulk[tagpos:], true
	} else {
		return nil, false
	}
}

// OpenImage opens WPK-file package by given file name.
func OpenImage(fname string) (pack *PackDir, err error) {
	pack = &PackDir{Package: &wpk.Package{}}
	pack.workspace = "."

	var bulk []byte
	if bulk, err = os.ReadFile(fname); err != nil {
		return
	}
	pack.bulk = bulk
	if err = pack.Read(bytes.NewReader(bulk)); err != nil {
		return
	}
	return
}

// Close does nothing, there is no any opened handles.
// Useful for interface compatibility.
// io.Closer implementation.
func (pack *PackDir) Close() error {
	return nil
}

// Sub clones object and gives access to pointed subdirectory.
// fs.SubFS implementation.
func (pack *PackDir) Sub(dir string) (fs.FS, error) {
	if !fs.ValidPath(dir) {
		return nil, &fs.PathError{Op: "sub", Path: dir, Err: fs.ErrInvalid}
	}
	var workspace = path.Join(pack.workspace, dir)
	var prefixdir string
	if workspace != "." {
		prefixdir = workspace + "/" // make prefix slash-terminated
	}
	for key := range pack.NFTO() {
		if strings.HasPrefix(key, prefixdir) {
			return &PackDir{
				pack.Package,
				workspace,
				pack.bulk,
			}, nil
		}
	}
	return nil, &fs.PathError{Op: "sub", Path: dir, Err: fs.ErrNotExist}
}

// Stat returns a fs.FileInfo describing the file.
// fs.StatFS implementation.
func (pack *PackDir) Stat(name string) (fs.FileInfo, error) {
	if !fs.ValidPath(name) {
		return nil, &fs.PathError{Op: "stat", Path: name, Err: fs.ErrInvalid}
	}
	var ts wpk.TagSlice
	var is bool
	if ts, is = pack.NamedTags(wpk.Normalize(path.Join(pack.workspace, name))); !is {
		return nil, &fs.PathError{Op: "stat", Path: name, Err: fs.ErrNotExist}
	}
	return ts, nil
}

// ReadFile returns slice with nested into package file content.
// fs.ReadFileFS implementation.
func (pack *PackDir) ReadFile(name string) ([]byte, error) {
	if !fs.ValidPath(name) {
		return nil, &fs.PathError{Op: "readfile", Path: name, Err: fs.ErrInvalid}
	}
	var offset, size int64
	var ts wpk.TagSlice
	var is bool
	if ts, is = pack.NamedTags(wpk.Normalize(path.Join(pack.workspace, name))); !is {
		return nil, &fs.PathError{Op: "readfile", Path: name, Err: fs.ErrNotExist}
	}
	offset, size = ts.Offset(), ts.Size()
	return pack.bulk[offset : offset+size], nil
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
		var buf bytes.Buffer
		wpk.Tagset{
			wpk.TIDfid:    wpk.TagUint32(0),
			wpk.TIDoffset: wpk.TagUint64(0),
			wpk.TIDsize:   wpk.TagUint64(uint64(len(pack.bulk))),
		}.WriteTo(&buf)
		return NewSliceFile(pack, buf.Bytes())
	}

	var fullname = path.Join(pack.workspace, dir)
	if ts, is := pack.NamedTags(wpk.Normalize(fullname)); is {
		return NewSliceFile(pack, ts)
	}
	return wpk.OpenDir(pack, fullname)
}

// The End.
