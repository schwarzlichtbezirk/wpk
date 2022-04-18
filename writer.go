package wpk

import (
	"encoding/binary"
	"io"
	"io/fs"
	"os"
)

// Writer is package writer structure.
type Writer struct {
	Package
	LastFID FID_t
}

// Reset initializes fields with zero values and sets prebuild signature.
func (pack *Package) Reset() {
	pack.mux.Lock()
	defer pack.mux.Unlock()

	// reset header
	copy(pack.signature[:], Prebuild)
	pack.fttoffset = HeaderSize
	pack.fttsize = 0
}

// Begin writes prebuild header for new empty package.
func (pack *Writer) Begin(w io.WriteSeeker) (err error) {
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
	// partially reset header
	pack.mux.Lock()
	copy(pack.signature[:], Prebuild)
	pack.mux.Unlock()
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
	// get tags table offset as actual end of file
	var pos int64
	if pos, err = w.Seek(0, io.SeekEnd); err != nil {
		return
	}
	pack.fttoffset = Offset_t(pos)
	// write files tags table
	pack.Enum(func(fkey string, ts *Tagset_t) bool {
		if _, err = w.Write(ts.Data()); err != nil {
			return false
		}
		return true
	})
	if err != nil {
		return
	}
	// write tags table end marker
	if err = binary.Write(w, binary.LittleEndian, TID_t(0)); err != nil {
		return
	}

	// get writer end marker and setup the file tags table size
	if pos, err = w.Seek(0, io.SeekEnd); err != nil {
		return
	}
	pack.fttsize = Size_t(pos) - Size_t(pack.fttoffset)

	// rewrite true header
	if _, err = w.Seek(0, io.SeekStart); err != nil {
		return
	}
	copy(pack.signature[:], Signature)
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

	// get offset and put provided data
	var offset, size int64
	if offset, err = w.Seek(0, io.SeekCurrent); err != nil {
		return
	}
	if size, err = io.Copy(w, r); err != nil {
		return
	}

	// insert new entry to tags table
	pack.LastFID++
	ts = &Tagset_t{}
	ts.Put(TIDfid, TagUint32(uint32(pack.LastFID)))
	ts.Put(TIDoffset, TagUint64(uint64(offset)))
	ts.Put(TIDsize, TagUint64(uint64(size)))
	ts.Put(TIDpath, TagString(ToSlash(kpath)))
	pack.ftt.Store(fkey, ts)

	// update header
	pack.fttoffset = Offset_t(offset + size)
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

// PackDirHook is function called before each file or directory with its parameters
// during PackDir processing. Returns whether to process the file or directory.
// Can be used as filter and logger.
type PackDirHook = func(fi os.FileInfo, kpath, fpath string) bool

// PackDir puts all files of given folder and it's subfolders into package.
// Hook function can be nil.
func (pack *Writer) PackDir(w io.WriteSeeker, dirname, prefix string, hook PackDirHook) (err error) {
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
			if hook == nil || hook(fi, kpath, fpath) {
				if fi.IsDir() {
					if err = pack.PackDir(w, fpath+"/", kpath+"/", hook); err != nil {
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
	pack.ftt.Delete(fkey1)
	pack.ftt.Store(fkey2, ts)
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
	pack.ftt.Store(fkey2, ts2)
	return nil
}

// DelAlias delete tagset with specified file name. Data block is still remains.
func (pack *Writer) DelAlias(name string) bool {
	var fkey = Normalize(name)
	var _, ok = pack.Tagset(fkey)
	if ok {
		pack.ftt.Delete(fkey)
	}
	return ok
}

// The End.
