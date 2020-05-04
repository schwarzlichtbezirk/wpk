package main

import (
	"crypto/hmac"
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"hash/crc32"
	"io"
	"path/filepath"

	"github.com/schwarzlichtbezirk/wpk"
)

func (pack *LuaPackage) adjusttagset(r io.ReadSeeker, tags wpk.TagSet) (err error) {
	if _, ok := tags[wpk.AID_mime]; !ok && pack.automime {
		var fname, _ = tags.String(wpk.AID_name)
		if ct, ok := mimeext[filepath.Ext(fname)]; ok {
			tags.SetString(wpk.AID_mime, ct)
		}
	}

	if _, ok := tags[wpk.AID_CRC32C]; !ok && pack.crc32 {
		if _, err = r.Seek(0, io.SeekStart); err != nil {
			return
		}
		var h = crc32.New(crc32.MakeTable(crc32.Castagnoli))
		if _, err = io.Copy(h, r); err != nil {
			return
		}
		tags[wpk.AID_CRC32C] = h.Sum(nil)
	}

	if _, ok := tags[wpk.AID_MD5]; !ok && pack.md5 {
		if _, err = r.Seek(0, io.SeekStart); err != nil {
			return
		}
		var mac = hmac.New(md5.New, []byte(pack.secret))
		if _, err = io.Copy(mac, r); err != nil {
			return
		}
		tags[wpk.AID_MD5] = mac.Sum(nil)
	}

	if _, ok := tags[wpk.AID_SHA1]; !ok && pack.sha1 {
		if _, err = r.Seek(0, io.SeekStart); err != nil {
			return
		}
		var mac = hmac.New(sha1.New, []byte(pack.secret))
		if _, err = io.Copy(mac, r); err != nil {
			return
		}
		tags[wpk.AID_SHA1] = mac.Sum(nil)
	}

	if _, ok := tags[wpk.AID_SHA224]; !ok && pack.sha224 {
		if _, err = r.Seek(0, io.SeekStart); err != nil {
			return
		}
		var mac = hmac.New(sha256.New224, []byte(pack.secret))
		if _, err = io.Copy(mac, r); err != nil {
			return
		}
		tags[wpk.AID_SHA224] = mac.Sum(nil)
	}

	if _, ok := tags[wpk.AID_SHA256]; !ok && pack.sha256 {
		if _, err = r.Seek(0, io.SeekStart); err != nil {
			return
		}
		var mac = hmac.New(sha256.New, []byte(pack.secret))
		if _, err = io.Copy(mac, r); err != nil {
			return
		}
		tags[wpk.AID_SHA256] = mac.Sum(nil)
	}

	return
}

// The End.
