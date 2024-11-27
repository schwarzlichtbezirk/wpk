package wpk

import (
	"io"
	"io/fs"
	"os"
	"strings"

	"gopkg.in/djherbis/times.v1"

	"github.com/schwarzlichtbezirk/wpk/util"
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
		signature: [SignSize]byte(util.S2B(SignBuild)),
		fttcount:  0,
		fttoffset: offset,
		fttsize:   0,
		datoffset: offset,
		datsize:   0,
	}
	if _, err = wpt.Seek(0, io.SeekStart); err != nil {
		return
	}
	if _, err = hdr.WriteTo(wpt); err != nil {
		return
	}
	// update data offset/pos
	ftt.datoffset, ftt.datsize = hdr.datoffset, hdr.datsize
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
	if _, err = wpt.Write(util.S2B(SignBuild)); err != nil {
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
		signature: [SignSize]byte(util.S2B(SignReady)),
		fttcount:  uint64(ftt.tsm.Len()),
		fttoffset: uint64(fftpos),
		fttsize:   uint64(fftend - fftpos),
		datoffset: uint64(datpos),
		datsize:   uint64(datend - datpos),
	}
	if _, err = wpt.Seek(0, io.SeekStart); err != nil {
		return
	}
	if _, err = hdr.WriteTo(wpt); err != nil {
		return
	}
	// update data offset/pos
	ftt.datoffset, ftt.datsize = hdr.datoffset, hdr.datsize
	return
}

// PackData puts data streamed by given reader into package as a file
// and associate keyname "fkey" with it.
func (pkg *Package) PackData(w io.WriteSeeker, r io.Reader, fkey string) (ts TagsetRaw, err error) {
	if _, ok := pkg.GetTagset(fkey); ok {
		err = &fs.PathError{Op: "packdata", Path: fkey, Err: fs.ErrExist}
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
	ts = pkg.BaseTagset(uint(offset), uint(size), fkey)
	pkg.SetTagset(fkey, ts)
	return
}

// PackFile puts file with given file handle into package and associate keyname "fkey" with it.
func (pkg *Package) PackFile(w io.WriteSeeker, file fs.File, fkey string) (ts TagsetRaw, err error) {
	var fi os.FileInfo
	if fi, err = file.Stat(); err != nil {
		return
	}
	if ts, err = pkg.PackData(w, file, fkey); err != nil {
		return
	}

	//ts = ts.Put(TIDmtime, TimeTag(fi.ModTime()))
	var tsp = times.Get(fi)
	ts = ts.Put(TIDmtime, TimeTag(tsp.ModTime()))
	ts = ts.Put(TIDatime, TimeTag(tsp.AccessTime()))
	if tsp.HasChangeTime() {
		ts = ts.Put(TIDctime, TimeTag(tsp.ChangeTime()))
	}
	if tsp.HasBirthTime() {
		ts = ts.Put(TIDbtime, TimeTag(tsp.BirthTime()))
	}
	pkg.SetTagset(fkey, ts)
	return
}

// Rename tagset with file name 'fkey1' to 'fkey2'.
// Keeps link to original file name.
func (pkg *Package) Rename(fkey1, fkey2 string) error {
	var ts, ok = pkg.GetTagset(fkey1)
	if !ok {
		return &fs.PathError{Op: "rename", Path: fkey1, Err: fs.ErrNotExist}
	}
	if pkg.HasTagset(fkey2) {
		return &fs.PathError{Op: "rename", Path: fkey2, Err: fs.ErrExist}
	}

	ts = CopyTagset(ts).Set(TIDpath, StrTag(pkg.FullPath(util.ToSlash(fkey2))))
	pkg.DelTagset(fkey1)
	pkg.SetTagset(fkey2, ts)
	return nil
}

// RenameDir renames all files in package with 'olddir' path to 'newdir' path.
func (pkg *Package) RenameDir(olddir, newdir string, skipexist bool) (count int, err error) {
	if len(olddir) > 0 && olddir[len(olddir)-1] != '/' {
		olddir += "/"
	}
	if len(newdir) > 0 && newdir[len(newdir)-1] != '/' {
		newdir += "/"
	}
	pkg.Enum(func(fkey string, ts TagsetRaw) bool {
		if strings.HasPrefix(fkey, olddir) {
			var newkey = newdir + fkey[len(olddir):]
			if _, ok := pkg.GetTagset(newkey); ok {
				err = &fs.PathError{Op: "renamedir", Path: newkey, Err: fs.ErrExist}
				return skipexist
			}
			ts = CopyTagset(ts).Set(TIDpath, StrTag(pkg.FullPath(util.ToSlash(newkey))))
			pkg.DelTagset(fkey)
			pkg.SetTagset(newkey, ts)
			count++
		}
		return true
	})
	if skipexist {
		err = nil
	}
	return
}

// PutAlias makes clone tagset with file name 'fkey1' and replace name tag
// in it to 'fkey2'. Keeps link to original file name.
func (pkg *Package) PutAlias(fkey1, fkey2 string) error {
	var ts, ok = pkg.GetTagset(fkey1)
	if !ok {
		return &fs.PathError{Op: "putalias", Path: fkey1, Err: fs.ErrNotExist}
	}
	if _, ok = pkg.GetTagset(fkey2); ok {
		return &fs.PathError{Op: "putalias", Path: fkey2, Err: fs.ErrExist}
	}

	ts = CopyTagset(ts).Set(TIDpath, StrTag(pkg.FullPath(util.ToSlash(fkey2))))
	pkg.SetTagset(fkey2, ts)
	return nil
}

// The End.
