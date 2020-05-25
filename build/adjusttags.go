package main

import (
	"crypto/hmac"
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"hash/crc32"
	"hash/crc64"
	"io"
	"path/filepath"

	"github.com/schwarzlichtbezirk/wpk"
)

func (pack *LuaPackage) adjusttagset(r io.ReadSeeker, tags wpk.Tagset) (err error) {
	if _, ok := tags[wpk.TID_mime]; !ok && pack.automime {
		var kpath, _ = tags.String(wpk.TID_path)
		if ct, ok := mimeext[filepath.Ext(kpath)]; ok {
			tags[wpk.TID_mime] = wpk.TagString(ct)
		}
	}

	if _, ok := tags[wpk.TID_CRC32C]; !ok && pack.crc32 {
		if _, err = r.Seek(0, io.SeekStart); err != nil {
			return
		}
		var h = crc32.New(crc32.MakeTable(crc32.Castagnoli))
		if _, err = io.Copy(h, r); err != nil {
			return
		}
		tags[wpk.TID_CRC32C] = h.Sum(nil)
	}

	if _, ok := tags[wpk.TID_CRC64ISO]; !ok && pack.crc64 {
		if _, err = r.Seek(0, io.SeekStart); err != nil {
			return
		}
		var h = crc64.New(crc64.MakeTable(crc64.ISO))
		if _, err = io.Copy(h, r); err != nil {
			return
		}
		tags[wpk.TID_CRC64ISO] = h.Sum(nil)
	}

	if _, ok := tags[wpk.TID_MD5]; !ok && pack.md5 {
		if _, err = r.Seek(0, io.SeekStart); err != nil {
			return
		}
		var mac = hmac.New(md5.New, []byte(pack.secret))
		if _, err = io.Copy(mac, r); err != nil {
			return
		}
		tags[wpk.TID_MD5] = mac.Sum(nil)
	}

	if _, ok := tags[wpk.TID_SHA1]; !ok && pack.sha1 {
		if _, err = r.Seek(0, io.SeekStart); err != nil {
			return
		}
		var mac = hmac.New(sha1.New, []byte(pack.secret))
		if _, err = io.Copy(mac, r); err != nil {
			return
		}
		tags[wpk.TID_SHA1] = mac.Sum(nil)
	}

	if _, ok := tags[wpk.TID_SHA224]; !ok && pack.sha224 {
		if _, err = r.Seek(0, io.SeekStart); err != nil {
			return
		}
		var mac = hmac.New(sha256.New224, []byte(pack.secret))
		if _, err = io.Copy(mac, r); err != nil {
			return
		}
		tags[wpk.TID_SHA224] = mac.Sum(nil)
	}

	if _, ok := tags[wpk.TID_SHA256]; !ok && pack.sha256 {
		if _, err = r.Seek(0, io.SeekStart); err != nil {
			return
		}
		var mac = hmac.New(sha256.New, []byte(pack.secret))
		if _, err = io.Copy(mac, r); err != nil {
			return
		}
		tags[wpk.TID_SHA256] = mac.Sum(nil)
	}

	if _, ok := tags[wpk.TID_SHA384]; !ok && pack.sha384 {
		if _, err = r.Seek(0, io.SeekStart); err != nil {
			return
		}
		var mac = hmac.New(sha512.New384, []byte(pack.secret))
		if _, err = io.Copy(mac, r); err != nil {
			return
		}
		tags[wpk.TID_SHA384] = mac.Sum(nil)
	}

	if _, ok := tags[wpk.TID_SHA512]; !ok && pack.sha512 {
		if _, err = r.Seek(0, io.SeekStart); err != nil {
			return
		}
		var mac = hmac.New(sha512.New, []byte(pack.secret))
		if _, err = io.Copy(mac, r); err != nil {
			return
		}
		tags[wpk.TID_SHA512] = mac.Sum(nil)
	}

	return
}

// The End.
