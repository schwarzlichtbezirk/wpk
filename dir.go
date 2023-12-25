package wpk

import (
	"errors"
	"io"
	"io/fs"
	"os"
	"path"
	"strings"
)

// MakeTagsPath receives file path and returns it with ".wpt" extension.
// It hepls to open splitted package.
func MakeTagsPath(fpath string) string {
	var ext = path.Ext(fpath)
	return fpath[:len(fpath)-len(ext)] + ".wpt"
}

// MakeDataPath receives file path and returns it with ".wpf" extension.
// It hepls to open splitted package.
func MakeDataPath(fpath string) string {
	var ext = path.Ext(fpath)
	return fpath[:len(fpath)-len(ext)] + ".wpf"
}

// PackDirFile is a directory file whose entries can be read with the ReadDir method.
// fs.ReadDirFile interface implementation.
type PackDirFile struct {
	TagsetRaw // has fs.FileInfo interface
	ftt       *FTT
}

// fs.ReadDirFile interface implementation.
func (f *PackDirFile) Stat() (fs.FileInfo, error) {
	return f, nil
}

// fs.ReadDirFile interface implementation.
func (f *PackDirFile) Read(b []byte) (n int, err error) {
	return 0, io.EOF
}

// fs.ReadDirFile interface implementation.
func (f *PackDirFile) Close() error {
	return nil
}

// fs.ReadDirFile interface implementation.
func (f *PackDirFile) ReadDir(n int) (matches []fs.DirEntry, err error) {
	return f.ftt.ReadDirN(f.Path(), n)
}

// DirExists check up directory existence.
func DirExists(fpath string) (bool, error) {
	var stat, err = os.Stat(fpath)
	if err == nil {
		return stat.IsDir(), nil
	}
	if errors.Is(err, fs.ErrNotExist) {
		return false, nil
	}
	return false, err
}

// FileExists check up file existence.
func FileExists(fpath string) (bool, error) {
	var stat, err = os.Stat(fpath)
	if err == nil {
		return !stat.IsDir(), nil
	}
	if errors.Is(err, fs.ErrNotExist) {
		return false, nil
	}
	return false, err
}

// TempPath returns filename located at temporary directory.
func TempPath(fname string) string {
	return JoinPath(ToSlash(os.TempDir()), fname)
}

// ReadDirN returns fs.DirEntry array with nested into given package directory presentation.
// It's core function for ReadDirFile and ReadDirFS structures.
func (ftt *FTT) ReadDirN(fulldir string, n int) (list []fs.DirEntry, err error) {
	fulldir = ToSlash(fulldir)
	var found = map[string]fs.DirEntry{}
	var prefix string
	if fulldir != "." && fulldir != "" {
		prefix = fulldir + "/" // set terminated slash
	}

	ftt.tsm.Range(func(fkey string, ts TagsetRaw) bool {
		if strings.HasPrefix(fkey, prefix) {
			var suffix = fkey[len(prefix):]
			var sp = strings.IndexByte(suffix, '/')
			if sp < 0 { // file detected
				found[suffix] = ts
				n--
			} else { // dir detected
				var subdir = JoinPath(prefix, suffix[:sp])
				if _, ok := found[subdir]; !ok {
					var dts = TagsetRaw{}.
						Put(TIDpath, StrTag(subdir))
					var f = &PackDirFile{
						TagsetRaw: dts,
						ftt:       ftt,
					}
					found[subdir] = f
					n--
				}
			}
		}
		return n != 0
	})

	list = make([]fs.DirEntry, len(found))
	var i int
	for _, de := range found {
		list[i] = de
		i++
	}
	if n > 0 {
		err = io.EOF
	}
	return
}

// OpenDir returns PackDirFile structure associated with group of files in package
// pooled with common directory prefix. Usable to implement fs.FileSystem interface.
func (ftt *FTT) OpenDir(fulldir string) (fs.ReadDirFile, error) {
	fulldir = ToSlash(fulldir)
	var prefix string
	if fulldir != "." && fulldir != "" {
		prefix = fulldir + "/" // set terminated slash
	}
	var f *PackDirFile
	ftt.tsm.Range(func(fkey string, ts TagsetRaw) bool {
		if strings.HasPrefix(fkey, prefix) {
			var dts = TagsetRaw{}.
				Put(TIDpath, StrTag(fulldir))
			f = &PackDirFile{
				TagsetRaw: dts,
				ftt:       ftt,
			}
			return false
		}
		return true
	})
	if f != nil {
		return f, nil
	}
	// on case if not found
	return nil, &fs.PathError{Op: "open", Path: fulldir, Err: fs.ErrNotExist}
}

// The End.
