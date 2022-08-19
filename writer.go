package wpk

import (
	"io"
	"io/fs"
	"os"

	"gopkg.in/djherbis/times.v1"
)

// WriteSeekCloser is typical useful interface for package writing.
type WriteSeekCloser interface {
	io.Writer
	io.Seeker
	io.Closer
}

// Begin writes prebuild header for new empty package.
func (pack *Package) Begin(wpt io.WriteSeeker) (err error) {
	pack.mux.Lock()
	defer pack.mux.Unlock()

	if err = pack.typesize.Checkup(); err != nil {
		return
	}
	if pack.tssize != pack.Header.typesize[PTStssize] {
		return ErrSizeTSSize
	}

	// reset header
	copy(pack.signature[:], Prebuild)
	pack.fttoffset = 0
	pack.fttsize = 0
	pack.datoffset = 0
	pack.datsize = 0
	// go to file start
	if _, err = wpt.Seek(0, io.SeekStart); err != nil {
		return
	}
	// write prebuild header
	if _, err = pack.Header.WriteTo(wpt); err != nil {
		return
	}
	return
}

// Append writes prebuild header for previously opened package to append new files.
func (pack *Package) Append(wpt, wpf io.WriteSeeker) (err error) {
	pack.mux.Lock()
	defer pack.mux.Unlock()

	// partially reset header
	copy(pack.signature[:], Prebuild)
	// go to file start
	if _, err = wpt.Seek(0, io.SeekStart); err != nil {
		return
	}
	// rewrite prebuild header
	if _, err = pack.Header.WriteTo(wpt); err != nil {
		return
	}
	// go to tags table start to replace it by new data
	if wpf != nil { // splitted package files
		if _, err = wpf.Seek(int64(pack.datoffset+pack.datsize), io.SeekStart); err != nil {
			return
		}
	} else { // single package file
		if _, err = wpt.Seek(int64(pack.datoffset+pack.datsize), io.SeekStart); err != nil {
			return
		}
	}
	return
}

// Sync writes actual file tags table and true signature with settings.
func (pack *Package) Sync(wpt, wpf io.WriteSeeker) (err error) {
	pack.mux.Lock()
	defer pack.mux.Unlock()

	var fftpos, fftend, datpos, datend int64

	if wpf != nil { // splitted package files
		// get tags table offset as actual end of file
		datpos = 0
		if datend, err = wpf.Seek(0, io.SeekCurrent); err != nil {
			return
		}
		fftpos = HeaderSize

		// update package info if it has
		if ts, ok := pack.Tagset(""); ok {
			ts.Set(TIDoffset, TagUintLen(uint(datpos), pack.PTS(PTSfoffset)))
			ts.Set(TIDsize, TagUintLen(uint(datend-datpos), pack.PTS(PTSfsize)))
		}

		// write file tags table
		if _, err = wpt.Seek(fftpos, io.SeekStart); err != nil {
			return
		}
		if _, err = pack.FTT_t.WriteTo(wpt); err != nil {
			return
		}
		// get writer end marker and setup the file tags table size
		if fftend, err = wpt.Seek(0, io.SeekCurrent); err != nil {
			return
		}
	} else { // single package file
		// get tags table offset as actual end of file
		datpos = HeaderSize
		if datend, err = wpt.Seek(0, io.SeekCurrent); err != nil {
			return
		}
		fftpos = datend

		// update package info if it has
		if ts, ok := pack.Tagset(""); ok {
			ts.Set(TIDoffset, TagUintLen(uint(datpos), pack.PTS(PTSfoffset)))
			ts.Set(TIDsize, TagUintLen(uint(datend-datpos), pack.PTS(PTSfsize)))
		}

		// write file tags table
		if _, err = pack.FTT_t.WriteTo(wpt); err != nil {
			return
		}
		// get writer end marker and setup the file tags table size
		if fftend, err = wpt.Seek(0, io.SeekCurrent); err != nil {
			return
		}
	}

	// rewrite true header
	if _, err = wpt.Seek(0, io.SeekStart); err != nil {
		return
	}
	copy(pack.signature[:], Signature)
	pack.fttoffset = uint64(fftpos)
	pack.fttsize = uint64(fftend - fftpos)
	pack.datoffset = uint64(datpos)
	pack.datsize = uint64(datend - datpos)
	if _, err = pack.Header.WriteTo(wpt); err != nil {
		return
	}
	return
}

