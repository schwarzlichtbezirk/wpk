package wpk

import (
	"errors"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
)

// DirEntry is directory representation of nested into package files.
// No any reader for directory implementation.
// fs.DirEntry interface implementation.
type DirEntry struct {
	Tagset_t // has fs.FileInfo interface
}

// Type is for fs.DirEntry interface compatibility.
func (f *DirEntry) Type() fs.FileMode {
	if f.Has(TIDfid) { // file ID is absent for dir
		return 0444
	}
	return fs.ModeDir
}

// Info returns the FileInfo for the file or subdirectory described by the entry.
func (f *DirEntry) Info() (fs.FileInfo, error) {
	return &f.Tagset_t, nil
}

// ReadDirFile is a directory file whose entries can be read with the ReadDir method.
// fs.ReadDirFile interface implementation.
type ReadDirFile struct {
	Tagset_t // has fs.FileInfo interface
	Pack     Tagger
}

// Stat is for fs.ReadDirFile interface compatibility.
func (f *ReadDirFile) Stat() (fs.FileInfo, error) {
	return &f.Tagset_t, nil
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
	return ReadDir(f.Pack, strings.TrimSuffix(f.Path(), "/"), n)
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

// ToSlash brings filenames to true slashes.
var ToSlash = filepath.ToSlash

// Normalize brings file path to normalized form. It makes argument lowercase,
// change back slashes to normal slashes. Normalized path is the key to FTTMap.
func Normalize(fpath string) string {
	return strings.ToLower(ToSlash(fpath))
}

// ReadDir returns fs.FileInfo array with nested into given package directory presentation.
// It's core function for ReadDirFile and ReadDirFS structures.
func ReadDir(pack Tagger, dir string, n int) (matches []fs.DirEntry, err error) {
	if !fs.ValidPath(dir) {
		return nil, &fs.PathError{Op: "readdir", Path: dir, Err: fs.ErrInvalid}
	}
	var prefix string
	if dir != "." {
		prefix = Normalize(dir) + "/" // set terminated slash
	}
	var dirs = map[string]struct{}{}
	pack.Enum(func(fkey string, ts *Tagset_t) bool {
		if strings.HasPrefix(fkey, prefix) {
			var suffix = fkey[len(prefix):]
			var sp = strings.IndexByte(suffix, '/')
			if sp < 0 { // file detected
				var ts, _ = pack.Tagset(fkey)
				matches = append(matches, &DirEntry{*ts})
				n--
			} else { // dir detected
				var subdir = path.Join(prefix, suffix[:sp])
				if _, ok := dirs[subdir]; !ok {
					dirs[subdir] = struct{}{}
					var ts, _ = pack.Tagset(fkey)
					var fp = ts.Path() // extract not normalized path
					var de DirEntry
					de.Put(TIDpath, TagString(fp[:len(subdir)]))
					matches = append(matches, &de)
					n--
				}
			}
		}
		if n == 0 {
			return false
		}
		return true
	})
	if n > 0 {
		err = io.EOF
	}
	return
}

// OpenDir returns ReadDirFile structure associated with group of files in package
// pooled with common directory prefix. Usable to implement fs.FileSystem interface.
func OpenDir(pack Tagger, dir string) (df fs.ReadDirFile, err error) {
	if !fs.ValidPath(dir) {
		return nil, &fs.PathError{Op: "open", Path: dir, Err: fs.ErrInvalid}
	}
	var prefix string
	if dir != "." {
		prefix = Normalize(dir) + "/" // set terminated slash
	}
	pack.Enum(func(fkey string, ts *Tagset_t) bool {
		if strings.HasPrefix(fkey, prefix) {
			var dts Tagset_t
			dts.Put(TIDpath, TagString(ToSlash(dir)))
			df, err = &ReadDirFile{
				Tagset_t: dts,
				Pack:     pack,
			}, nil
			return false
		}
		return true
	})
	if df == nil { // on case if not found
		err = &fs.PathError{Op: "opendir", Path: dir, Err: fs.ErrNotExist}
	}
	return
}

// The End.
