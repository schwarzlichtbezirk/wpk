package bulk

import (
	"bytes"
	"io/fs"
	"os"
	"path"
	"strings"

	"github.com/schwarzlichtbezirk/wpk"
)

// PackDir is wrapper for package to hold WPK-file whole content as a slice.
// Gives access to pointed directory in package.
// fs.FS interface implementation.
type PackDir struct {
	*wpk.Package
	bulk      []byte
	workspace string // workspace directory in package
}

// OpenFile creates file object to give access to nested into package file by given tagset.
func (pack *PackDir) OpenFile(ts wpk.TagSlice) (fs.File, error) {
	var offset, size = ts.Offset(), ts.Size()
	return &wpk.File{
		TagSlice: ts,
		Reader:   *bytes.NewReader(pack.bulk[offset : offset+size]),
	}, nil
}

// NamedTags returns tags set referred by offset at named file tags map field.
// Function receives normalized full path of file.
func (pack *PackDir) NamedTags(key string) (wpk.TagSlice, bool) {
	var tagpos, is = pack.Tags[key]
	return pack.bulk[tagpos:], is
}

// OpenImage opens WPK-file package by given file name.
func OpenImage(fname string) (pack *PackDir, err error) {
	pack = &PackDir{Package: &wpk.Package{}}

	var bulk []byte
	if bulk, err = os.ReadFile(fname); err != nil {
		return
	}

	if pack.Package == nil {
		pack.Package = &wpk.Package{}
	}
	pack.bulk = bulk
	pack.workspace = "."

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
	var rootdir = path.Join(pack.workspace, dir)
	var rootcmp string
	if rootdir != "." {
		rootcmp = rootdir + "/" // make prefix slash-terminated
	}
	for key := range pack.NFTO() {
		if strings.HasPrefix(key, rootcmp) {
			return &PackDir{
				pack.Package,
				pack.bulk,
				rootdir,
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
		return &wpk.File{
			TagSlice: buf.Bytes(),
			Reader:   *bytes.NewReader(pack.bulk),
		}, nil
	}

	var rootdir = path.Join(pack.workspace, dir)
	if ts, is := pack.NamedTags(wpk.Normalize(rootdir)); is {
		return pack.OpenFile(ts)
	}
	return wpk.OpenDir(pack, rootdir)
}

// The End.