// PackData puts data streamed by given reader into package as a file
// and associate keyname "kpath" with it.
func (pack *Package) PackData(w io.WriteSeeker, r io.Reader, fpath string) (ts *Tagset_t, err error) {
	if _, ok := pack.Tagset(fpath); ok {
		err = &fs.PathError{Op: "packdata", Path: fpath, Err: fs.ErrExist}
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
	}(); err != nil {
		return
	}

	// insert new entry to tags table
	ts = pack.BaseTagset(uint(offset), uint(size), fpath)
	pack.SetTagset(fpath, ts)
	return
}

// PackFile puts file with given file handle into package and associate keyname "kpath" with it.
func (pack *Package) PackFile(w io.WriteSeeker, file *os.File, kpath string) (ts *Tagset_t, err error) {
	var fi os.FileInfo
	if fi, err = file.Stat(); err != nil {
		return
	}
	if ts, err = pack.PackData(w, file, kpath); err != nil {
		return
	}

	//ts.Put(TIDmtime, TagTime(fi.ModTime()))
	var tsp = times.Get(fi)
	ts.Put(TIDmtime, TagTime(tsp.ModTime()))
	ts.Put(TIDatime, TagTime(tsp.AccessTime()))
	if tsp.HasChangeTime() {
		ts.Put(TIDctime, TagTime(tsp.ChangeTime()))
	}
	if tsp.HasBirthTime() {
		ts.Put(TIDbtime, TagTime(tsp.BirthTime()))
	}
	ts.Put(TIDlink, TagString(ToSlash(kpath)))
	return
}

// PackDirLogger is function called during PackDir processing after each
// file with OS file object and inserted tagset, that can be modified.
type PackDirLogger func(r io.ReadSeeker, ts *Tagset_t) error

// PackDir puts all files of given folder and it's subfolders into package.
// Logger function can be nil.
func (pack *Package) PackDir(w io.WriteSeeker, dirname, prefix string, logger PackDirLogger) (err error) {
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
			if fi.IsDir() {
				if err = pack.PackDir(w, fpath+"/", kpath+"/", logger); err != nil {
					return
				}
			} else if func() {
				var file *os.File
				var ts *Tagset_t
				if file, err = os.Open(fpath); err != nil {
					return
				}
				defer file.Close()

				if ts, err = pack.PackFile(w, file, kpath); err != nil {
					return
				}
				if err = logger(file, ts); err != nil {
					return
				}
			}(); err != nil {
				return
			}
		}
	}
	return
}

// Rename tagset with file name 'oldname' to 'newname'.
// Keeps link to original file name.
func (pack *Package) Rename(oldname, newname string) error {
	var ts, ok = pack.Tagset(oldname)
	if !ok {
		return &fs.PathError{Op: "rename", Path: oldname, Err: fs.ErrNotExist}
	}
	if _, ok = pack.Tagset(newname); ok {
		return &fs.PathError{Op: "rename", Path: newname, Err: fs.ErrExist}
	}

	ts.Set(TIDpath, TagString(ToSlash(newname)))
	pack.DelTagset(oldname)
	pack.SetTagset(newname, ts)
	return nil
}

// PutAlias makes clone tagset with file name 'oldname' and replace name tag
// in it to 'newname'. Keeps link to original file name.
func (pack *Package) PutAlias(oldname, newname string) error {
	var ts1, ok = pack.Tagset(oldname)
	if !ok {
		return &fs.PathError{Op: "putalias", Path: oldname, Err: fs.ErrNotExist}
	}
	if _, ok = pack.Tagset(newname); ok {
		return &fs.PathError{Op: "putalias", Path: newname, Err: fs.ErrExist}
	}

	var tsi = ts1.Iterator()
	var ts2 = pack.NewTagset()
	for tsi.Next() {
		if tsi.tid != TIDpath {
			ts2.Put(tsi.tid, tsi.Tag())
		} else {
			ts2.Put(TIDpath, TagString(ToSlash(newname)))
		}
	}
	pack.SetTagset(newname, ts2)
	return nil
}

// The End.
