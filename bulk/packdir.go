package bulk

import (
	"bytes"
	"io/fs"
	"io/ioutil"
	"strings"

	"github.com/schwarzlichtbezirk/wpk"
)

// PackDir is wrapper for package to hold WPK-file whole content as a slice.
// Gives access to directory in package with prefix "pref".
// http.FileSystem interface implementation.
type PackDir struct {
	*wpk.Package
	bulk []byte
	pref string
}

// OpenFile creates file object to give access to nested into package file by given tagset.
func (pack *PackDir) OpenFile(ts wpk.TagSlice) (fs.File, error) {
	var offset, size = ts.Offset(), ts.Size()
	return &wpk.File{
		TagSlice: ts,
		Reader:   *bytes.NewReader(pack.bulk[offset : offset+size]),
	}, nil
}

// NamedTags returns tags set referred by offset at FTT field.
// Function receives normalized full path of file.
func (pack *PackDir) NamedTags(key string) (wpk.TagSlice, bool) {
	var tagpos, is = pack.FTT[key]
	return pack.bulk[tagpos:], is
}

// OpenWPK opens WPK-file package by given file name.
func (pack *PackDir) OpenWPK(fname string) (err error) {
	var bulk []byte
	if bulk, err = ioutil.ReadFile(fname); err != nil {
		return
	}

	if pack.Package == nil {
		pack.Package = &wpk.Package{}
	}
	pack.bulk = bulk
	pack.pref = ""

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
	if dir != "" && !fs.ValidPath(dir) {
		return nil, &fs.PathError{Op: "sub", Path: dir, Err: fs.ErrInvalid}
	}
	var subpath = wpk.Normalize(dir)
	if subpath == "." {
		subpath = ""
	} else if len(subpath) > 0 && subpath[len(subpath)-1] != '/' {
		subpath += "/"
	}
	var rootpath = pack.pref + subpath
	for key := range pack.Enum() {
		if strings.HasPrefix(key, rootpath) {
			return &PackDir{
				pack.Package,
				pack.bulk,
				rootpath,
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
	if ts, is = pack.NamedTags(wpk.Normalize(pack.pref + name)); !is {
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
	if ts, is := pack.NamedTags(wpk.Normalize(pack.pref + name)); is {
		offset, size = ts.Offset(), ts.Size()
	} else {
		return nil, &fs.PathError{Op: "readfile", Path: name, Err: fs.ErrNotExist}
	}
	return pack.bulk[offset : offset+size], nil
}

// ReadDir reads the named directory
// and returns a list of directory entries sorted by filename.
func (pack *PackDir) ReadDir(subpath string) ([]fs.DirEntry, error) {
	return wpk.ReadDir(pack, pack.pref+subpath, 0)
}

// Open implements access to nested into package file or directory by keyname.
// fs.FS implementation.
func (pack *PackDir) Open(subpath string) (fs.File, error) {
	var rootpath = pack.pref + subpath
	if subpath == "" || subpath == "." {
		return wpk.OpenDir(pack, pack.pref)
	} else if rootpath == "wpk" {
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

	if ts, is := pack.NamedTags(wpk.Normalize(rootpath)); is {
		return pack.OpenFile(ts)
	}
	return wpk.OpenDir(pack, rootpath)
}

// The End.
