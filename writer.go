package wpk

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
)

// Writes prebuild header for new empty package.
func (pack *Package) Begin(w io.WriteSeeker) (err error) {
	// reset header
	copy(pack.Signature[:], Prebuild)
	pack.TagOffset = PackHdrSize
	pack.RecNumber = 0
	pack.TagNumber = 0
	// setup empty tags table
	pack.Tags = map[string]Tagset{}
	// go to file start
	if _, err = w.Seek(0, io.SeekStart); err != nil {
		return
	}
	// write prebuild header
	if err = binary.Write(w, binary.LittleEndian, &pack.PackHdr); err != nil {
		return
	}
	return
}

// Writes prebuild header for previously opened package to append new files.
func (pack *Package) Append(w io.WriteSeeker) (err error) {
	// partially reset header
	copy(pack.Signature[:], Prebuild)
	// go to file start
	if _, err = w.Seek(0, io.SeekStart); err != nil {
		return
	}
	// rewrite prebuild header
	if err = binary.Write(w, binary.LittleEndian, &pack.PackHdr); err != nil {
		return
	}
	// go to tags table start to replace it by new data
	if _, err = w.Seek(int64(pack.TagOffset), io.SeekStart); err != nil {
		return
	}
	return
}

// Finalize package writing. Writes true signature and header settings.
func (pack *Package) Complete(w io.WriteSeeker) (err error) {
	// get tags table offset as actual end of file
	var tagoffset int64
	if tagoffset, err = w.Seek(0, io.SeekEnd); err != nil {
		return
	}
	pack.TagOffset = OFFSET(tagoffset)
	pack.TagNumber = FID(len(pack.Tags))
	// write files tags table
	for _, tags := range pack.Tags {
		if _, err = tags.WriteTo(w); err != nil {
			return
		}
	}
	// write tags table end marker
	if err = binary.Write(w, binary.LittleEndian, TID(0)); err != nil {
		return
	}

	// rewrite true header
	if _, err = w.Seek(0, io.SeekStart); err != nil {
		return
	}
	copy(pack.Signature[:], Signature)
	if err = binary.Write(w, binary.LittleEndian, &pack.PackHdr); err != nil {
		return
	}
	return
}

// Puts data streamed by given reader into package as a file and associate keyname "kpath" with it.
func (pack *Package) PackData(w io.WriteSeeker, r io.Reader, kpath string) (tags Tagset, err error) {
	var key = ToKey(kpath)
	if _, ok := pack.Tags[key]; ok {
		err = ErrAlready
		return
	}

	// get offset and put data ckage
	var offset, size int64
	if offset, err = w.Seek(0, io.SeekCurrent); err != nil {
		return
	}
	if size, err = io.Copy(w, r); err != nil {
		return
	}

	// insert new entry to tags table
	tags = Tagset{
		TID_FID:    TagUint32(uint32(pack.RecNumber + 1)),
		TID_offset: TagUint64(uint64(offset)),
		TID_size:   TagUint64(uint64(size)),
		TID_path:   TagString(kpath),
	}
	pack.Tags[key] = tags

	// update header
	pack.TagOffset = OFFSET(offset + size)
	pack.RecNumber++
	pack.TagNumber = FID(len(pack.Tags))
	return
}

// Puts file with given file full path "fpath" into package and associate keyname "kpath" with it.
func (pack *Package) PackFile(w io.WriteSeeker, kpath, fpath string) (tags Tagset, err error) {
	var file *os.File
	if file, err = os.Open(fpath); err != nil {
		return
	}
	defer file.Close()

	var fi os.FileInfo
	if fi, err = file.Stat(); err != nil {
		return
	}
	if tags, err = pack.PackData(w, file, kpath); err != nil {
		return
	}

	tags[TID_created] = TagUint64(uint64(fi.ModTime().Unix()))
	return
}

// Wrapper to hold file name with error.
type FileError struct {
	What error
	Name string
}

func (e *FileError) Error() string {
	return fmt.Sprintf("error on file '%s': %s", e.Name, e.What.Error())
}

func (e *FileError) Unwrap() error {
	return e.What
}

// Function to report about each file start processing by PackDir function.
type FileReport = func(fi os.FileInfo, kpath, fpath string)

// Puts all files of given folder and it's subfolders into package.
func (pack *Package) PackDir(w io.WriteSeeker, dirname, prefix string, report FileReport) (err error) {
	var fis []os.FileInfo
	if func() {
		var dir *os.File
		if dir, err = os.Open(dirname); err != nil {
			return
		}
		defer dir.Close()

		if fis, err = dir.Readdir(-1); err != nil {
			return
		}
	}(); err != nil {
		return
	}
	for _, fi := range fis {
		var kpath = prefix + fi.Name()
		var fpath = dirname + fi.Name()
		if fi.IsDir() {
			if err = pack.PackDir(w, fpath+"/", kpath+"/", report); err != nil {
				return
			}
		} else {
			if _, err = pack.PackFile(w, kpath, fpath); err != nil {
				err = &FileError{What: err, Name: kpath}
				return
			}
			if report != nil {
				report(fi, kpath, fpath)
			}
		}
	}
	return
}

// Clone tags set with file name 'oldname' and replace name tag in it to 'newname'.
// Puts link to original file name.
func (pack *Package) PutAlias(oldname, newname string) error {
	var key1 = ToKey(oldname)
	var key2 = ToKey(newname)
	var tags1, ok = pack.Tags[key1]
	if !ok {
		return ErrNotFound
	}
	if _, ok = pack.Tags[key2]; ok {
		return ErrAlready
	}
	var tags2 = Tagset{}
	for k, v := range tags1 {
		tags2[k] = v
	}
	tags2[TID_path] = TagString(newname)
	if _, ok := tags2.String(TID_link); !ok {
		tags2[TID_link] = TagString(oldname)
	}
	pack.Tags[key2] = tags2
	pack.TagNumber = FID(len(pack.Tags))
	return nil
}

// Delete tags set with specified file name. Data block is still remains.
func (pack *Package) DelAlias(name string) bool {
	var key = ToKey(name)
	var _, ok = pack.Tags[key]
	if ok {
		delete(pack.Tags, key)
		pack.TagNumber = FID(len(pack.Tags))
	}
	return ok
}

// The End.
