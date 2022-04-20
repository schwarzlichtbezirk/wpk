package wpk_test

import (
	"path/filepath"
	"testing"

	"github.com/schwarzlichtbezirk/wpk"
)

func TestTagset(t *testing.T) {
	const (
		fid    = 100
		offset = 0xDEADBEEF
		size   = 1234
		kpath1 = `Dir\FileName.Ext`
		kpath2 = `DIR\FILENAME.EXT`
		fkey   = `dir/filename.ext`
		mime   = "image/jpeg"
	)
	var ts wpk.Tagset_t
	ts.Put(wpk.TIDoffset, wpk.TagFOffset(offset))
	ts.Put(wpk.TIDsize, wpk.TagFSize(size))
	ts.Put(wpk.TIDfid, wpk.TagFID(fid))
	ts.Put(wpk.TIDpath, wpk.TagString(wpk.ToSlash(kpath1)))

	if wpk.Normalize(kpath1) != fkey || wpk.Normalize(kpath2) != fkey {
		t.Fatal("normalize test failed")
	}

	var tsi = ts.Iterator()
	if tsi.TID() != wpk.TIDnone {
		t.Fatal("tag ID in created iterator should be 'none'")
	}

	if ts.Num() != 4 {
		t.Fatalf("wrong number of tags, %d expected, got %d", 4, ts.Num())
	}

	var (
		tag wpk.Tag_t
		ok  bool
		fv  wpk.FID_t
		ov  wpk.FOffset_t
		sv  wpk.FSize_t
		str string
	)

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
	if ov, ok = tag.FOffset(); !ok {
		t.Fatal("can not convert 'offset' tag to value")
	}
	if ov != offset {
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
	if sv, ok = tag.FSize(); !ok {
		t.Fatal("can not convert 'size' tag to value")
	}
	if sv != size {
		t.Fatal("'size' tag is not equal to original value")
	}

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
	if fv, ok = tag.FID(); !ok {
		t.Fatal("can not convert 'fid' tag to value")
	}
	if fv != fid {
		t.Fatal("'fid' tag is not equal to original value")
	}

	// check up PATH
	if !tsi.Next() {
		t.Fatal("can not iterate to 'path'")
	}
	if tsi.TID() != wpk.TIDpath {
		t.Fatal("tag #4 is not 'path'")
	}
	if tsi.TagLen() != wpk.TSSize_t(len(kpath1)) {
		t.Fatal("length of 'path' tag does not equal to original length")
	}
	tag = tsi.Tag()
	if tag == nil {
		t.Fatal("can not get 'path' tag")
	}
	if str, ok = tag.String(); !ok {
		t.Fatal("can not convert 'path' tag to value")
	}
	if str != wpk.ToSlash(kpath1) {
		t.Fatal("'path' tag is not equal to original value")
	}

	// check up valid iterations finish
	if tsi.Failed() {
		t.Fatal("content is broken")
	}
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
	if !(ts.Has(wpk.TIDoffset) && ts.Has(wpk.TIDsize) && ts.Has(wpk.TIDfid) && ts.Has(wpk.TIDpath)) {
		t.Fatal("something does not pointed that should be present")
	}
	if ts.Has(wpk.TIDmd5) {
		t.Fatal("'md5' tag is not set, but it's pointed that it present")
	}

	// check up helpers functions
	if v, ok := ts.FID(); !ok || v != fid {
		t.Fatal("'FID' function does not work correctly")
	}
	if v, ok := ts.FOffset(); !ok || v != offset {
		t.Fatal("'FOffset' function does not work correctly")
	}
	if v, ok := ts.FSize(); !ok || v != size {
		t.Fatal("'FSize' function does not work correctly")
	}
	if ts.Path() != wpk.ToSlash(kpath1) {
		t.Fatal("'Path' function does not work correctly")
	}
	if ts.Name() != filepath.Base(kpath1) {
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

// The End.
