package mmap

import (
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

// Unmaps memory and closes mapped memory handle.
func (f *MappedFile) Close() error {
	return f.Unmap()
}

// Wrapper for package to get access to nested files as to memory mapped blocks.
// Gives access to directory in package with prefix "pref".
// http.FileSystem interface implementation.
type PackDir struct {
	*wpk.Package
	file *os.File
	pref string
}

// Opens WPK-file package by given file name.
func (pack *PackDir) OpenWPK(fname string) (err error) {
	if pack.file, err = os.Open(fname); err != nil {
		return
	}
	pack.pref = ""

	if err = pack.Read(pack.file); err != nil {
		return
	}
	return
}

// Closes file handle. This function must be called only for root object,
// not subdirectories.
func (pack *PackDir) Close() error {
	return pack.file.Close()
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
		pack.file,
		pack.pref + pref,
	}
}

// Creates file object to give access to nested into package file by given tagset.
func (pack *PackDir) OpenFile(tags wpk.Tagset) (http.File, error) {
	// calculate paged size/offset
	var offset, size = tags.Record()
	var pgoff = offset % pagesize
	var offsetx = offset - pgoff
	var sizex = ((size+pgoff-1)/pagesize + 1) * pagesize
	// create mapped memory block
	var f MappedFile
	var err error
	if f.MMap, err = mm.MapRegion(pack.file, offsetx, sizex, mm.RDONLY, 0); err != nil {
		return nil, err
	}
	// init file struct
	f.Tagset = tags
	f.Reader.Reset(f.MMap[pgoff : pgoff+size])
	f.Pack = pack.Package
	return &f, nil
}

// Returns slice with nested into package file content.
// Makes content copy to prevent ambiguous access to closed mapped memory block.
func (pack *PackDir) Extract(key string) ([]byte, error) {
	var tags wpk.Tagset
	var is bool
	if tags, is = pack.Tags[key]; !is {
		return nil, wpk.ErrNotFound
	}
	var f, err = pack.OpenFile(tags)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var size = tags.Size()
	var buf = make([]byte, size, size)
	_, err = f.Read(buf)
	return buf, err
}

// Implements access to nested into package file or directory by keyname.
func (pack *PackDir) Open(kname string) (http.File, error) {
	var kpath = pack.pref + strings.TrimPrefix(kname, "/")
	if kpath == "" {
		return pack.OpenDir(kpath)
	} else if kpath == "wpk" {
		return pack.file, nil
	}

	var key = wpk.ToKey(kpath)
	if tags, is := pack.Tags[key]; is {
		return pack.OpenFile(tags)
	} else {
		return pack.OpenDir(kpath)
	}
}

// The End.
