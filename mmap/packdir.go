package mmap

import (
	"net/http"
	"os"
	"strings"

	mm "github.com/edsrzf/mmap-go"
	"github.com/schwarzlichtbezirk/wpk"
)

// System pages granulation for memory mapping system calls.
const pagesize int64 = 65536 // 64K

// Gives access to nested into package file by memory mapping.
// http.File interface implementation.
type File struct {
	wpk.File
	mm.MMap
}

// Unmaps memory and closes mapped memory handle.
func (f *File) Close() error {
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

// Reads WPK-file package by given file name.
func (pack *PackDir) ReadWPK(fname string) (err error) {
	if pack.file, err = os.Open(fname); err != nil {
		return
	}
	pack.Package = &wpk.Package{}
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
func (pack *PackDir) SubDir(pref string) *PackDir {
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
func (pack *PackDir) NewFile(tags wpk.Tagset) (http.File, error) {
	var f File
	var offset, size = tags.Record()
	// calculate paged size/offset
	var pgoff = offset % pagesize
	var offsetx = offset - pgoff
	var sizex = int(((size+pgoff-1)/pagesize + 1) * pagesize)
	// create mapped memory block
	var err error
	if f.MMap, err = mm.MapRegion(pack.file, sizex, mm.RDONLY, 0, offsetx); err != nil {
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
func (pack *PackDir) Extract(tags wpk.Tagset) ([]byte, error) {
	var f, err = pack.NewFile(tags)
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
func (pack *PackDir) Open(kpath string) (http.File, error) {
	var key = pack.pref + strings.TrimPrefix(wpk.ToKey(kpath), "/")
	if key == "" {
		return pack.NewDir(key), nil
	} else if key == "wpk" {
		return pack.file, nil
	}

	if tags, is := pack.Tags[key]; is {
		return pack.NewFile(tags)
	} else {
		if key[len(key)-1] != '/' { // bring key to dir
			key += "/"
		}
		for k := range pack.Tags {
			if strings.HasPrefix(k, key) {
				return pack.NewDir(key), nil
			}
		}
		return nil, wpk.ErrNotFound
	}
}

// The End.
