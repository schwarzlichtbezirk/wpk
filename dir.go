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
	*TagsetRaw // has fs.FileInfo interface
	ftt        *FTT
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
	return path.Join(ToSlash(os.TempDir()), fname)
}

// ToSlash brings filenames to true slashes
// without superfluous allocations if it possible.
func ToSlash(s string) string {
	var b = S2B(s)
	var bc = b
	var c bool
	for i, v := range b {
		if v == '\\' {
			if !c {
				bc, c = []byte(s), true
			}
			bc[i] = '/'
		}
	}
	return B2S(bc)
}

// Normalize brings file path to normalized form. Normalized path is the key to FTT map.
var Normalize = ToSlash

// ReadDirN returns fs.DirEntry array with nested into given package directory presentation.
// It's core function for ReadDirFile and ReadDirFS structures.
func (ftt *FTT) ReadDirN(fulldir string, n int) (list []fs.DirEntry, err error) {
	var found = map[string]fs.DirEntry{}
	var prefix string
	if fulldir != "." && fulldir != "" {
		prefix = Normalize(fulldir) + "/" // set terminated slash
	}

	ftt.rwm.Range(func(fkey string, ts *TagsetRaw) bool {
		if strings.HasPrefix(fkey, prefix) {
			var suffix = fkey[len(prefix):]
			var sp = strings.IndexByte(suffix, '/')
			if sp < 0 { // file detected
				found[suffix] = ts
				n--
			} else { // dir detected
				var subdir = path.Join(prefix, suffix[:sp])
				if _, ok := found[subdir]; !ok {
					var dts = MakeTagset(nil, 2, 2).
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
	var prefix string
	if fulldir != "." && fulldir != "" {
		prefix = Normalize(fulldir) + "/" // set terminated slash
	}
	var f *PackDirFile
	ftt.rwm.Range(func(fkey string, ts *TagsetRaw) bool {
		if strings.HasPrefix(fkey, prefix) {
			var dts = MakeTagset(nil, 2, 2).
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
