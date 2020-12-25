package wpk

import (
	"encoding/binary"
	"io"
	"os"
)

// Tags sets map.
type TagsMap map[string]Tagset

// Package writer structure.
type Writer struct {
	PackHdr
	Tags TagsMap
}

// Opens package for reading. At first its checkup file signature, then
// reads records table, and reads file tags set table. Tags set
// for each file must contain at least file ID, file name and creation time.
func (pack *Writer) Read(r io.ReadSeeker) (err error) {
	// go to file start
	if _, err = r.Seek(0, io.SeekStart); err != nil {
		return
	}
	// read header
	if err = binary.Read(r, binary.LittleEndian, &pack.PackHdr); err != nil {
		return
	}
	if string(pack.Signature[:]) == Prebuild {
		return ErrSignPre
	}
	if string(pack.Signature[:]) != Signature {
		return ErrSignBad
	}
	pack.Tags = make(TagsMap, pack.TagNumber)

	// read file tags set table
	if _, err = r.Seek(int64(pack.TagOffset), io.SeekStart); err != nil {
		return
	}
	var n int64
	for {
		if _, err = r.Seek(0, io.SeekCurrent); err != nil {
			return
		}
		var tags = Tagset{}
		if n, err = tags.ReadFrom(r); err != nil {
			return
		}
		if n == 2 {
			break // end marker was readed
		}

		// check tags fields
		if _, ok := tags[TID_path]; !ok {
			return &ErrTag{ErrKey{ErrNoPath, ""}, TID_path}
		}
		var key = ToKey(tags.Path())
		if _, ok := pack.Tags[key]; ok {
			return &ErrTag{ErrKey{ErrAlready, key}, TID_path}
		}

		if _, ok := tags[TID_FID]; !ok {
			return &ErrTag{ErrKey{ErrNoFID, key}, TID_FID}
		}
		var fid = tags.FID()
		if fid > pack.RecNumber {
			return &ErrTag{ErrKey{ErrOutFID, key}, TID_FID}
		}

		if _, ok := tags[TID_offset]; !ok {
			return &ErrTag{ErrKey{ErrNoOffset, key}, TID_offset}
		}
		if _, ok := tags[TID_size]; !ok {
			return &ErrTag{ErrKey{ErrNoSize, key}, TID_size}
		}
		var offset, size = tags.Offset(), tags.Size()
		if offset < PackHdrSize || offset >= int64(pack.TagOffset) {
			return &ErrTag{ErrKey{ErrOutOff, key}, TID_offset}
		}
		if offset+size > int64(pack.TagOffset) {
			return &ErrTag{ErrKey{ErrOutSize, key}, TID_size}
		}

		// insert file tags
		pack.Tags[key] = tags
	}

	return
}

// Writes prebuild header for new empty package.
func (pack *Writer) Begin(w io.WriteSeeker) (err error) {
	// reset header
	copy(pack.Signature[:], Prebuild)
	pack.TagOffset = PackHdrSize
	pack.RecNumber = 0
	pack.TagNumber = 0
	// setup empty tags table
	pack.Tags = TagsMap{}
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
func (pack *Writer) Append(w io.WriteSeeker) (err error) {
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
func (pack *Writer) Complete(w io.WriteSeeker) (err error) {
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
func (pack *Writer) PackData(w io.WriteSeeker, r io.Reader, kpath string) (tags Tagset, err error) {
	var key = ToKey(kpath)
	if _, ok := pack.Tags[key]; ok {
		err = &ErrKey{ErrAlready, key}
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
		TID_path:   TagString(ToSlash(kpath)),
	}
	pack.Tags[key] = tags

	// update header
	pack.TagOffset = OFFSET(offset + size)
	pack.RecNumber++
	pack.TagNumber++
	return
}

// Puts file with given file handle into package and associate keyname "kpath" with it.
func (pack *Writer) PackFile(w io.WriteSeeker, file *os.File, kpath string) (tags Tagset, err error) {
	var fi os.FileInfo
	if fi, err = file.Stat(); err != nil {
		return
	}
	if tags, err = pack.PackData(w, file, kpath); err != nil {
		return
	}

	tags[TID_created] = TagUint64(uint64(fi.ModTime().Unix()))
	tags[TID_link] = TagString(ToSlash(kpath))
	return
}

// Function called before each file or directory with its parameters
// during PackDir processing. Returns whether to process the file or directory.
// Can be used as filter and logger.
type PackDirHook = func(fi os.FileInfo, kpath, fpath string) bool

// Puts all files of given folder and it's subfolders into package.
// Hook function can be nil.
func (pack *Writer) PackDir(w io.WriteSeeker, dirname, prefix string, hook PackDirHook) (err error) {
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
		if fi != nil {
			var kpath = prefix + fi.Name()
			var fpath = dirname + fi.Name()
			if hook == nil || hook(fi, kpath, fpath) {
				if fi.IsDir() {
					if err = pack.PackDir(w, fpath+"/", kpath+"/", hook); err != nil {
						return
					}
				} else if func() {
					var file *os.File
					if file, err = os.Open(fpath); err != nil {
						return
					}
					defer file.Close()

					if _, err = pack.PackFile(w, file, kpath); err != nil {
						return
					}
				}(); err != nil {
					return
				}
			}
		}
	}
	return
}

// Rename tags set with file name 'oldname' to 'newname'.
// Keeps link to original file name.
func (pack *Writer) Rename(oldname, newname string) error {
	var key1 = ToKey(oldname)
	var key2 = ToKey(newname)
	var tags, ok = pack.Tags[key1]
	if !ok {
		return &ErrKey{ErrNotFound, key1}
	}
	if _, ok = pack.Tags[key2]; ok {
		return &ErrKey{ErrAlready, key2}
	}

	tags[TID_path] = TagString(ToSlash(newname))
	delete(pack.Tags, key1)
	pack.Tags[key2] = tags
	return nil
}

// Clone tags set with file name 'oldname' and replace name tag in it to 'newname'.
// Keeps link to original file name.
func (pack *Writer) PutAlias(oldname, newname string) error {
	var key1 = ToKey(oldname)
	var key2 = ToKey(newname)
	var tags1, ok = pack.Tags[key1]
	if !ok {
		return &ErrKey{ErrNotFound, key1}
	}
	if _, ok = pack.Tags[key2]; ok {
		return &ErrKey{ErrAlready, key2}
	}

	var tags2 = Tagset{}
	for k, v := range tags1 {
		tags2[k] = v
	}
	tags2[TID_path] = TagString(ToSlash(newname))
	pack.Tags[key2] = tags2
	pack.TagNumber++
	return nil
}

// Delete tags set with specified file name. Data block is still remains.
func (pack *Writer) DelAlias(name string) bool {
	var key = ToKey(name)
	var _, ok = pack.Tags[key]
	if ok {
		delete(pack.Tags, key)
		pack.TagNumber--
	}
	return ok
}

// The End.
