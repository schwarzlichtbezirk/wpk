package wpk

import (
	"io"
	"io/fs"
	"os"
	"sync"
	"sync/atomic"
)

// Writer is package writer structure.
type Writer struct {
	Package
	LastFID uint32
	mux     sync.Mutex
}

// Begin writes prebuild header for new empty package.
func (pack *Writer) Begin(w io.WriteSeeker) (err error) {
	pack.mux.Lock()
	defer pack.mux.Unlock()

	// reset header
	pack.Reset()
	// go to file start
	if _, err = w.Seek(0, io.SeekStart); err != nil {
		return
	}
	// write prebuild header
	if _, err = pack.Header.WriteTo(w); err != nil {
		return
	}
	return
}

// Append writes prebuild header for previously opened package to append new files.
func (pack *Writer) Append(w io.WriteSeeker) (err error) {
	pack.mux.Lock()
	defer pack.mux.Unlock()

	// partially reset header
	copy(pack.signature[:], Prebuild)
	// go to file start
	if _, err = w.Seek(0, io.SeekStart); err != nil {
		return
	}
	// rewrite prebuild header
	if _, err = pack.Header.WriteTo(w); err != nil {
		return
	}
	// go to tags table start to replace it by new data
	if _, err = w.Seek(int64(pack.fttoffset), io.SeekStart); err != nil {
		return
	}
	return
}

// Finalize finalizes package writing. Writes true signature and header settings.
func (pack *Writer) Finalize(w io.WriteSeeker) (err error) {
	pack.mux.Lock()
	defer pack.mux.Unlock()

	// get tags table offset as actual end of file
	var pos1, pos2 int64
	if pos1, err = w.Seek(0, io.SeekEnd); err != nil {
		return
	}
	// update package info if it has
	if ts, ok := pack.Tagset(""); ok {
		ts.Set(TIDoffset, TagFOffset(HeaderSize))
		ts.Set(TIDsize, TagFSize(FSize_t(pos1-HeaderSize)))
	}
	// write file tags table
	if _, err = pack.FTT_t.WriteTo(w); err != nil {
		return
	}
	// get writer end marker and setup the file tags table size
	if pos2, err = w.Seek(0, io.SeekEnd); err != nil {
		return
	}

	// rewrite true header
	if _, err = w.Seek(0, io.SeekStart); err != nil {
		return
	}
	copy(pack.signature[:], Signature)
	pack.fttoffset = uint64(pos1)
	pack.fttsize = uint64(pos2 - pos1)
	if _, err = pack.Header.WriteTo(w); err != nil {
		return
	}
	return
}

// PackData puts data streamed by given reader into package as a file
// and associate keyname "kpath" with it.
func (pack *Writer) PackData(w io.WriteSeeker, r io.Reader, kpath string) (ts *Tagset_t, err error) {
	var fkey = Normalize(kpath)
	if _, ok := pack.Tagset(fkey); ok {
		err = &fs.PathError{Op: "packdata", Path: kpath, Err: fs.ErrExist}
		return
	}

	var offset, size int64
	if func() {
		pack.mux.Lock()
		defer pack.mux.Unlock()

		// get offset and put provided data
		if offset, err = w.Seek(0, io.SeekCurrent); err != nil {
			return
		}
		if size, err = io.Copy(w, r); err != nil {
			return
		}
		// update header
		pack.fttoffset = uint64(offset + size)
	}(); err != nil {
		return
	}

	// insert new entry to tags table
	var fid = FID_t(atomic.AddUint32(&pack.LastFID, 1))
	ts = NewTagset().
		Put(TIDoffset, TagFOffset(FOffset_t(offset))).
		Put(TIDsize, TagFSize(FSize_t(size))).
		Put(TIDfid, TagFID(fid)).
		Put(TIDpath, TagString(ToSlash(kpath)))
	pack.SetTagset(fkey, ts)
	return
}

