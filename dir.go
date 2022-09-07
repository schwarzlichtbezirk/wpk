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

// ReadDirFile is a directory file whose entries can be read with the ReadDir method.
// fs.ReadDirFile interface implementation.
type ReadDirFile struct {
	*Tagset_t // has fs.FileInfo interface
	FTT       *FTT_t
}

// Stat is for fs.ReadDirFile interface compatibility.
func (f *ReadDirFile) Stat() (fs.FileInfo, error) {
	return f, nil
}

// Read is for fs.ReadDirFile interface compatibility.
func (f *ReadDirFile) Read(b []byte) (n int, err error) {
	return 0, io.EOF
}

// Close is for fs.ReadDirFile interface compatibility.
func (f *ReadDirFile) Close() error {
	return nil
}

// ReadDir returns fs.FileInfo array with nested into given package directory presentation.
func (f *ReadDirFile) ReadDir(n int) (matches []fs.DirEntry, err error) {
	return f.FTT.ReadDir(strings.TrimSuffix(f.Path(), "/"), n)
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

// ReadDir returns fs.FileInfo array with nested into given package directory presentation.
// It's core function for ReadDirFile and ReadDirFS structures.
func (ftt *FTT_t) ReadDir(dir string, n int) (matches []fs.DirEntry, err error) {
	var prefix string
	if dir != "." {
		prefix = Normalize(dir) + "/" // set terminated slash
	}
	var dirs = map[string]Void{}
	ftt.Enum(func(fkey string, ts *Tagset_t) bool {
		if strings.HasPrefix(fkey, prefix) {
			var suffix = fkey[len(prefix):]
			var sp = strings.IndexByte(suffix, '/')
			if sp < 0 { // file detected
				var ts, _ = ftt.Tagset(fkey)
				matches = append(matches, ts)
				n--
			} else { // dir detected
				var subdir = path.Join(prefix, suffix[:sp])
				if _, ok := dirs[subdir]; !ok {
					dirs[subdir] = Void{}
					var ts, _ = ftt.Tagset(fkey)
					var fp = ts.Path() // extract not normalized path
					var de = (&Tagset_t{nil, ts.tidsz, ts.tagsz}).
						Put(TIDpath, TagString(fp[:len(subdir)]))
					matches = append(matches, de)
					n--
				}
			}
		}
		return n > 0
	})
	if n > 0 {
		err = io.EOF
	}
	return
}

// Open returns ReadDirFile structure associated with group of files in package
// pooled with common directory prefix. Usable to implement fs.FileSystem interface.
func (ftt *FTT_t) Open(dir string) (df fs.ReadDirFile, err error) {
	var prefix string
	if dir != "." {
		prefix = Normalize(dir) + "/" // set terminated slash
	}
	ftt.Enum(func(fkey string, ts *Tagset_t) bool {
		if strings.HasPrefix(fkey, prefix) {
			var dts = ftt.NewTagset()
			dts.Put(TIDpath, TagString(ToSlash(dir)))
			df, err = &ReadDirFile{
				Tagset_t: dts,
				FTT:      ftt,
			}, nil
			return false
		}
		return true
	})
	if df == nil { // on case if not found
		err = &fs.PathError{Op: "open", Path: dir, Err: fs.ErrNotExist}
	}
	return
}

// The End.
