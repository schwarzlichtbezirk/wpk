package wpk

import (
	"errors"
	"io"
	"io/fs"
	"os"
	"path"
	"regexp"
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

// JoinFast performs fast join of two path chunks.
func JoinFast(dir, base string) string {
	if dir == "" || dir == "." {
		return base
	}
	if base == "" || base == "." {
		return dir
	}
	if dir[len(dir)-1] == '/' {
		if base[0] == '/' {
			return dir + base[1:]
		} else {
			return dir + base
		}
	}
	if base[0] == '/' {
		return dir + base
	}
	return dir + "/" + base
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

var (
	evlre = regexp.MustCompile(`\$\w+`)     // env var with linux-like syntax
	evure = regexp.MustCompile(`\$\{\w+\}`) // env var with unix-like syntax
	evwre = regexp.MustCompile(`\%\w+\%`)   // env var with windows-like syntax
)

// Envfmt replaces environment variables entries in file path to there values.
// Environment variables must be enclosed as ${...} in string.
func Envfmt(p string) string {
	return evwre.ReplaceAllStringFunc(evure.ReplaceAllStringFunc(evlre.ReplaceAllStringFunc(p, func(name string) string {
		// strip $VAR and replace by environment value
		if val, ok := os.LookupEnv(name[1:]); ok {
			return val
		} else {
			return name
		}
	}), func(name string) string {
		// strip ${VAR} and replace by environment value
		if val, ok := os.LookupEnv(name[2 : len(name)-1]); ok {
			return val
		} else {
			return name
		}
	}), func(name string) string {
		// strip %VAR% and replace by environment value
		if val, ok := os.LookupEnv(name[1 : len(name)-1]); ok {
			return val
		} else {
			return name
		}
	})
}

// PathExists check up file or directory existence.
func PathExists(path string) (bool, error) {
	var err error
	if _, err = os.Stat(path); err == nil {
		return true, nil
	}
	if errors.Is(err, fs.ErrNotExist) {
		return false, nil
	}
	return true, err
}

// TempPath returns filename located at temporary directory.
func TempPath(fname string) string {
	return JoinFast(ToSlash(os.TempDir()), fname)
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
				var subdir = JoinFast(prefix, suffix[:sp])
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
