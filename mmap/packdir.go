package mmap

import (
	"bytes"
	"io/fs"
	"os"
	"strings"

	mm "github.com/schwarzlichtbezirk/mmap-go"
	"github.com/schwarzlichtbezirk/wpk"
)

// System pages granulation for memory mapping system calls.
// The page size on most Unixes is 4KB, but on Windows it's 64KB.
// os.Getpagesize() returns incorrect value on Windows.
const pagesize = int64(64 * 1024)

// MappedFile gives access to nested into package file by memory mapping.
// http.File interface implementation.
type MappedFile struct {
	wpk.File
	mm.MMap
	region []byte
}

// OpenTags maps nested to package file with given tags slice.
func (f *MappedFile) OpenTags(pack *PackDir, ts wpk.TagSlice) error {
	// calculate paged size/offset
	var offset, size = ts.Offset(), ts.Size()
	var pgoff = offset % pagesize
	var offsetx = offset - pgoff
	var sizex = size + pgoff
	// create mapped memory block
	var err error
	if f.MMap, err = mm.MapRegion(pack.fwpk, offsetx, sizex, mm.RDONLY, 0); err != nil {
		return err
	}
	f.region = f.MMap[pgoff : pgoff+size]
	// init file struct
	f.TagSlice = ts
	f.Reader.Reset(f.region)
	f.Pack = pack
	return nil
}

// Close unmaps memory and closes mapped memory handle.
func (f *MappedFile) Close() error {
	return f.Unmap()
}

// PackDir is wrapper for package to get access to nested files as to memory mapped blocks.
// Gives access to directory in package with prefix "pref".
// http.FileSystem interface implementation.
type PackDir struct {
	*wpk.Package
	fwpk *os.File
	ftag MappedFile
	pref string
}

// OpenFile creates file object to give access to nested into package file by given tagset.
func (pack *PackDir) OpenFile(ts wpk.TagSlice) (fs.File, error) {
	var f MappedFile
	return &f, f.OpenTags(pack, ts)
}

// NamedTags returns tags set referred by offset at FTT field.
func (pack *PackDir) NamedTags(key string) (wpk.TagSlice, bool) {
	var tagpos, is = pack.FTT[key]
	return wpk.TagSlice(pack.ftag.region[tagpos-pack.TagOffset:]), is
}

// OpenWPK opens WPK-file package by given file name.
func (pack *PackDir) OpenWPK(fname string) (err error) {
	if pack.Package == nil {
		pack.Package = &wpk.Package{}
	}
	if pack.fwpk, err = os.Open(fname); err != nil {
		return
	}
	pack.pref = ""

	if err = pack.Read(pack.fwpk); err != nil {
		return
	}

	// open tags set file
	var fi fs.FileInfo
	if fi, err = pack.fwpk.Stat(); err != nil {
		return
	}
	var buf bytes.Buffer
	var tags = wpk.Tagset{
		wpk.TIDfid:    wpk.TagUint32(0),
		wpk.TIDoffset: wpk.TagUint64(uint64(pack.TagOffset)),
		wpk.TIDsize:   wpk.TagUint64(uint64(fi.Size()) - uint64(pack.TagOffset)),
	}
	tags.WriteTo(&buf)
	if err = pack.ftag.OpenTags(pack, buf.Bytes()); err != nil {
		return
	}
	return
}

// Close file handle. This function must be called only for root object,
// not subdirectories.
// io.Closer implementation.
func (pack *PackDir) Close() error {
	var err1 = pack.ftag.Close()
	var err2 = pack.fwpk.Close()
	if err1 != nil {
		return err1
	}
	if err2 != nil {
		return err2
	}
	return nil
}

// Sub clones object and gives access to pointed subdirectory.
// Copies file handle, so it must be closed only once for root object.
// fs.SubFS implementation.
func (pack *PackDir) Sub(dir string) (fs.FS, error) {
	dir = wpk.ToKey(dir)
	if len(dir) > 0 && dir[len(dir)-1] != '/' {
		dir += "/"
	}
	return &PackDir{
		pack.Package,
		pack.fwpk,
		pack.ftag,
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
// Makes content copy to prevent ambiguous access to closed mapped memory block.
// fs.ReadFileFS implementation.
func (pack *PackDir) ReadFile(name string) ([]byte, error) {
	var ts wpk.TagSlice
	var is bool
	if ts, is = pack.NamedTags(name); !is {
		return nil, &fs.PathError{Op: "readfile", Path: name, Err: fs.ErrNotExist}
	}
	var f, err = pack.OpenFile(ts)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var size = ts.Size()
	var buf = make([]byte, size)
	_, err = f.Read(buf)
	return buf, err
}

// Open implements access to nested into package file or directory by keyname.
// fs.FS implementation.
func (pack *PackDir) Open(kname string) (fs.File, error) {
	var kpath = pack.pref + strings.TrimPrefix(kname, "/")
	if kpath == "" {
		return wpk.OpenDir(pack, kpath)
	} else if kpath == "wpk" {
		return pack.fwpk, nil
	}

	var key = wpk.ToKey(kpath)
	if ts, is := pack.NamedTags(key); is {
		return pack.OpenFile(ts)
	}
	return wpk.OpenDir(pack, kpath)
}

// The End.
