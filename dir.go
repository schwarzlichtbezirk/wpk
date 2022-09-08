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

// PackDirFile is a directory file whose entries can be read with the ReadDir method.
// fs.ReadDirFile interface implementation.
type PackDirFile struct {
	*Tagset_t // has fs.FileInfo interface
	FTT       *FTT_t
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
	return f.FTT.ReadDirN(f.Path(), n)
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
	return path.Join(ToSlash(os.TempDir()), fname)
}

// ToSlash brings filenames to true slashes.
func ToSlash(fpath string) string {
	return strings.ReplaceAll(fpath, "\\", "/")
}

// Normalize brings file path to normalized form. It makes argument lowercase,
// change back slashes to normal slashes. Normalized path is the key to FTT map.
func Normalize(fpath string) string {
	return strings.ToLower(path.Clean(ToSlash(fpath)))
}

// ReadDirN returns fs.DirEntry array with nested into given package directory presentation.
// It's core function for ReadDirFile and ReadDirFS structures.
func (ftt *FTT_t) ReadDirN(dir string, n int) (list []fs.DirEntry, err error) {
	var prefix string
	if dir != "." {
		prefix = Normalize(dir) + "/" // set terminated slash
	}
	var found = map[string]Void{}
	ftt.Enum(func(fkey string, ts *Tagset_t) bool {
		if strings.HasPrefix(fkey, prefix) {
			var suffix = fkey[len(prefix):]
			var sp = strings.IndexByte(suffix, '/')
			if sp < 0 { // file detected
				list = append(list, ts)
				n--
			} else { // dir detected
				var subdir = path.Join(prefix, suffix[:sp])
				if _, ok := found[subdir]; !ok {
					found[subdir] = Void{}
					var fpath = ts.Path() // extract not normalized path
					var dts = MakeTagset(nil, 2, 2).
						Put(TIDpath, StrTag(fpath[:len(subdir)]))
					var f = &PackDirFile{
						Tagset_t: dts,
						FTT:      ftt,
					}
					list = append(list, f)
					n--
				}
			}
		}
		return n != 0
	})
	if n > 0 {
		err = io.EOF
	}
	return
}

// OpenDir returns PackDirFile structure associated with group of files in package
// pooled with common directory prefix. Usable to implement fs.FileSystem interface.
func (ftt *FTT_t) OpenDir(dir string) (fs.ReadDirFile, error) {
	var prefix string
	if dir != "." {
		prefix = Normalize(dir) + "/" // set terminated slash
	}
	var f *PackDirFile
	ftt.Enum(func(fkey string, ts *Tagset_t) bool {
		if strings.HasPrefix(fkey, prefix) {
			var dts = MakeTagset(nil, 2, 2).
				Put(TIDpath, StrTag(dir))
			f = &PackDirFile{
				Tagset_t: dts,
				FTT:      ftt,
			}
			return false
		}
		return true
	})
	if f != nil {
		return f, nil
	}
	// on case if not found
	return nil, &fs.PathError{Op: "open", Path: dir, Err: fs.ErrNotExist}
}

// The End.
