package mmap

import (
	"bytes"
	"io/fs"
	"os"
	"path"
	"strings"

	mm "github.com/schwarzlichtbezirk/mmap-go"
	"github.com/schwarzlichtbezirk/wpk"
)

// System pages granulation for memory mapping system calls.
// The page size on most Unixes is 4KB, but on Windows it's 64KB.
// os.Getpagesize() returns incorrect value on Windows.
const pagesize = int64(64 * 1024)

// MappedFile structure gives access to nested into package file by memory mapping.
// wpk.NestedFile interface implementation.
type MappedFile struct {
	wpk.FileReader
	tags   wpk.TagSlice // has fs.FileInfo interface
	region []byte
	mm.MMap
}

// NewMappedFile maps nested to package file based on given tags slice.
func NewMappedFile(pack *PackDir, ts wpk.TagSlice) (f *MappedFile, err error) {
	// calculate paged size/offset
	var offset, size = ts.Offset(), ts.Size()
	var pgoff = offset % pagesize
	var offsetx = offset - pgoff
	var sizex = size + pgoff
	// create mapped memory block
	var mmap mm.MMap
	if mmap, err = mm.MapRegion(pack.filewpk, offsetx, sizex, mm.RDONLY, 0); err != nil {
		return
	}
	f = &MappedFile{
		tags:       ts,
		FileReader: bytes.NewReader(mmap[pgoff : pgoff+size]),
		region:     mmap[pgoff : pgoff+size],
		MMap:       mmap,
	}
	return
}

// Stat is for fs.File interface compatibility.
func (f *MappedFile) Stat() (fs.FileInfo, error) {
	return f.tags, nil
}

// Close unmaps memory and closes mapped memory handle.
func (f *MappedFile) Close() error {
	return f.Unmap()
}

// PackDir is wrapper for package to get access to nested files as to memory mapped blocks.
// Gives access to pointed directory in package.
// fs.FS interface implementation.
type PackDir struct {
	*wpk.Package
	workspace string // workspace directory in package
	filewpk   *os.File
	ftt       *MappedFile
}

// OpenTags creates file object to give access to nested into package file by given tagset.
func (pack *PackDir) OpenTags(ts wpk.TagSlice) (wpk.NestedFile, error) {
	return NewMappedFile(pack, ts)
}

// NamedTags returns tags set referred by offset at named file tags map field.
// Function receives normalized full path of file.
func (pack *PackDir) NamedTags(key string) (wpk.TagSlice, bool) {
	if tagpos, is := pack.Tags[key]; is {
		return pack.ftt.region[tagpos-wpk.OFFSET(pack.FTTOffset()):], true
	} else {
		return nil, false
	}
}

// OpenImage opens WPK-file package by given file name.
func OpenImage(fname string) (pack *PackDir, err error) {
	pack = &PackDir{Package: &wpk.Package{}}
	pack.workspace = "."

	if pack.filewpk, err = os.Open(fname); err != nil {
		return
	}
	if err = pack.Read(pack.filewpk); err != nil {
		return
	}

	// open tags set file
	var buf bytes.Buffer
	wpk.Tagset{
		wpk.TIDfid:    wpk.TagUint32(0),
		wpk.TIDoffset: wpk.TagUint64(uint64(pack.FTTOffset())),
		wpk.TIDsize:   wpk.TagUint64(uint64(pack.FTTSize())),
	}.WriteTo(&buf)
	if pack.ftt, err = NewMappedFile(pack, buf.Bytes()); err != nil {
		return
	}
	return
}

// Close file handle. This function must be called only for root object,
// not subdirectories.
// io.Closer implementation.
func (pack *PackDir) Close() error {
	var err1 = pack.ftt.Close()
	var err2 = pack.filewpk.Close()
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
				pack.filewpk,
				pack.ftt,
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
// Makes content copy to prevent ambiguous access to closed mapped memory block.
// fs.ReadFileFS implementation.
func (pack *PackDir) ReadFile(name string) ([]byte, error) {
	if !fs.ValidPath(name) {
		return nil, &fs.PathError{Op: "readfile", Path: name, Err: fs.ErrInvalid}
	}
	var ts wpk.TagSlice
	var is bool
	if ts, is = pack.NamedTags(wpk.Normalize(path.Join(pack.workspace, name))); !is {
		return nil, &fs.PathError{Op: "readfile", Path: name, Err: fs.ErrNotExist}
	}
	var f, err = NewMappedFile(pack, ts)
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
func (pack *PackDir) ReadDir(dir string) ([]fs.DirEntry, error) {
	return wpk.ReadDir(pack, path.Join(pack.workspace, dir), -1)
}

// Open implements access to nested into package file or directory by keyname.
// fs.FS implementation.
func (pack *PackDir) Open(dir string) (fs.File, error) {
	if dir == "wpk" && pack.workspace == "." {
		return pack.filewpk, nil
	}

	var fullname = path.Join(pack.workspace, dir)
	if ts, is := pack.NamedTags(wpk.Normalize(fullname)); is {
		return NewMappedFile(pack, ts)
	}
	return wpk.OpenDir(pack, fullname)
}

// The End.