// PackFile puts file with given file handle into package and associate keyname "kpath" with it.
func (pack *Writer) PackFile(w io.WriteSeeker, file *os.File, kpath string) (ts *Tagset_t, err error) {
	var fi os.FileInfo
	if fi, err = file.Stat(); err != nil {
		return
	}
	if ts, err = pack.PackData(w, file, kpath); err != nil {
		return
	}

	ts.Put(TIDcreated, TagUint64(uint64(fi.ModTime().Unix())))
	ts.Put(TIDlink, TagString(ToSlash(kpath)))
	return
}

// PackDirFilter is function called before each file or directory with its parameters
// during PackDir processing. Returns whether to process the file or directory.
// Can be used as filter and logger.
type PackDirFilter = func(fi os.FileInfo, kpath, fpath string) bool

// PackDir puts all files of given folder and it's subfolders into package.
// Filter function can be nil.
func (pack *Writer) PackDir(w io.WriteSeeker, dirname, prefix string, filter PackDirFilter) (err error) {
	var fis []os.FileInfo
	if func() {
		var dir *os.File
		if dir, err = os.Open(dirname); err != nil {
			return
		}
		defer dir.Close()

		if fis, err = dir.Readdir(-1); err != nil {
			return
		}
	}(); err != nil {
		return
	}
	for _, fi := range fis {
		if fi != nil {
			var kpath = prefix + fi.Name()
			var fpath = dirname + fi.Name()
			if filter == nil || filter(fi, kpath, fpath) {
				if fi.IsDir() {
					if err = pack.PackDir(w, fpath+"/", kpath+"/", filter); err != nil {
						return
					}
				} else if func() {
					var file *os.File
					if file, err = os.Open(fpath); err != nil {
						return
					}
					defer file.Close()

					if _, err = pack.PackFile(w, file, kpath); err != nil {
						return
					}
				}(); err != nil {
					return
				}
			}
		}
	}
	return
}

// Rename tagset with file name 'oldname' to 'newname'.
// Keeps link to original file name.
func (pack *Writer) Rename(oldname, newname string) error {
	var fkey1 = Normalize(oldname)
	var fkey2 = Normalize(newname)
	var ts, ok = pack.Tagset(fkey1)
	if !ok {
		return &fs.PathError{Op: "rename", Path: oldname, Err: fs.ErrNotExist}
	}
	if _, ok = pack.Tagset(fkey2); ok {
		return &fs.PathError{Op: "rename", Path: newname, Err: fs.ErrExist}
	}

	ts.Set(TIDpath, TagString(ToSlash(newname)))
	pack.DelTagset(fkey1)
	pack.SetTagset(fkey2, ts)
	return nil
}

// PutAlias makes clone tagset with file name 'oldname' and replace name tag
// in it to 'newname'. Keeps link to original file name.
func (pack *Writer) PutAlias(oldname, newname string) error {
	var fkey1 = Normalize(oldname)
	var fkey2 = Normalize(newname)
	var ts1, ok = pack.Tagset(fkey1)
	if !ok {
		return &fs.PathError{Op: "putalias", Path: oldname, Err: fs.ErrNotExist}
	}
	if _, ok = pack.Tagset(fkey2); ok {
		return &fs.PathError{Op: "putalias", Path: newname, Err: fs.ErrExist}
	}

	var tsi = ts1.Iterator()
	var ts2 = &Tagset_t{}
	for tsi.Next() {
		if tsi.tid != TIDpath {
			ts2.Put(tsi.tid, tsi.Tag())
		} else {
			ts2.Put(TIDpath, TagString(ToSlash(newname)))
		}
	}
	pack.SetTagset(fkey2, ts2)
	return nil
}

// DelAlias delete tagset with specified file name. Data block is still remains.
func (pack *Writer) DelAlias(name string) bool {
	var fkey = Normalize(name)
	var _, ok = pack.Tagset(fkey)
	if ok {
		pack.DelTagset(fkey)
	}
	return ok
}

// The End.
