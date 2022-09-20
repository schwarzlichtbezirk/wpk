
--[[
Package can be formed by several steps. At first step package is created
and placed there first portion of files. Then this existing package opens
again and then placed there next portion of files. This script is the
first step with package creation.
]]

local pkgpath = path.join(tmpdir, "steps.wpk") -- make package full file name on temporary directory

print ""
log "starts step 1"

-- inits new package
local pkg = wpk.new()
pkg.label = "two-steps" -- image label
pkg.automime = true -- put MIME type for each file if it is not given explicit
pkg.secret = "package-private-key" -- private key to sign cryptographic hashes for each file
pkg.crc32 = true -- generate CRC32 Castagnoli code for each file
pkg.md5 = true -- generate MD5 hash for each file

-- pack given file with common preset
local n = 0
local function packfile(fkey, keywords)
	n = n + 1
	pkg:putfile(fkey, path.join(scrdir, "media", fkey))
	pkg:addtags(fkey, {fid=n, keywords=keywords, author="schwarzlichtbezirk"})
	log(string.format("#%d file %s, crc=%s", n, fkey,
		tostring(pkg:gettag(fkey, "crc32"))))
end

-- open wpk-file for write
pkg:begin(pkgpath)
log("create: "..pkgpath)

-- put images with keywords and author addition tags
packfile("bounty.jpg", "beach")
packfile("img1/Qarataşlar.jpg", "beach;rock")
packfile("img1/claustral.jpg", "beach;rock")

log(string.format("packed %d files, fft %d bytes, data %s bytes", pkg.recnum, pkg.fftsize, pkg.datasize))

-- write records table, tags table and finalize wpk-file
pkg:finalize()

log(tostring(pkg))
log "done step 1."
