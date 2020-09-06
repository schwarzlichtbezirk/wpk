package bulk

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/schwarzlichtbezirk/wpk"
)

type PackDir struct {
	*wpk.Package
	bulk []byte
	pref string
}

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

func (pack *PackDir) Close() error {
	return nil
}

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

func (pack *PackDir) NewFile(tags wpk.Tagset) http.File {
	var offset, size = tags.Record()
	return &wpk.File{
		Tagset: tags,
		Reader: *bytes.NewReader(pack.bulk[offset : offset+size]),
		Pack:   pack.Package,
	}
}

func (pack *PackDir) Extract(tags wpk.Tagset) []byte {
	var offset, size = tags.Record()
	return pack.bulk[offset : offset+size]
}

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
		return pack.NewFile(tags), nil
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
