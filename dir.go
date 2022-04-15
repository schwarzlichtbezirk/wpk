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
func Normalize(kpath string) string {
	return strings.ToLower(ToSlash(kpath))
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
	for key := range pack.TOM() {
		if strings.HasPrefix(key, prefix) {
			var suffix = key[len(prefix):]
			var sp = strings.IndexByte(suffix, '/')
			if sp < 0 { // file detected
				var ts, _ = pack.NamedTags(key)
				matches = append(matches, &DirEntry{ts})
				n--
			} else { // dir detected
				var subdir = path.Join(prefix, suffix[:sp])
				if _, ok := dirs[subdir]; !ok {
					dirs[subdir] = struct{}{}
					var ts, _ = pack.NamedTags(key)
					var fp = ts.Path() // extract not normalized path
					var de DirEntry
					de.PutTag(TIDpath, TagString(fp[:len(subdir)]))
					matches = append(matches, &de)
					n--
				}
			}
		}
		if n == 0 {
			return
		}
	}
	if n > 0 {
		err = io.EOF
	}
	return
}

// OpenDir returns ReadDirFile structure associated with group of files in package
// pooled with common directory prefix. Usable to implement fs.FileSystem interface.
func OpenDir(pack Tagger, dir string) (fs.ReadDirFile, error) {
	if !fs.ValidPath(dir) {
		return nil, &fs.PathError{Op: "open", Path: dir, Err: fs.ErrInvalid}
	}
	var prefix string
	if dir != "." {
		prefix = Normalize(dir) + "/" // set terminated slash
	}
	for key := range pack.TOM() {
		if strings.HasPrefix(key, prefix) {
			var ts Tagset_t
			ts.PutTag(TIDpath, TagString(ToSlash(dir)))
			return &ReadDirFile{
				Tagset_t: ts,
				Pack:     pack,
			}, nil
		}
	}
	return nil, &fs.PathError{Op: "opendir", Path: dir, Err: fs.ErrNotExist}
}

// The End.
