
local pkgpath = path.join(tmpdir, "steps.wpk") -- make package full file name on temporary directory

print ""
log "starts step 1"

-- inits new package
local pkg = wpk.new()
pkg.automime = true -- put MIME type for each file if it is not given explicit
pkg.secret = "package-private-key" -- private key to sign cryptographic hashes for each file
pkg.crc32 = true -- generate CRC32 Castagnoli code for each file
pkg.md5 = true -- generate MD5 hash for each file

-- pack given file with common preset
local function packfile(kpath, keywords)
	pkg:putfile(kpath, path.join(scrdir, "media", kpath))
	pkg:addtags(kpath, {keywords=keywords, [104]="schwarzlichtbezirk"})
	log(string.format("#%d file %s, crc=%s",
		pkg:gettag(kpath, "fid").uint32, kpath,
		tostring(pkg:gettag(kpath, "crc32"))))
end

-- open wpk-file for write
pkg:begin(pkgpath)
log("create: "..pkgpath)

-- put images with keywords and author addition tags
packfile("bounty.jpg", "beach")
packfile("img1/qarataslar.jpg", "beach;rock")
packfile("img1/claustral.jpg", "beach;rock")

log(string.format("packaged %d files on sum %d bytes", pkg.recnum, pkg.datasize))

-- write records table, tags table and finalize wpk-file
pkg:complete()

log "done step 1."
