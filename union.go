package wpk

import (
	"io"
	"io/fs"
	"path"
	"strconv"
	"strings"
)

// Void is empty structure to release of the set of keys.
type Void = struct{}

// UnionDir is helper to get access to union directories,
// that contains files from all packages with same dir if it present.
type UnionDir struct {
	TagsetRaw
	*Union
}

// fs.ReadDirFile interface implementation.
func (f *UnionDir) Stat() (fs.FileInfo, error) {
	return f, nil
}

// fs.ReadDirFile interface implementation.
func (f *UnionDir) Read([]byte) (int, error) {
	return 0, io.EOF
}

// fs.ReadDirFile interface implementation.
func (f *UnionDir) Close() error {
	return nil
}

// fs.ReadDirFile interface implementation.
func (f *UnionDir) ReadDir(n int) ([]fs.DirEntry, error) {
	var dir = f.Path()
	if len(f.List) > 0 {
		if dir = f.List[0].TrimPath(dir); dir == "" {
			return nil, ErrOtherSubdir
		}
	}
	return f.ReadDirN(dir, n)
}

// Union glues list of packages into single filesystem.
type Union struct {
	List []*Package
}

// Close call Close-function for all included into the union packages.
// io.Closer implementation.
func (u *Union) Close() (err error) {
	for _, pkg := range u.List {
		if err1 := pkg.Tagger.Close(); err1 != nil {
			err = err1
		}
	}
	return
}

// AllKeys returns list of all accessible files in union of packages.
// If union have more than one file with the same name, only first
// entry will be included to result.
func (u *Union) AllKeys() (res []string) {
	var found = map[string]Void{}
	for _, pkg := range u.List {
		pkg.Enum(func(fkey string, ts TagsetRaw) bool {
			if _, ok := found[fkey]; !ok {
				res = append(res, fkey)
				found[fkey] = Void{}
			}
			return true
		})
	}
	return
}

// Sub clones object and gives access to pointed subdirectory.
// fs.SubFS implementation.
func (u *Union) Sub(dir string) (fs.FS, error) {
	var u1 Union
	for _, pkg := range u.List {
		if sub1, err1 := pkg.Sub(dir); err1 == nil {
			u1.List = append(u1.List, sub1.(*Package))
		}
	}
	if len(u1.List) == 0 {
		return nil, &fs.PathError{Op: "sub", Path: dir, Err: fs.ErrNotExist}
	}
	return &u1, nil
}

// Stat returns a fs.FileInfo describing the file.
// If union have more than one file with the same name, info of the first will be returned.
// fs.StatFS implementation.
func (u *Union) Stat(fpath string) (fs.FileInfo, error) {
	var ts TagsetRaw
	var is bool
	for _, pkg := range u.List {
		if ts, is = pkg.GetTagset(fpath); is {
			return ts, nil
		}
	}
	return nil, &fs.PathError{Op: "stat", Path: fpath, Err: fs.ErrNotExist}
}

// Glob returns the names of all files in union matching pattern or nil
// if there is no matching file.
func (u *Union) Glob(pattern string) (res []string, err error) {
	pattern = ToSlash(pattern)
	if _, err = path.Match(pattern, ""); err != nil {
		return
	}
	var found = map[string]Void{}
	for _, pkg := range u.List {
		pkg.Enum(func(fkey string, ts TagsetRaw) bool {
			if _, ok := found[fkey]; !ok {
				if matched, _ := path.Match(pattern, fkey); matched {
					res = append(res, fkey)
				}
				found[fkey] = Void{}
			}
			return true
		})
	}
	return
}

// ReadFile returns slice with nested into union of packages file content.
// If union have more than one file with the same name, first will be returned.
// fs.ReadFileFS implementation.
func (u *Union) ReadFile(fpath string) ([]byte, error) {
	for _, pkg := range u.List {
		if ts, is := pkg.GetTagset(fpath); is {
			var f, err = pkg.Tagger.OpenTagset(ts)
			if err != nil {
				return nil, err
			}
			defer f.Close()

			var size = ts.Size()
			var buf = make([]byte, size)
			_, err = f.Read(buf)
			return buf, err
		}
	}
	return nil, &fs.PathError{Op: "readfile", Path: fpath, Err: fs.ErrNotExist}
}

