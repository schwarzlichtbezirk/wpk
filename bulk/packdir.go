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
		Pack:     pack,
	}, nil
}

// NamedTags returns tags set referred by offset at FTT field.
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
	dir = wpk.ToKey(dir)
	if len(dir) > 0 && dir[len(dir)-1] != '/' {
		dir += "/"
	}
	return &PackDir{
		pack.Package,
		pack.bulk,
		pack.pref + dir,
	}, nil
}

// Stat returns a fs.FileInfo describing the file.
// fs.StatFS implementation.
func (pack *PackDir) Stat(name string) (fs.FileInfo, error) {
	var ts wpk.TagSlice
	var is bool
	if ts, is = pack.NamedTags(name); !is {
		return nil, &fs.PathError{Op: "stat", Path: name, Err: fs.ErrNotExist}
	}
	return ts, nil
}

// ReadFile returns slice with nested into package file content.
// fs.ReadFileFS implementation.
func (pack *PackDir) ReadFile(name string) ([]byte, error) {
	var offset, size int64
	if ts, is := pack.NamedTags(name); is {
		offset, size = ts.Offset(), ts.Size()
	} else {
		return nil, &fs.PathError{Op: "readfile", Path: name, Err: fs.ErrNotExist}
	}
	return pack.bulk[offset : offset+size], nil
}

// Open implements access to nested into package file or directory by keyname.
// fs.FS implementation.
func (pack *PackDir) Open(kname string) (fs.File, error) {
	var kpath = pack.pref + strings.TrimPrefix(kname, "/")
	if kpath == "" {
		return wpk.OpenDir(pack, kpath)
	} else if kpath == "wpk" {
		var buf bytes.Buffer
		wpk.Tagset{
			wpk.TIDfid:    wpk.TagUint32(0),
			wpk.TIDoffset: wpk.TagUint64(0),
			wpk.TIDsize:   wpk.TagUint64(uint64(len(pack.bulk))),
		}.WriteTo(&buf)
		return &wpk.File{
			TagSlice: buf.Bytes(),
			Reader:   *bytes.NewReader(pack.bulk),
			Pack:     pack,
		}, nil
	}

	var key = wpk.ToKey(kpath)
	if ts, is := pack.NamedTags(key); is {
		return pack.OpenFile(ts)
	}
	return wpk.OpenDir(pack, kpath)
}

// The End.
