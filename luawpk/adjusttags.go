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
	"net/http"
	"path/filepath"

	"github.com/schwarzlichtbezirk/wpk"
)

const sniffLen = 512

func (pack *LuaPackage) adjusttagset(r io.ReadSeeker, tags wpk.Tagset) (err error) {
	if _, ok := tags[wpk.TIDmime]; !ok && pack.automime {
		var kext = filepath.Ext(tags.Path())
		var ctype = mime.TypeByExtension(kext)
		if ctype == "" {
			var ok bool
			if ctype, ok = MimeExt[kext]; !ok {
				// rewind to file start
				if _, err = r.Seek(0, io.SeekStart); err != nil {
					return
				}
				// read a chunk to decide between utf-8 text and binary
				var buf [sniffLen]byte
				var n int
				if n, err = io.ReadFull(r, buf[:]); err != nil {
					if err == io.ErrUnexpectedEOF {
						err = nil
					} else {
						return
					}
				}
				ctype = http.DetectContentType(buf[:n])
			}
		}
		tags[wpk.TIDmime] = wpk.TagString(ctype)
	}

	if pack.nolink {
		delete(tags, wpk.TIDlink)
	}

	if _, ok := tags[wpk.TIDcrc32c]; !ok && pack.crc32 {
		if _, err = r.Seek(0, io.SeekStart); err != nil {
			return
		}
		var h = crc32.New(crc32.MakeTable(crc32.Castagnoli))
		if _, err = io.Copy(h, r); err != nil {
			return
		}
		tags[wpk.TIDcrc32c] = h.Sum(nil)
	}

	if _, ok := tags[wpk.TIDcrc64iso]; !ok && pack.crc64 {
		if _, err = r.Seek(0, io.SeekStart); err != nil {
			return
		}
		var h = crc64.New(crc64.MakeTable(crc64.ISO))
		if _, err = io.Copy(h, r); err != nil {
			return
		}
		tags[wpk.TIDcrc64iso] = h.Sum(nil)
	}

	if _, ok := tags[wpk.TIDmd5]; !ok && pack.md5 {
		if _, err = r.Seek(0, io.SeekStart); err != nil {
			return
		}
		var mac = hmac.New(md5.New, []byte(pack.secret))
		if _, err = io.Copy(mac, r); err != nil {
			return
		}
		tags[wpk.TIDmd5] = mac.Sum(nil)
	}

	if _, ok := tags[wpk.TIDsha1]; !ok && pack.sha1 {
		if _, err = r.Seek(0, io.SeekStart); err != nil {
			return
		}
		var mac = hmac.New(sha1.New, []byte(pack.secret))
		if _, err = io.Copy(mac, r); err != nil {
			return
		}
		tags[wpk.TIDsha1] = mac.Sum(nil)
	}

	if _, ok := tags[wpk.TIDsha224]; !ok && pack.sha224 {
		if _, err = r.Seek(0, io.SeekStart); err != nil {
			return
		}
		var mac = hmac.New(sha256.New224, []byte(pack.secret))
		if _, err = io.Copy(mac, r); err != nil {
			return
		}
		tags[wpk.TIDsha224] = mac.Sum(nil)
	}

	if _, ok := tags[wpk.TIDsha256]; !ok && pack.sha256 {
		if _, err = r.Seek(0, io.SeekStart); err != nil {
			return
		}
		var mac = hmac.New(sha256.New, []byte(pack.secret))
		if _, err = io.Copy(mac, r); err != nil {
			return
		}
		tags[wpk.TIDsha256] = mac.Sum(nil)
	}

	if _, ok := tags[wpk.TIDsha384]; !ok && pack.sha384 {
		if _, err = r.Seek(0, io.SeekStart); err != nil {
			return
		}
		var mac = hmac.New(sha512.New384, []byte(pack.secret))
		if _, err = io.Copy(mac, r); err != nil {
			return
		}
		tags[wpk.TIDsha384] = mac.Sum(nil)
	}

	if _, ok := tags[wpk.TIDsha512]; !ok && pack.sha512 {
		if _, err = r.Seek(0, io.SeekStart); err != nil {
			return
		}
		var mac = hmac.New(sha512.New, []byte(pack.secret))
		if _, err = io.Copy(mac, r); err != nil {
			return
		}
		tags[wpk.TIDsha512] = mac.Sum(nil)
	}

	return
}

// The End.
