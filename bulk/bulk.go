package bulk

import (
	"bytes"
	"io/fs"
	"os"

	"github.com/schwarzlichtbezirk/wpk"
)

// SliceFile structure gives access to nested into package file.
// wpk.NestedFile interface implementation.
type SliceFile struct {
	wpk.FileReader
	tags *wpk.TagsetRaw // has fs.FileInfo interface
}

// NewSliceFile creates SliceFile file structure based on given tags slice.
func NewSliceFile(tgr *Tagger, ts *wpk.TagsetRaw) (f *SliceFile, err error) {
	var offset, size = ts.Pos()
	f = &SliceFile{
		FileReader: bytes.NewReader(tgr.bulk[offset : offset+size]),
		tags:       ts,
	}
	return
}

// Stat is for fs.File interface compatibility.
func (f *SliceFile) Stat() (fs.FileInfo, error) {
	return f.tags, nil
}

// Close is for fs.File interface compatibility.
func (f *SliceFile) Close() error {
	return nil
}

// Tagger is object to get access to package nested files
// by reading sections of bytes slice.
type Tagger struct {
	bulk []byte // slice with whole package content
}

// MakeTagger creates Tagger object to get access to package nested files.
func MakeTagger(ftt *wpk.FTT, fpath string) (wpk.Tagger, error) {
	var err error
	var tgr Tagger
	var dpath string
	if ftt.IsSplitted() {
		dpath = wpk.MakeDataPath(fpath)
	} else {
		dpath = fpath
	}
	if tgr.bulk, err = os.ReadFile(dpath); err != nil {
		return nil, err
	}
	return &tgr, nil
}

// OpenTagset creates file object to give access to nested into package file by given tagset.
func (tgr *Tagger) OpenTagset(ts *wpk.TagsetRaw) (wpk.NestedFile, error) {
	return NewSliceFile(tgr, ts)
}

// Close does nothing, there is no any opened handles.
// Useful for interface compatibility.
// io.Closer implementation.
func (tgr *Tagger) Close() error {
	return nil
}

// The End.
