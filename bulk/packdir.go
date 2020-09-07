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

// Reads WPK-file package by given file name.
func (pack *PackDir) ReadWPK(fname string) (err error) {
	var bulk []byte
	if bulk, err = ioutil.ReadFile(fname); err != nil {
		return
	}

	pack.Package = &wpk.Package{}
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
func (pack *PackDir) SubDir(pref string) *PackDir {
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
func (pack *PackDir) NewFile(tags wpk.Tagset) (http.File, error) {
	var offset, size = tags.Record()
	return &wpk.File{
		Tagset: tags,
		Reader: *bytes.NewReader(pack.bulk[offset : offset+size]),
		Pack:   pack.Package,
	}, nil
}

// Returns slice with nested into package file content.
func (pack *PackDir) Extract(tags wpk.Tagset) ([]byte, error) {
	var offset, size = tags.Record()
	return pack.bulk[offset : offset+size], nil
}

// Implements access to nested into package file or directory by keyname.
func (pack *PackDir) Open(kpath string) (http.File, error) {
	var key = pack.pref + strings.TrimPrefix(wpk.ToKey(kpath), "/")
	if key == "" {
		return pack.NewDir(key), nil
	} else if key == "wpk" {
		var tags = wpk.Tagset{
			wpk.TID_FID:    wpk.TagUint32(0),
			wpk.TID_size:   wpk.TagUint64(uint64(len(pack.bulk))),
			wpk.TID_offset: wpk.TagUint64(0),
		}
		return pack.NewFile(tags)
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
