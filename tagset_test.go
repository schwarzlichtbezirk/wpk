package wpk_test

import (
	"path/filepath"
	"testing"

	"github.com/schwarzlichtbezirk/wpk"
)

func TestTagset(t *testing.T) {
	const (
		fid    uint32 = 100
		offset uint64 = 0xDEADBEEF
		size   uint64 = 1234
		kpath  string = "Dir\\FileName.Ext"
		kpath2 string = "DIR\\FILENAME.EXT"
		fkey   string = "dir/filename.ext"
		mime   string = "image/jpeg"
	)
	var ts wpk.Tagset_t
	ts.Put(wpk.TIDfid, wpk.TagUint32(uint32(fid)))
	ts.Put(wpk.TIDoffset, wpk.TagUint64(uint64(offset)))
	ts.Put(wpk.TIDsize, wpk.TagUint64(uint64(size)))
	ts.Put(wpk.TIDpath, wpk.TagString(wpk.ToSlash(kpath)))

	if wpk.Normalize(kpath) != fkey || wpk.Normalize(kpath2) != fkey {
		t.Fatal("normalize test failed")
	}

	var tsi = ts.Iterator()
	if tsi == nil {
		t.Fatal("can not create iterator")
	}
	if tsi.TID() != wpk.TIDnone {
		t.Fatal("tag ID in created iterator should be 'none'")
	}

	if ts.Num() != 4 {
		t.Fatalf("wrong number of tags, %d expected, got %d", 4, ts.Num())
	}

	var (
		tag wpk.Tag_t
		ok  bool
		u32 uint32
		u64 uint64
		str string
	)

	// check up FID
	if !tsi.Next() {
		t.Fatal("can not iterate to 'fid'")
	}
	if tsi.TID() != wpk.TIDfid {
		t.Fatal("tag #1 is not 'fid'")
	}
	tag = tsi.Tag()
	if tag == nil {
		t.Fatal("can not get 'fid' tag")
	}
	if u32, ok = tag.Uint32(); !ok {
		t.Fatal("can not convert 'fid' tag to value")
	}
	if u32 != fid {
		t.Fatal("'fid' tag is not equal to original value")
	}

	// check up OFFSET
	if !tsi.Next() {
		t.Fatal("can not iterate to 'offset'")
	}
	if tsi.TID() != wpk.TIDoffset {
		t.Fatal("tag #2 is not 'offset'")
	}
	tag = tsi.Tag()
	if tag == nil {
		t.Fatal("can not get 'offset' tag")
	}
	if u64, ok = tag.Uint64(); !ok {
		t.Fatal("can not convert 'offset' tag to value")
	}
	if u64 != offset {
		t.Fatal("'offset' tag is not equal to original value")
	}

	// check up SIZE
	if !tsi.Next() {
		t.Fatal("can not iterate to 'size'")
	}
	if tsi.TID() != wpk.TIDsize {
		t.Fatal("tag #3 is not 'size'")
	}
	tag = tsi.Tag()
	if tag == nil {
		t.Fatal("can not get 'size' tag")
	}
	if u64, ok = tag.Uint64(); !ok {
		t.Fatal("can not convert 'size' tag to value")
	}
	if u64 != size {
		t.Fatal("'size' tag is not equal to original value")
	}

	// check up PATH
	if !tsi.Next() {
		t.Fatal("can not iterate to 'path'")
	}
	if tsi.TID() != wpk.TIDpath {
		t.Fatal("tag #4 is not 'path'")
	}
	if tsi.TagLen() != uint16(len(kpath)) {
		t.Fatal("length of 'path' tag does not equal to original length")
	}
	tag = tsi.Tag()
	if tag == nil {
		t.Fatal("can not get 'path' tag")
	}
	if str, ok = tag.String(); !ok {
		t.Fatal("can not convert 'path' tag to value")
	}
	if str != wpk.ToSlash(kpath) {
		t.Fatal("'path' tag is not equal to original value")
	}

	// check up valid iterations finish
	if tsi.Next() {
		t.Fatal("iterator does not finished")
	}
	if !tsi.Passed() {
		t.Fatal("iterations does not reached till the end")
	}
	if tsi.TID() != wpk.TIDnone {
		t.Fatal("tag ID in finished iterator should be 'none'")
	}

	// check up 'Has'
	if !(ts.Has(wpk.TIDfid) && ts.Has(wpk.TIDoffset) && ts.Has(wpk.TIDsize) && ts.Has(wpk.TIDpath)) {
		t.Fatal("something does not pointed that should be present")
	}
	if ts.Has(wpk.TIDmd5) {
		t.Fatal("'md5' tag is not set, but it's pointed that it present")
	}

	// check up helpers functions
	if ts.FID() != wpk.FID_t(fid) {
		t.Fatal("'FID' function does not work correctly")
	}
	if ts.Offset() != int64(offset) {
		t.Fatal("'Offset' function does not work correctly")
	}
	if ts.Size() != int64(size) {
		t.Fatal("'Size' function does not work correctly")
	}
	if ts.Path() != wpk.ToSlash(kpath) {
		t.Fatal("'Path' function does not work correctly")
	}
	if ts.Name() != filepath.Base(kpath) {
		t.Fatal("'Name' function does not work correctly")
	}

	// check up 'Set' and 'Del'
	if ts.Set(wpk.TIDpath, wpk.TagString(wpk.ToSlash(kpath2))) {
		t.Fatal("content of 'path' tag should be replased by 'Set'")
	}
	if ts.Num() != 4 {
		t.Fatal("number of tags after replace 'path' must not be changed")
	}
	if ts.Path() != wpk.ToSlash(kpath2) {
		t.Fatal("'Set' function does not work correctly")
	}
	if !ts.Set(wpk.TIDmime, wpk.TagString(mime)) {
		t.Fatal("content of 'mime' tag should be added by 'Set'")
	}
	if ts.Num() != 4+1 {
		t.Fatal("number of tags after add 'mime' must be added by one")
	}
	if tag, ok = ts.Get(wpk.TIDmime); !ok {
		t.Fatal("can not get 'mime' tag content")
	}
	if str, _ = tag.String(); str != mime {
		t.Fatal("'mime' tag is not equal to original value")
	}
	if !ts.Del(wpk.TIDmime) {
		t.Fatal("'mime' tag is not deleted")
	}
	if ts.Has(wpk.TIDmime) {
		t.Fatal("'mime' tag must not be found after deletion")
	}
	if ts.Num() != 4 {
		t.Fatal("number of tags after delete 'mime' must be restored")
	}
	if ts.Del(wpk.TIDmime) {
		t.Fatal("'mime' tag can not be deleted again")
	}
	if ts.Num() != 4 {
		t.Fatal("number of tags after repeated delete 'mime' must be unchanged")
	}
}
