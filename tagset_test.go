package wpk_test

import (
	"fmt"
	"path"
	"testing"
	"time"

	"github.com/schwarzlichtbezirk/wpk"
)

func TestTagset(t *testing.T) {
	const (
		fid    = 100
		offset = 0xDEADBEEF
		size   = 1234
		fkey   = `Dir/FileName.ext`
		fkey1  = `Dir/FileName.ext`
		fkey2  = `Dir\FileName.ext`
		mime   = "image/jpeg"
	)
	var ts = wpk.TagsetRaw{}.
		Put(wpk.TIDoffset, wpk.UintTag(offset)).
		Put(wpk.TIDsize, wpk.UintTag(size)).
		Put(wpk.TIDpath, wpk.StrTag(wpk.ToSlash(fkey1))).
		Put(wpk.TIDfid, wpk.UintTag(fid))
	var tsi = ts.Iterator()

	var (
		tag wpk.TagRaw
		ok  bool
		fv  uint
		ov  uint
		sv  uint
		str string
	)

	var assert = func(cond bool, msg string) {
		if !cond {
			t.Error(msg)
		}
	}

	assert(wpk.ToSlash(fkey1) == fkey, "toslash test failed")
	assert(wpk.ToSlash(fkey2) == fkey, "toslash test failed")
	assert(tsi.TID() == wpk.TIDnone, "tag ID in created iterator should be 'none'")
	assert(ts.Num() == 4, "wrong number of tags")

	// check up OFFSET
	assert(tsi.Next(), "can not iterate to 'offset'")
	assert(tsi.TID() == wpk.TIDoffset, "tag #1 is not 'offset'")
	tag = tsi.Tag()
	assert(tag != nil, "can not get 'offset' tag")
	ov, ok = tsi.TagUint(wpk.TIDoffset)
	assert(ok, "can not convert 'offset' tag to value")
	assert(ov == offset, "'offset' tag is not equal to original value")

	// check up SIZE
	assert(tsi.Next(), "can not iterate to 'size'")
	assert(tsi.TID() == wpk.TIDsize, "tag #2 is not 'size'")
	tag = tsi.Tag()
	assert(tag != nil, "can not get 'size' tag")
	sv, ok = tsi.TagUint(wpk.TIDsize)
	assert(ok, "can not convert 'size' tag to value")
	assert(sv == size, "'size' tag is not equal to original value")

	// check up PATH
	assert(tsi.Next(), "can not iterate to 'path'")
	assert(tsi.TID() == wpk.TIDpath, "tag #3 is not 'path'")
	assert(tsi.TagLen() == len(fkey), "length of 'path' tag does not equal to original length")
	tag = tsi.Tag()
	assert(tag != nil, "can not get 'path' tag")
	str, ok = tag.TagStr()
	assert(ok, "can not convert 'path' tag to value")
	assert(str == fkey, "'path' tag is not equal to original value")

	// check up FID
	assert(tsi.Next(), "can not iterate to 'fid'")
	assert(tsi.TID() == wpk.TIDfid, "tag #4 is not 'fid'")
	tag = tsi.Tag()
	assert(tag != nil, "can not get 'fid' tag")
	fv, ok = tsi.TagUint(wpk.TIDfid)
	assert(ok, "can not convert 'fid' tag to value")
	assert(fv == fid, "'fid' tag is not equal to original value")

	// check up valid iterations finish
	assert(!tsi.Failed(), "content is broken")
	assert(!tsi.Next(), "iterator does not finished")
	assert(tsi.Passed(), "iterations does not reached till the end")
	assert(tsi.TID() == wpk.TIDnone, "tag ID in finished iterator should be 'none'")

	// check up 'Has'
	assert(ts.Has(wpk.TIDoffset) && ts.Has(wpk.TIDsize) && ts.Has(wpk.TIDfid) && ts.Has(wpk.TIDpath),
		"something does not pointed that should be present")
	assert(!ts.Has(wpk.TIDmd5), "'md5' tag is not set, but it's pointed that it present")

	// check up helpers functions
	fv, ok = ts.TagUint(wpk.TIDfid)
	assert(ok && fv == fid, "FID getter does not work correctly")
	ov, ok = ts.TagUint(wpk.TIDoffset)
	assert(ok && ov == offset, "FOffset getter does not work correctly")
	sv, ok = ts.TagUint(wpk.TIDsize)
	assert(ok && sv == size, "FSize getter does not work correctly")
	assert(ts.Path() == fkey, "'Path' function does not work correctly")
	assert(ts.Name() == path.Base(fkey), "'Name' function does not work correctly")

	// check up 'Set' and 'Del'
	ts, ok = ts.SetOk(wpk.TIDpath, wpk.StrTag(fkey))
	assert(!ok, "content of 'path' tag should be replaced by 'Set'")
	assert(ts.Num() == 4, "number of tags after replace 'path' must not be changed")
	assert(ts.Path() == wpk.ToSlash(fkey2), "'Set' function does not work correctly")
	ts, ok = ts.SetOk(wpk.TIDmime, wpk.StrTag(mime))
	assert(ok, "content of 'mime' tag should be added by 'Set'")
	assert(ts.Num() == 4+1, "number of tags after add 'mime' must be added by one")
	tag, ok = ts.Get(wpk.TIDmime)
	assert(ok, "can not get 'mime' tag content")
	str, _ = tag.TagStr()
	assert(str == mime, "'mime' tag is not equal to original value")
	ts, ok = ts.DelOk(wpk.TIDmime)
	assert(ok, "'mime' tag is not deleted")
	assert(!ts.Has(wpk.TIDmime), "'mime' tag must not be found after deletion")
	assert(ts.Num() == 4, "number of tags after delete 'mime' must be restored")
	ts, ok = ts.DelOk(wpk.TIDmime)
	assert(!ok, "'mime' tag can not be deleted again")
	assert(ts.Num() == 4, "number of tags after repeated delete 'mime' must be unchanged")
}

func ExampleTagsetIterator_Next() {
	var ts = wpk.TagsetRaw{}.
		Put(wpk.TIDpath, wpk.StrTag("picture.jpg")).
		Put(wpk.TIDmtime, wpk.TimeTag(time.Now())).
		Put(wpk.TIDmime, wpk.StrTag("image/jpeg"))
	var tsi = ts.Iterator()
	for tsi.Next() {
		fmt.Printf("tid=%d, len=%d\n", tsi.TID(), tsi.TagLen())
	}
	// Output:
	// tid=3, len=11
	// tid=5, len=12
	// tid=10, len=10
}

func ExampleTagsetIterator_Passed() {
	var slice = []byte{
		3, 0, 4, 0, 10, 0, 0, 0,
		4, 0, 12, 0, 115, 111, 109, 101, 102, 105, 108, 101, 46, 100, 97, 116,
	}
	var tsi = wpk.TagsetRaw(slice).Iterator()
	for tsi.Next() {
		// place some handler code here
	}
	fmt.Println(tsi.Passed())
	// Output: true
}

// The End.
