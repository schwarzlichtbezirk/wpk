package wpk

import (
	"bytes"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
)

var efre = regexp.MustCompile(`\$\{\w+\}`)

// Envfmt replaces environment variables entries in file path to there values.
// Environment variables must be enclosed as ${...} in string.
func Envfmt(p string) string {
	return filepath.ToSlash(efre.ReplaceAllStringFunc(p, func(name string) string {
		return os.Getenv(name[2 : len(name)-1]) // strip ${...} and replace by env value
	}))
}

// PathExists check up file or directory existence.
func PathExists(path string) (bool, error) {
	var err error
	if _, err = os.Stat(path); err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
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
	for key := range pack.NFTO() {
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
					var buf bytes.Buffer
					Tagset{
						TIDpath: TagString(fp[:len(subdir)]),
					}.WriteTo(&buf)
					matches = append(matches, &DirEntry{buf.Bytes()})
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
	for key := range pack.NFTO() {
		if strings.HasPrefix(key, prefix) {
			var buf bytes.Buffer
			Tagset{
				TIDpath: TagString(ToSlash(dir)),
			}.WriteTo(&buf)
			return &ReadDirFile{
				TagSlice: buf.Bytes(),
				Pack:     pack,
			}, nil
		}
	}
	return nil, &fs.PathError{Op: "opendir", Path: dir, Err: fs.ErrNotExist}
}

// The End.
