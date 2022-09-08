package luawpk

import (
	"crypto/hmac"
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"hash/crc32"
	"hash/crc64"
	"io"
	"mime"
	"path"
	"strings"

	"github.com/h2non/filetype"
	"github.com/schwarzlichtbezirk/wpk"
)

func adjustmime(ts *wpk.TagsetRaw, r io.ReadSeeker, skip bool) (err error) {
	var ok bool
	if ok = ts.Has(wpk.TIDmime); ok || skip {
		return
	}
	var ext = strings.ToLower(path.Ext(ts.Path()))
	var ctype string
	if ctype = mime.TypeByExtension(ext); ctype == "" {
		if ctype, ok = MimeExt[ext]; !ok {
			if _, err = r.Seek(0, io.SeekStart); err != nil {
				return
			}
			if kind, err := filetype.MatchReader(r); err == nil && kind != filetype.Unknown {
				ctype = kind.MIME.Value
			} else {
				ctype = "application/octet-stream"
			}
		}
	}
	ts.Put(wpk.TIDmime, wpk.StrTag(ctype))
	return
}

func adjustcrc32c(ts *wpk.TagsetRaw, r io.ReadSeeker, skip bool) (err error) {
	if ok := ts.Has(wpk.TIDcrc32c); ok || skip {
		return
	}
	if _, err = r.Seek(0, io.SeekStart); err != nil {
		return
	}
	var h = crc32.New(crc32.MakeTable(crc32.Castagnoli))
	if _, err = io.Copy(h, r); err != nil {
		return
	}
	ts.Put(wpk.TIDcrc32c, h.Sum(nil))
	return
}

func adjustcrc64iso(ts *wpk.TagsetRaw, r io.ReadSeeker, skip bool) (err error) {
	if ok := ts.Has(wpk.TIDcrc64iso); ok && skip {
		return
	}
	if _, err = r.Seek(0, io.SeekStart); err != nil {
		return
	}
	var h = crc64.New(crc64.MakeTable(crc64.ISO))
	if _, err = io.Copy(h, r); err != nil {
		return
	}
	ts.Put(wpk.TIDcrc64iso, h.Sum(nil))
	return
}

func adjustmd5(ts *wpk.TagsetRaw, r io.ReadSeeker, skip bool, secret []byte) (err error) {
	if ok := ts.Has(wpk.TIDmd5); ok && skip {
		return
	}
	if _, err = r.Seek(0, io.SeekStart); err != nil {
		return
	}
	var mac = hmac.New(md5.New, secret)
	if _, err = io.Copy(mac, r); err != nil {
		return
	}
	ts.Put(wpk.TIDmd5, mac.Sum(nil))
	return
}

func adjustsha1(ts *wpk.TagsetRaw, r io.ReadSeeker, skip bool, secret []byte) (err error) {
	if ok := ts.Has(wpk.TIDsha1); ok && skip {
		return
	}
	if _, err = r.Seek(0, io.SeekStart); err != nil {
		return
	}
	var mac = hmac.New(sha1.New, secret)
	if _, err = io.Copy(mac, r); err != nil {
		return
	}
	ts.Put(wpk.TIDsha1, mac.Sum(nil))
	return
}

func adjustsha224(ts *wpk.TagsetRaw, r io.ReadSeeker, skip bool, secret []byte) (err error) {
	if ok := ts.Has(wpk.TIDsha224); ok && skip {
		return
	}
	if _, err = r.Seek(0, io.SeekStart); err != nil {
		return
	}
	var mac = hmac.New(sha256.New224, secret)
	if _, err = io.Copy(mac, r); err != nil {
		return
	}
	ts.Put(wpk.TIDsha224, mac.Sum(nil))
	return
}

func adjustsha256(ts *wpk.TagsetRaw, r io.ReadSeeker, skip bool, secret []byte) (err error) {
	if ok := ts.Has(wpk.TIDsha256); ok && skip {
		return
	}
	if _, err = r.Seek(0, io.SeekStart); err != nil {
		return
	}
	var mac = hmac.New(sha256.New, secret)
	if _, err = io.Copy(mac, r); err != nil {
		return
	}
	ts.Put(wpk.TIDsha256, mac.Sum(nil))
	return
}

func adjustsha384(ts *wpk.TagsetRaw, r io.ReadSeeker, skip bool, secret []byte) (err error) {
	if ok := ts.Has(wpk.TIDsha384); ok && skip {
		return
	}
	if _, err = r.Seek(0, io.SeekStart); err != nil {
		return
	}
	var mac = hmac.New(sha512.New384, secret)
	if _, err = io.Copy(mac, r); err != nil {
		return
	}
	ts.Put(wpk.TIDsha384, mac.Sum(nil))
	return
}

func adjustsha512(ts *wpk.TagsetRaw, r io.ReadSeeker, skip bool, secret []byte) (err error) {
	if ok := ts.Has(wpk.TIDsha512); ok && skip {
		return
	}
	if _, err = r.Seek(0, io.SeekStart); err != nil {
		return
	}
	var mac = hmac.New(sha512.New, secret)
	if _, err = io.Copy(mac, r); err != nil {
		return
	}
	ts.Put(wpk.TIDsha512, mac.Sum(nil))
	return
}

func (pack *LuaPackage) adjusttagset(r io.ReadSeeker, ts *wpk.TagsetRaw) (err error) {
	if err = adjustmime(ts, r, !pack.automime); err != nil {
		return
	}

	if pack.nolink {
		ts.Del(wpk.TIDlink)
	}

	if err = adjustcrc32c(ts, r, !pack.crc32); err != nil {
		return
	}

	if err = adjustcrc64iso(ts, r, !pack.crc64); err != nil {
		return
	}

	if err = adjustmd5(ts, r, !pack.md5, pack.secret); err != nil {
		return
	}

	if err = adjustsha1(ts, r, !pack.sha1, pack.secret); err != nil {
		return
	}

	if err = adjustsha224(ts, r, !pack.sha224, pack.secret); err != nil {
		return
	}

	if err = adjustsha256(ts, r, !pack.sha256, pack.secret); err != nil {
		return
	}

	if err = adjustsha384(ts, r, !pack.sha384, pack.secret); err != nil {
		return
	}

	if err = adjustsha512(ts, r, !pack.sha512, pack.secret); err != nil {
		return
	}

	return
}

// The End.
