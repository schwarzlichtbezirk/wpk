package wpk

import (
	"encoding/binary"
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
func (ftt *FTT) Begin(wpt, wpf io.WriteSeeker) (err error) {
	ftt.mux.Lock()
	defer ftt.mux.Unlock()

	// write prebuild header
	var offset uint64
	if wpf == nil || wpf == wpt {
		offset = HeaderSize
	}
	var hdr = Header{
		typesize:  TypeSize{ftt.tidsz, ftt.tagsz, ftt.tssize},
		fttoffset: offset,
		fttsize:   0,
		datoffset: offset,
		datsize:   0,
	}
	copy(hdr.signature[:], SignBuild)
	if _, err = wpt.Seek(0, io.SeekStart); err != nil {
		return
	}
	if _, err = hdr.WriteTo(wpt); err != nil {
		return
	}
	// update data offset/pos
	ftt.datoffset, ftt.datsize = offset, 0
	return
}

// Append writes prebuild header for previously opened package to append new files.
func (ftt *FTT) Append(wpt, wpf io.WriteSeeker) (err error) {
	ftt.mux.Lock()
	defer ftt.mux.Unlock()

	// go to file start
	if _, err = wpt.Seek(0, io.SeekStart); err != nil {
		return
	}
	// rewrite prebuild signature
	if err = binary.Write(wpt, binary.LittleEndian, []byte(SignBuild)); err != nil {
		return
	}
	// go to tags table start to replace it by new data
	if wpf != nil && wpf != wpt { // splitted package files
		if _, err = wpf.Seek(int64(ftt.datoffset+ftt.datsize), io.SeekStart); err != nil {
			return
		}
	} else { // single package file
		if _, err = wpt.Seek(int64(ftt.datoffset+ftt.datsize), io.SeekStart); err != nil {
			return
		}
	}
	return
}

// Sync writes actual file tags table and true signature with settings.
func (ftt *FTT) Sync(wpt, wpf io.WriteSeeker) (err error) {
	ftt.mux.Lock()
	defer ftt.mux.Unlock()

	var fftpos, fftend, datpos, datend int64

	if wpf != nil && wpf != wpt { // splitted package files
		// get tags table offset as actual end of file
		datpos = 0
		if datend, err = wpf.Seek(0, io.SeekCurrent); err != nil {
			return
		}
		fftpos = HeaderSize

		// update package info if it has
		if ts, ok := ftt.Info(); ok {
			ts.Set(TIDoffset, UintTag(Uint(datpos)))
			ts.Set(TIDsize, UintTag(Uint(datend-datpos)))
		}

		// write file tags table
		if _, err = wpt.Seek(fftpos, io.SeekStart); err != nil {
			return
		}
		if _, err = ftt.WriteTo(wpt); err != nil {
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
		if ts, ok := ftt.Info(); ok {
			ts.Set(TIDoffset, UintTag(Uint(datpos)))
			ts.Set(TIDsize, UintTag(Uint(datend-datpos)))
		}

		// write file tags table
		if _, err = ftt.WriteTo(wpt); err != nil {
			return
		}
		// get writer end marker and setup the file tags table size
		if fftend, err = wpt.Seek(0, io.SeekCurrent); err != nil {
			return
		}
	}

	// rewrite true header
	var hdr = Header{
		typesize:  TypeSize{ftt.tidsz, ftt.tagsz, ftt.tssize},
		fttoffset: uint64(fftpos),
		fttsize:   uint64(fftend - fftpos),
		datoffset: uint64(datpos),
		datsize:   uint64(datend - datpos),
	}
	copy(hdr.signature[:], SignReady)
	if _, err = wpt.Seek(0, io.SeekStart); err != nil {
		return
	}
	if _, err = hdr.WriteTo(wpt); err != nil {
		return
	}
	// update data offset/pos
	ftt.datoffset, ftt.datsize = uint64(datpos), uint64(datend-datpos)
	return
}

// PackData puts data streamed by given reader into package as a file
// and associate keyname "kpath" with it.
func (pkg *Package) PackData(w io.WriteSeeker, r io.Reader, fpath string) (ts *TagsetRaw, err error) {
	if _, ok := pkg.Tagset(fpath); ok {
		err = &fs.PathError{Op: "packdata", Path: fpath, Err: fs.ErrExist}
		return
	}

	var offset, size int64
	if func() {
		pkg.mux.Lock()
		defer pkg.mux.Unlock()

		// get offset and put provided data
		if offset, err = w.Seek(0, io.SeekCurrent); err != nil {
			return
		}
		if size, err = io.Copy(w, r); err != nil {
			return
		}
		// update actual package data size
		pkg.datsize += uint64(size)
	}(); err != nil {
		return
	}

	// insert new entry to tags table
	ts = pkg.BaseTagset(Uint(offset), Uint(size), fpath)
	pkg.SetTagset(fpath, ts)
	return
}

// PackFile puts file with given file handle into package and associate keyname "fpath" with it.
func (pkg *Package) PackFile(w io.WriteSeeker, file fs.File, fpath string) (ts *TagsetRaw, err error) {
	var fi os.FileInfo
	if fi, err = file.Stat(); err != nil {
		return
	}
	if ts, err = pkg.PackData(w, file, fpath); err != nil {
		return
	}

	//ts.Put(TIDmtime, TimeTag(fi.ModTime()))
	var tsp = times.Get(fi)
	ts.Put(TIDmtime, TimeTag(tsp.ModTime()))
	ts.Put(TIDatime, TimeTag(tsp.AccessTime()))
	if tsp.HasChangeTime() {
		ts.Put(TIDctime, TimeTag(tsp.ChangeTime()))
	}
	if tsp.HasBirthTime() {
		ts.Put(TIDbtime, TimeTag(tsp.BirthTime()))
	}
	ts.Put(TIDlink, StrTag(ToSlash(fpath)))
	return
}

// PackDirLogger is function called during PackDir processing after each
// file with OS file object and inserted tagset, that can be modified.
type PackDirLogger func(r io.ReadSeeker, ts *TagsetRaw) error

// PackDir puts all files of given folder and it's subfolders into package.
// Logger function can be nil.
func (pkg *Package) PackDir(w io.WriteSeeker, dirname, prefix string, logger PackDirLogger) (err error) {
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
				if err = pkg.PackDir(w, fpath+"/", kpath+"/", logger); err != nil {
					return
				}
			} else if func() {
				var file *os.File
				var ts *TagsetRaw
				if file, err = os.Open(fpath); err != nil {
					return
				}
				defer file.Close()

				if ts, err = pkg.PackFile(w, file, kpath); err != nil {
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
func (pkg *Package) Rename(oldname, newname string) error {
	var ts, ok = pkg.Tagset(oldname)
	if !ok {
		return &fs.PathError{Op: "rename", Path: oldname, Err: fs.ErrNotExist}
	}
	if _, ok = pkg.Tagset(newname); ok {
		return &fs.PathError{Op: "rename", Path: newname, Err: fs.ErrExist}
	}

	ts.Set(TIDpath, StrTag(ToSlash(pkg.FullPath(newname))))
	pkg.DelTagset(oldname)
	pkg.SetTagset(newname, ts)
	return nil
}

// PutAlias makes clone tagset with file name 'oldname' and replace name tag
// in it to 'newname'. Keeps link to original file name.
func (pkg *Package) PutAlias(oldname, newname string) error {
	var ts1, ok = pkg.Tagset(oldname)
	if !ok {
		return &fs.PathError{Op: "putalias", Path: oldname, Err: fs.ErrNotExist}
	}
	if _, ok = pkg.Tagset(newname); ok {
		return &fs.PathError{Op: "putalias", Path: newname, Err: fs.ErrExist}
	}

	var tsi = ts1.Iterator()
	var ts2 = pkg.NewTagset()
	for tsi.Next() {
		if tsi.tid != TIDpath {
			ts2.Put(tsi.tid, tsi.Tag())
		} else {
			ts2.Put(TIDpath, StrTag(ToSlash(pkg.FullPath(newname))))
		}
	}
	pkg.SetTagset(newname, ts2)
	return nil
}

// The End.
