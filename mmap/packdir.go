package mmap

import (
	"bytes"
	"net/http"
	"os"
	"strings"

	mm "github.com/schwarzlichtbezirk/mmap-go"
	"github.com/schwarzlichtbezirk/wpk"
)

// System pages granulation for memory mapping system calls.
// The page size on most Unixes is 4KB, but on Windows it's 64KB.
// os.Getpagesize() returns incorrect value on Windows.
const pagesize = int64(64 * 1024)

// Gives access to nested into package file by memory mapping.
// http.File interface implementation.
type MappedFile struct {
	wpk.File
	mm.MMap
}

func (f *MappedFile) OpenTags(pack *PackDir, ts wpk.TagSlice) error {
	// calculate paged size/offset
	var offset, size = ts.Offset(), ts.Size()
	var pgoff = offset % pagesize
	var offsetx = offset - pgoff
	var sizex = ((size+pgoff-1)/pagesize + 1) * pagesize
	// create mapped memory block
	var err error
	if f.MMap, err = mm.MapRegion(pack.fwpk, offsetx, sizex, mm.RDONLY, 0); err != nil {
		return err
	}
	// init file struct
	f.TagSlice = ts
	f.Reader.Reset(f.MMap[pgoff : pgoff+size])
	f.Pack = pack
	return nil
}

// Unmaps memory and closes mapped memory handle.
func (f *MappedFile) Close() error {
	return f.Unmap()
}

// Wrapper for package to get access to nested files as to memory mapped blocks.
// Gives access to directory in package with prefix "pref".
// http.FileSystem interface implementation.
type PackDir struct {
	*wpk.Package
	fwpk *os.File
	ftag MappedFile
	pref string
}

// Returns tags set referred by offset at TAT field.
func (pack *PackDir) NamedTags(key string) (wpk.TagSlice, bool) {
	var tagpos, is = pack.TAT[key]
	return wpk.TagSlice(pack.ftag.MMap[tagpos-pack.TagOffset:]), is
}

// Opens WPK-file package by given file name.
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
	var fi os.FileInfo
	if fi, err = pack.fwpk.Stat(); err != nil {
		return
	}
	var buf bytes.Buffer
	var tags = wpk.Tagset{
		wpk.TID_FID:    wpk.TagUint32(0),
		wpk.TID_offset: wpk.TagUint64(uint64(pack.TagOffset)),
		wpk.TID_size:   wpk.TagUint64(uint64(fi.Size()) - uint64(pack.TagOffset)),
	}
	tags.WriteTo(&buf)
	if err = pack.ftag.OpenTags(pack, buf.Bytes()); err != nil {
		return
	}
	return
}

// Closes file handle. This function must be called only for root object,
// not subdirectories.
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

// Clones object and gives access to pointed subdirectory.
// Copies file handle, so it must be closed only once for root object.
func (pack *PackDir) SubDir(pref string) wpk.Packager {
	pref = wpk.ToKey(pref)
	if len(pref) > 0 && pref[len(pref)-1] != '/' {
		pref += "/"
	}
	return &PackDir{
		pack.Package,
		pack.fwpk,
		pack.ftag,
		pack.pref + pref,
	}
}

// Creates file object to give access to nested into package file by given tagset.
func (pack *PackDir) OpenFile(ts wpk.TagSlice) (http.File, error) {
	var f MappedFile
	return &f, f.OpenTags(pack, ts)
}

// Returns slice with nested into package file content.
// Makes content copy to prevent ambiguous access to closed mapped memory block.
func (pack *PackDir) Extract(key string) ([]byte, error) {
	var ts wpk.TagSlice
	var is bool
	if ts, is = pack.NamedTags(key); !is {
		return nil, wpk.ErrNotFound
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

// Implements access to nested into package file or directory by keyname.
func (pack *PackDir) Open(kname string) (http.File, error) {
	var kpath = pack.pref + strings.TrimPrefix(kname, "/")
	if kpath == "" {
		return wpk.OpenDir(pack, kpath)
	} else if kpath == "wpk" {
		return pack.fwpk, nil
	}

	var key = wpk.ToKey(kpath)
	if ts, is := pack.NamedTags(key); is {
		return pack.OpenFile(ts)
	} else {
		return wpk.OpenDir(pack, kpath)
	}
}

// The End.
