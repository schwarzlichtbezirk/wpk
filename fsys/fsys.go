package fsys

import (
	"io"
	"io/fs"
	"os"

	"github.com/schwarzlichtbezirk/wpk"
)

type ReaserAtCloser interface {
	io.ReaderAt
	io.Closer
}

// ChunkFile structure gives access to nested into package file.
// wpk.NestedFile interface implementation.
type ChunkFile struct {
	wpk.FileReader
	wpkf io.Closer
	tags *wpk.TagsetRaw // has fs.FileInfo interface
}

// NewChunkFile creates ChunkFile file structure based on given tags slice.
func NewChunkFile(fpath string, ts *wpk.TagsetRaw) (f *ChunkFile, err error) {
	var wpkf *os.File
	if wpkf, err = os.Open(fpath); err != nil {
		return
	}
	var offset, size = ts.Pos()
	f = &ChunkFile{
		FileReader: io.NewSectionReader(wpkf, int64(offset), int64(size)),
		wpkf:       wpkf,
		tags:       ts,
	}
	return
}

// Stat is for fs.File interface compatibility.
func (f *ChunkFile) Stat() (fs.FileInfo, error) {
	return f.tags, nil
}

// Close closes associated wpk-file handle.
func (f *ChunkFile) Close() error {
	return f.wpkf.Close()
}

// Tagger is object to get access to package nested files
// by sections of wpk-file reading.
type Tagger struct {
	dpath string // package filename
}

// MakeTagger creates Tagger object to get access to package nested files.
func MakeTagger(pack *wpk.Package, fpath string) (wpk.Tagger, error) {
	var tgr Tagger
	if pack.IsSplitted() {
		tgr.dpath = wpk.MakeDataPath(fpath)
	} else {
		tgr.dpath = fpath
	}
	return &tgr, nil
}

// OpenTagset creates file object to give access to nested into package file by given tagset.
func (tgr *Tagger) OpenTagset(ts *wpk.TagsetRaw) (wpk.NestedFile, error) {
	return NewChunkFile(tgr.dpath, ts)
}

// Close file handle. This function must be called only for root object,
// not subdirectories.
// io.Closer implementation.
func (tgr *Tagger) Close() error {
	return nil
}

// The End.