// ReadDir reads the named directory
// and returns a list of directory entries sorted by filename.
func (u *Union) ReadDirN(dir string, n int) (list []fs.DirEntry, err error) {
	dir = ToSlash(dir)
	var found = map[string]fs.DirEntry{}
	var ni = n
	for _, pkg := range u.List {
		var fulldir = pkg.FullPath(dir)
		var prefix string
		if fulldir != "." {
			prefix = fulldir + "/" // set terminated slash
		}

		pkg.rwm.Range(func(fkey string, ts TagsetRaw) bool {
			if strings.HasPrefix(fkey, prefix) {
				var suffix = fkey[len(prefix):]
				var sp = strings.IndexByte(suffix, '/')
				if sp < 0 { // file detected
					found[suffix] = ts
					ni--
				} else { // dir detected
					var subdir = path.Join(prefix, suffix[:sp])
					if _, ok := found[subdir]; !ok {
						var dts = TagsetRaw{}.
							Put(TIDpath, StrTag(subdir))
						var f = &PackDirFile{
							TagsetRaw: dts,
							ftt:       pkg.FTT,
						}
						found[subdir] = f
						ni--
					}
				}
			}
			return ni != 0
		})

		if ni == 0 {
			break
		}
	}

	list = make([]fs.DirEntry, len(found))
	var i int
	for _, de := range found {
		list[i] = de
		i++
	}
	if ni > 0 {
		err = io.EOF
	}
	return
}

// ReadDir reads the named directory
// and returns a list of directory entries sorted by filename.
// fs.ReadDirFS interface implementation.
func (u *Union) ReadDir(dir string) ([]fs.DirEntry, error) {
	return u.ReadDirN(dir, -1)
}

// Open implements access to nested into union of packages file or directory by keyname.
// If union have more than one file with the same name, first will be returned.
// fs.FS implementation.
func (u *Union) Open(dir string) (fs.File, error) {
	dir = ToSlash(dir)
	if len(u.List) == 0 {
		return nil, &fs.PathError{Op: "open", Path: dir, Err: fs.ErrNotExist}
	}

	if len(u.List) > 0 {
		if fulldir := u.List[0].FullPath(dir); strings.HasPrefix(fulldir, PackName+"/") {
			var idx, err = strconv.ParseUint(dir[len(PackName)+1:], 10, 32)
			if err != nil {
				return nil, &fs.PathError{Op: "open", Path: dir, Err: err}
			}
			if idx >= uint64(len(u.List)) {
				return nil, &fs.PathError{Op: "open", Path: dir, Err: fs.ErrNotExist}
			}
			return u.List[idx].Open(PackName)
		}
	}

	// try to get the file
	for _, pkg := range u.List {
		if ts, is := pkg.GetTagset(dir); is {
			return pkg.Tagger.OpenTagset(ts)
		}
	}

	// try to get the folder
	var prefix string
	if dir != "." && dir != "" {
		prefix = dir + "/" // set terminated slash
	}
	for _, pkg := range u.List {
		var f *UnionDir
		pkg.Enum(func(fkey string, ts TagsetRaw) bool {
			if strings.HasPrefix(fkey, prefix) {
				var dts = TagsetRaw{}.
					Put(TIDpath, StrTag(pkg.FullPath(dir)))
				f = &UnionDir{
					TagsetRaw: dts,
					Union:     u,
				}
				return false
			}
			return true
		})
		if f != nil {
			return f, nil
		}
	}
	// on case if not found
	return nil, &fs.PathError{Op: "open", Path: dir, Err: fs.ErrNotExist}
}

// The End.
