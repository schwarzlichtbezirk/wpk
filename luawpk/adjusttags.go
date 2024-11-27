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

	"github.com/h2non/filetype"

	"github.com/schwarzlichtbezirk/wpk"
	"github.com/schwarzlichtbezirk/wpk/util"
)

func adjustmime(ts wpk.TagsetRaw, r io.ReadSeeker, skip bool) (wpk.TagsetRaw, error) {
	var err error
	var ok bool
	if skip || ts.Has(wpk.TIDmime) {
		return ts, err
	}
	var ext = util.ToLower(path.Ext(ts.Path()))
	var ctype string
	if ctype = mime.TypeByExtension(ext); ctype == "" {
		if ctype, ok = MimeExt[ext]; !ok {
			if _, err = r.Seek(0, io.SeekStart); err != nil {
				return ts, err
			}
			if kind, err := filetype.MatchReader(r); err == nil && kind != filetype.Unknown {
				ctype = kind.MIME.Value
			} else {
				ctype = "application/octet-stream"
			}
		}
	}
	return ts.Put(wpk.TIDmime, wpk.StrTag(ctype)), nil
}

func adjustcrc32c(ts wpk.TagsetRaw, r io.ReadSeeker, skip bool) (wpk.TagsetRaw, error) {
	var err error
	if skip || ts.Has(wpk.TIDcrc32c) {
		return ts, err
	}
	if _, err = r.Seek(0, io.SeekStart); err != nil {
		return ts, err
	}
	var h = crc32.New(crc32.MakeTable(crc32.Castagnoli))
	if _, err = io.Copy(h, r); err != nil {
		return ts, err
	}
	return ts.Put(wpk.TIDcrc32c, h.Sum(nil)), nil
}

func adjustcrc64iso(ts wpk.TagsetRaw, r io.ReadSeeker, skip bool) (wpk.TagsetRaw, error) {
	var err error
	if skip || ts.Has(wpk.TIDcrc64iso) {
		return ts, err
	}
	if _, err = r.Seek(0, io.SeekStart); err != nil {
		return ts, err
	}
	var h = crc64.New(crc64.MakeTable(crc64.ISO))
	if _, err = io.Copy(h, r); err != nil {
		return ts, err
	}
	return ts.Put(wpk.TIDcrc64iso, h.Sum(nil)), nil
}

func adjustmd5(ts wpk.TagsetRaw, r io.ReadSeeker, skip bool, secret []byte) (wpk.TagsetRaw, error) {
	var err error
	if skip || ts.Has(wpk.TIDmd5) {
		return ts, err
	}
	if _, err = r.Seek(0, io.SeekStart); err != nil {
		return ts, err
	}
	var mac = hmac.New(md5.New, secret)
	if _, err = io.Copy(mac, r); err != nil {
		return ts, err
	}
	return ts.Put(wpk.TIDmd5, mac.Sum(nil)), nil
}

func adjustsha1(ts wpk.TagsetRaw, r io.ReadSeeker, skip bool, secret []byte) (wpk.TagsetRaw, error) {
	var err error
	if skip || ts.Has(wpk.TIDsha1) {
		return ts, err
	}
	if _, err = r.Seek(0, io.SeekStart); err != nil {
		return ts, err
	}
	var mac = hmac.New(sha1.New, secret)
	if _, err = io.Copy(mac, r); err != nil {
		return ts, err
	}
	return ts.Put(wpk.TIDsha1, mac.Sum(nil)), nil
}

func adjustsha224(ts wpk.TagsetRaw, r io.ReadSeeker, skip bool, secret []byte) (wpk.TagsetRaw, error) {
	var err error
	if skip || ts.Has(wpk.TIDsha224) {
		return ts, err
	}
	if _, err = r.Seek(0, io.SeekStart); err != nil {
		return ts, err
	}
	var mac = hmac.New(sha256.New224, secret)
	if _, err = io.Copy(mac, r); err != nil {
		return ts, err
	}
	return ts.Put(wpk.TIDsha224, mac.Sum(nil)), nil
}

func adjustsha256(ts wpk.TagsetRaw, r io.ReadSeeker, skip bool, secret []byte) (wpk.TagsetRaw, error) {
	var err error
	if skip || ts.Has(wpk.TIDsha256) {
		return ts, err
	}
	if _, err = r.Seek(0, io.SeekStart); err != nil {
		return ts, err
	}
	var mac = hmac.New(sha256.New, secret)
	if _, err = io.Copy(mac, r); err != nil {
		return ts, err
	}
	return ts.Put(wpk.TIDsha256, mac.Sum(nil)), nil
}

func adjustsha384(ts wpk.TagsetRaw, r io.ReadSeeker, skip bool, secret []byte) (wpk.TagsetRaw, error) {
	var err error
	if skip || ts.Has(wpk.TIDsha384) {
		return ts, err
	}
	if _, err = r.Seek(0, io.SeekStart); err != nil {
		return ts, err
	}
	var mac = hmac.New(sha512.New384, secret)
	if _, err = io.Copy(mac, r); err != nil {
		return ts, err
	}
	return ts.Put(wpk.TIDsha384, mac.Sum(nil)), nil
}

func adjustsha512(ts wpk.TagsetRaw, r io.ReadSeeker, skip bool, secret []byte) (wpk.TagsetRaw, error) {
	var err error
	if skip || ts.Has(wpk.TIDsha512) {
		return ts, err
	}
	if _, err = r.Seek(0, io.SeekStart); err != nil {
		return ts, err
	}
	var mac = hmac.New(sha512.New, secret)
	if _, err = io.Copy(mac, r); err != nil {
		return ts, err
	}
	return ts.Put(wpk.TIDsha512, mac.Sum(nil)), nil
}

func (pkg *LuaPackage) adjusttagset(r io.ReadSeeker, ts wpk.TagsetRaw) (wpk.TagsetRaw, error) {
	var err error

	if pkg.autofid && !ts.Has(wpk.TIDfid) {
		pkg.fidcount++
		ts = ts.Put(wpk.TIDfid, wpk.UintTag(pkg.fidcount))
	}

	if ts, err = adjustmime(ts, r, !pkg.automime); err != nil {
		return ts, err
	}

	if ts, err = adjustcrc32c(ts, r, !pkg.crc32); err != nil {
		return ts, err
	}

	if ts, err = adjustcrc64iso(ts, r, !pkg.crc64); err != nil {
		return ts, err
	}

	if ts, err = adjustmd5(ts, r, !pkg.md5, pkg.secret); err != nil {
		return ts, err
	}

	if ts, err = adjustsha1(ts, r, !pkg.sha1, pkg.secret); err != nil {
		return ts, err
	}

	if ts, err = adjustsha224(ts, r, !pkg.sha224, pkg.secret); err != nil {
		return ts, err
	}

	if ts, err = adjustsha256(ts, r, !pkg.sha256, pkg.secret); err != nil {
		return ts, err
	}

	if ts, err = adjustsha384(ts, r, !pkg.sha384, pkg.secret); err != nil {
		return ts, err
	}

	if ts, err = adjustsha512(ts, r, !pkg.sha512, pkg.secret); err != nil {
		return ts, err
	}

	return ts, err
}

// The End.
