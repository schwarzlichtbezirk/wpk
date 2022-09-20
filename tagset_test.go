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
		kpath1 = `Dir\FileName.Ext`
		kpath2 = `DIR\FILENAME.EXT`
		fkey   = `dir/filename.ext`
		mime   = "image/jpeg"
	)
	var ts = wpk.MakeTagset(nil, tidsz, tagsz).
		Put(wpk.TIDoffset, wpk.UintTag(offset)).
		Put(wpk.TIDsize, wpk.UintTag(size)).
		Put(wpk.TIDfid, wpk.UintTag(fid)).
		Put(wpk.TIDpath, wpk.StrTag(wpk.ToSlash(kpath1)))
	var tsi = ts.Iterator()

	var (
		tag wpk.TagRaw
		ok  bool
		fv  wpk.Uint
		ov  wpk.Uint
		sv  wpk.Uint
		str string
	)

	for _, check := range []struct {
		cond func() bool
		msg  string
	}{
		{func() bool { return wpk.Normalize(kpath1) != fkey || wpk.Normalize(kpath2) != fkey },
			"normalize test failed",
		},
		{func() bool { return tsi.TID() != wpk.TIDnone },
			"tag ID in created iterator should be 'none'",
		},
		{func() bool { return ts.Num() != 4 },
			"wrong number of tags",
		},

		// check up OFFSET
		{func() bool { return !tsi.Next() },
			"can not iterate to 'offset'",
		},
		{func() bool { return tsi.TID() != wpk.TIDoffset },
			"tag #2 is not 'offset'",
		},
		{func() bool { tag = tsi.Tag(); return tag == nil },
			"can not get 'offset' tag",
		},
		{func() bool { ov, ok = tsi.TagUint(wpk.TIDoffset); return !ok },
			"can not convert 'offset' tag to value",
		},
		{func() bool { return ov != offset },
			"'offset' tag is not equal to original value",
		},

		// check up SIZE
		{func() bool { return !tsi.Next() },
			"can not iterate to 'size'",
		},
		{func() bool { return tsi.TID() != wpk.TIDsize },
			"tag #3 is not 'size'",
		},
		{func() bool { tag = tsi.Tag(); return tag == nil },
			"can not get 'size' tag",
		},
		{func() bool { sv, ok = tsi.TagUint(wpk.TIDsize); return !ok },
			"can not convert 'size' tag to value",
		},
		{func() bool { return sv != size },
			"'size' tag is not equal to original value",
		},

		// check up FID
		{func() bool { return !tsi.Next() },
			"can not iterate to 'fid'",
		},
		{func() bool { return tsi.TID() != wpk.TIDfid },
			"tag #1 is not 'fid'",
		},
		{func() bool { tag = tsi.Tag(); return tag == nil },
			"can not get 'fid' tag",
		},
		{func() bool { fv, ok = tsi.TagUint(wpk.TIDfid); return !ok },
			"can not convert 'fid' tag to value",
		},
		{func() bool { return fv != fid },
			"'fid' tag is not equal to original value",
		},

		// check up PATH
		{func() bool { return !tsi.Next() },
			"can not iterate to 'path'",
		},
		{func() bool { return tsi.TID() != wpk.TIDpath },
			"tag #4 is not 'path'",
		},
		{func() bool { return tsi.TagLen() != len(kpath1) },
			"length of 'path' tag does not equal to original length",
		},
		{func() bool { tag = tsi.Tag(); return tag == nil },
			"can not get 'path' tag",
		},
		{func() bool { str, ok = tag.TagStr(); return !ok },
			"can not convert 'path' tag to value",
		},
		{func() bool { return str != wpk.ToSlash(kpath1) },
			"'path' tag is not equal to original value",
		},

		// check up valid iterations finish
		{func() bool { return tsi.Failed() },
			"content is broken",
		},
		{func() bool { return tsi.Next() },
			"iterator does not finished",
		},
		{func() bool { return !tsi.Passed() },
			"iterations does not reached till the end",
		},
		{func() bool { return tsi.TID() != wpk.TIDnone },
			"tag ID in finished iterator should be 'none'",
		},

		// check up 'Has'
		{func() bool {
			return !(ts.Has(wpk.TIDoffset) && ts.Has(wpk.TIDsize) && ts.Has(wpk.TIDfid) && ts.Has(wpk.TIDpath))
		},
			"something does not pointed that should be present",
		},
		{func() bool { return ts.Has(wpk.TIDmd5) },
			"'md5' tag is not set, but it's pointed that it present",
		},

		// check up helpers functions
		{func() bool {
			v, ok := ts.TagUint(wpk.TIDfid)
			return !ok || v != fid
		},
			"FID getter does not work correctly",
		},
		{func() bool {
			v, ok := ts.TagUint(wpk.TIDoffset)
			return !ok || v != offset
		},
			"FOffset getter does not work correctly",
		},
		{func() bool {
			v, ok := ts.TagUint(wpk.TIDsize)
			return !ok || v != size
		},
			"FSize getter does not work correctly",
		},
		{func() bool { return ts.Path() != wpk.ToSlash(kpath1) },
			"'Path' function does not work correctly",
		},
		{func() bool { return ts.Name() != path.Base(wpk.ToSlash(kpath1)) },
			"'Name' function does not work correctly",
		},

		// check up 'Set' and 'Del'
		{func() bool { return ts.Set(wpk.TIDpath, wpk.StrTag(wpk.ToSlash(kpath2))) },
			"content of 'path' tag should be replaced by 'Set'",
		},
		{func() bool { return ts.Num() != 4 },
			"number of tags after replace 'path' must not be changed",
		},
		{func() bool { return ts.Path() != wpk.ToSlash(kpath2) },
			"'Set' function does not work correctly",
		},
		{func() bool { return !ts.Set(wpk.TIDmime, wpk.StrTag(mime)) },
			"content of 'mime' tag should be added by 'Set'",
		},
		{func() bool { return ts.Num() != 4+1 },
			"number of tags after add 'mime' must be added by one",
		},
		{func() bool { tag, ok = ts.Get(wpk.TIDmime); return !ok },
			"can not get 'mime' tag content",
		},
		{func() bool { str, _ = tag.TagStr(); return str != mime },
			"'mime' tag is not equal to original value",
		},
		{func() bool { return !ts.Del(wpk.TIDmime) },
			"'mime' tag is not deleted",
		},
		{func() bool { return ts.Has(wpk.TIDmime) },
			"'mime' tag must not be found after deletion",
		},
		{func() bool { return ts.Num() != 4 },
			"number of tags after delete 'mime' must be restored",
		},
		{func() bool { return ts.Del(wpk.TIDmime) },
			"'mime' tag can not be deleted again",
		},
		{func() bool { return ts.Num() != 4 },
			"number of tags after repeated delete 'mime' must be unchanged",
		},
	} {
		if check.cond() {
			t.Fatal(check.msg)
		}
	}
}

func ExampleTagsetIterator_Next() {
	var ts = wpk.MakeTagset(nil, tidsz, tagsz).
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
	var tsi = wpk.MakeTagset(slice, tidsz, tagsz).Iterator()
	for tsi.Next() {
		// place some handler code here
	}
	fmt.Println(tsi.Passed())
	// Output: true
}

// The End.
