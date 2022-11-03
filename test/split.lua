
--[[
Package can be used as files database without deleting. For this task
package stored at two splitted files. First is file with header and
tagset table. Second file is set of datablocks with nested files content.
So, if program abnormal failure is happens, package contains state with
files on last flush-function call, or package finalize.
]]

local pkgpath = path.join(tmpdir, "build.wpt") -- make package tagset full file name on temporary directory
local datpath = path.join(tmpdir, "build.wpf") -- make package data full file name on temporary directory

-- inits new package
local pkg = wpk.new()
pkg.automime = true -- put MIME type for each file if it is not given explicit
pkg.secret = "package-private-key" -- private key to sign cryptographic hashes for each file
pkg.crc64 = true -- generate CRC64 ISO code for each file
pkg.sha384 = true -- generate SHA-384 hash for each file
pkg:setinfo{ -- setup package info
	label="splitted-package",
	link="github.com/schwarzlichtbezirk/wpk",
	keywords="thumb;thumbnail;photo",
	category="image",
	version="v3.4",
}

-- open wpk-file for write
pkg:begin(pkgpath, datpath)
log("starts header file: "..pkgpath)
log("starts data file:   "..datpath)

-- pack given file, then add keywords and author to tagset
local function packfile(fkey, keywords)
	pkg:putfile(fkey, path.join(scrdir, "media", fkey))
	pkg:addtags(fkey, {keywords=keywords, author="schwarzlichtbezirk"})
end

-- put images with keywords and author addition tags
local mediadir = path.join(scrdir, "media").."/"

-- workflow part 1
packfile("bounty.jpg", "beach")
packfile("img1/Qarataşlar.jpg", "beach;rock")
packfile("img1/claustral.jpg", "beach;rock")
pkg:flush() -- after this point process can be broken, and files above will remains.

-- workflow part 2
packfile("img2/marble.jpg", "beach")
packfile("img2/Uzuncı.jpg", "rock")
pkg:flush() -- after this point process can be broken, and files above will remains.

-- workflow part 3
pkg:putdata("sample.txt", "The quick brown fox jumps over the lazy dog")
pkg:settags("sample.txt", {mime="text/plain;charset=utf-8", keywords="fox;dog"})
pkg:putalias("img1/claustral.jpg", "jasper.jpg") -- make 2 file name aliases to 1 file

log(string.format("'Qarataşlar' file size: %d bytes", pkg:filesize("img1/Qarataşlar.jpg")))
log(string.format("total files size sum: %d bytes", pkg:sumsize()))
log(string.format("packaged: %d files to %d aliases", pkg.recnum, pkg.tagnum))

-- finalize wpk-file
pkg:finalize()

log(tostring(pkg))
log "done."
