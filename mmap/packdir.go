package mmap

import (
	"io"
	"net/http"
	"os"
	"strings"

	mm "github.com/edsrzf/mmap-go"
	"github.com/schwarzlichtbezirk/wpk"
)

const pagesize int64 = 65536 // 64K

type File struct {
	wpk.File
	mm.MMap
}

func (f *File) Close() error {
	return f.Unmap()
}

type PackDir struct {
	*wpk.Package
	file *os.File
	pref string
}

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

func (pack *PackDir) Close() error {
	return pack.file.Close()
}

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

func (pack *PackDir) NewFile(tags wpk.Tagset) http.File {
	var f File
	var offset, size = tags.Record()
	// calculate paged size/offset
	var pgoff = offset % pagesize
	var offsetx = offset - pgoff
	var sizex = int(((size+pgoff-1)/pagesize + 1) * pagesize)
	// create mapped memory block
	f.MMap, _ = mm.MapRegion(pack.file, sizex, mm.RDONLY, 0, offsetx)
	// init file struct
	f.Tagset = tags
	f.Reader.Reset(f.MMap[pgoff : pgoff+size])
	f.Pack = pack.Package
	return &f
}

func (pack *PackDir) Extract(tags wpk.Tagset) []byte {
	var size = tags.Size()
	var buf = make([]byte, size, size)
	var f = pack.NewFile(tags)
	defer f.Close()
	f.Read(buf)
	return buf
}

func (pack *PackDir) Open(kpath string) (http.File, error) {
	var key = pack.pref + strings.TrimPrefix(wpk.ToKey(kpath), "/")
	if key == "" {
		return pack.NewDir(key), nil
	} else if key == "wpk" {
		if _, err := pack.file.Seek(0, io.SeekStart); err != nil {
			return nil, err
		}
		return pack.file, nil
	}

	if tags, is := pack.Tags[key]; is {
		return pack.NewFile(tags), nil
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
