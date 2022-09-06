package wpk

import (
	"io/fs"
	"path"
	"path/filepath"
)

type Void = struct{}

type Union struct {
	List      []Packager
	workspace string // workspace directory in package
}

// Close call Close-function for all included into the union packages.
// io.Closer implementation.
func (u *Union) Close() (err error) {
	for _, pack := range u.List {
		if err1 := pack.Close(); err1 != nil {
			err = err1
		}
	}
	return
}

// AllKeys returns list of all accessible files in union of packages.
// If union have more than one file with the same name, only first
// entry will be included to result.
func (u *Union) AllKeys() (res []string) {
	var m = map[string]Void{}
	for _, pack := range u.List {
		pack.Enum(func(fkey string, ts *Tagset_t) bool {
			if _, ok := m[fkey]; !ok {
				res = append(res, fkey)
				m[fkey] = Void{}
			}
			return true
		})
	}
	return
}

// Glob returns the names of all files in union matching pattern or nil
// if there is no matching file.
func (u *Union) Glob(pattern string) (res []string, err error) {
	pattern = path.Join(u.workspace, Normalize(pattern))
	if _, err = filepath.Match(pattern, ""); err != nil {
		return
	}
	var m = map[string]Void{}
	for _, pack := range u.List {
		pack.Enum(func(fkey string, ts *Tagset_t) bool {
			if _, ok := m[fkey]; !ok {
				if matched, _ := filepath.Match(pattern, fkey); matched {
					res = append(res, fkey)
				}
				m[fkey] = Void{}
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
	u1.workspace = path.Join(u.workspace, dir)
	for _, pack := range u.List {
		if sub1, err1 := pack.Sub(dir); err1 == nil {
			u1.List = append(u1.List, sub1.(Packager))
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
func (u *Union) Stat(name string) (fs.FileInfo, error) {
	var fullname = path.Join(u.workspace, name)
	var ts *Tagset_t
	var is bool
	for _, pack := range u.List {
		if ts, is = pack.Tagset(fullname); is {
			return ts, nil
		}
	}
	return nil, &fs.PathError{Op: "stat", Path: name, Err: fs.ErrNotExist}
}

// ReadFile returns slice with nested into union of packages file content.
// If union have more than one file with the same name, first will be returned.
// fs.ReadFileFS implementation.
func (u *Union) ReadFile(name string) ([]byte, error) {
	var fullname = path.Join(u.workspace, name)
	var ts *Tagset_t
	var is bool
	for _, pack := range u.List {
		if ts, is = pack.Tagset(fullname); is {
			var f, err = pack.OpenTagset(ts)
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
	return nil, &fs.PathError{Op: "readfile", Path: name, Err: fs.ErrNotExist}
}

// ReadDir reads the named directory
// and returns a list of directory entries sorted by filename.
func (u *Union) ReadDir(dir string) (list []fs.DirEntry, err error) {
	var fullname = path.Join(u.workspace, dir)
	var m = map[string]Void{}
	for _, pack := range u.List {
		var pl []fs.DirEntry
		if pl, err = pack.ReadDir(fullname); err != nil {
			return
		}
		for _, de := range pl {
			var name = de.Name()
			if _, ok := m[name]; !ok {
				list = append(list, de)
				m[name] = Void{}
			}
		}
	}
	return
}

// Open implements access to nested into union of packages file or directory by keyname.
// If union have more than one file with the same name, first will be returned.
// fs.FS implementation.
func (u *Union) Open(dir string) (fs.File, error) {
	var fullname = path.Join(u.workspace, dir)
	for _, pack := range u.List {
		if ts, is := pack.Tagset(fullname); is {
			return pack.OpenTagset(ts)
		}
	}
	for _, pack := range u.List {
		if f, err := pack.Open(fullname); err == nil {
			return f, nil
		}
	}
	return nil, &fs.PathError{Op: "open", Path: dir, Err: fs.ErrNotExist}
}

// The End.
