package bulk

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/schwarzlichtbezirk/wpk"
)

// Wrapper for package to hold WPK-file whole content as a slice.
// Gives access to directory in package with prefix "pref".
// http.FileSystem interface implementation.
type PackDir struct {
	*wpk.Package
	bulk []byte
	pref string
}

// Opens WPK-file package by given file name.
func (pack *PackDir) OpenWPK(fname string) (err error) {
	var bulk []byte
	if bulk, err = ioutil.ReadFile(fname); err != nil {
		return
	}

	pack.bulk = bulk
	pack.pref = ""

	if err = pack.Read(bytes.NewReader(bulk)); err != nil {
		return
	}
	return
}

// Does nothing, there is no any opened handles.
// Useful for interface compatibility.
func (pack *PackDir) Close() error {
	return nil
}

// Clones object and gives access to pointed subdirectory.
func (pack *PackDir) SubDir(pref string) wpk.Packager {
	pref = wpk.ToKey(pref)
	if len(pref) > 0 && pref[len(pref)-1] != '/' {
		pref += "/"
	}
	return &PackDir{
		pack.Package,
		pack.bulk,
		pack.pref + pref,
	}
}

// Creates file object to give access to nested into package file by given tagset.
func (pack *PackDir) OpenFile(tags wpk.Tagset) (http.File, error) {
	var offset, size = tags.Record()
	return &wpk.File{
		Tagset: tags,
		Reader: *bytes.NewReader(pack.bulk[offset : offset+size]),
		Pack:   pack.Package,
	}, nil
}

// Returns slice with nested into package file content.
func (pack *PackDir) Extract(key string) ([]byte, error) {
	var offset, size, err = pack.NamedRecord(key)
	if err != nil {
		return nil, err
	}
	return pack.bulk[offset : offset+size], nil
}

// Implements access to nested into package file or directory by keyname.
func (pack *PackDir) Open(kname string) (http.File, error) {
	var kpath = pack.pref + strings.TrimPrefix(kname, "/")
	if kpath == "" {
		return pack.OpenDir(kpath)
	} else if kpath == "wpk" {
		return &wpk.File{
			Tagset: wpk.Tagset{
				wpk.TID_FID:    wpk.TagUint32(0),
				wpk.TID_offset: wpk.TagUint64(0),
				wpk.TID_size:   wpk.TagUint64(uint64(len(pack.bulk))),
			},
			Reader: *bytes.NewReader(pack.bulk),
			Pack:   pack.Package,
		}, nil
	}

	var key = wpk.ToKey(kpath)
	if tags, is := pack.Tags[key]; is {
		return pack.OpenFile(tags)
	} else {
		return pack.OpenDir(kpath)
	}
}

// The End.
