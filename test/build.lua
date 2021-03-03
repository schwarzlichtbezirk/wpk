
local pkgpath = path.join(bindir, "build.wpk") -- make package full file name on temporary directory

-- inits new package
local pkg = wpk.new()
pkg.automime = true -- put MIME type for each file if it is not given explicit
pkg.secret = "package-private-key" -- private key to sign cryptographic hashes for each file
pkg.crc64 = true -- generate CRC64 ISO code for each file
pkg.sha256 = true -- generate SHA256 hash for each file

-- open wpk-file for write
pkg:begin(pkgpath)
log("starts: "..pkgpath)

-- pack given file, then add keywords and author to tags set
local function packfile(kpath, fpath, keywords)
	pkg:putfile(kpath, fpath)
	pkg:addtags(kpath, {keywords=keywords, author="schwarzlichtbezirk"})
end

-- put images with keywords and author addition tags
local mediadir = path.join(scrdir, "media").."/"
packfile("bounty.jpg", mediadir.."bounty.jpg", "beach")
packfile("qarataslar.jpg", mediadir.."img1/qarataslar.jpg", "beach;rock")
packfile("claustral.jpg", mediadir.."img1/claustral.jpg", "beach;rock")
packfile("marble.jpg", mediadir.."img2/marble.jpg", "beach")
packfile("uzunji.jpg", mediadir.."img2/uzunji.jpg", "rock")

-- put file created from given string
pkg:putdata("sample.txt", "The quick brown fox jumps over the lazy dog")
pkg:settags("sample.txt", {mime="text/plain;charset=utf-8", keywords="fox;dog"})

-- make 2 file name aliases to 1 file
pkg:putalias("claustral.jpg", "jasper.jpg")

log(string.format("qarataşlar file size: %d bytes", pkg:filesize("qarataslar.jpg")))
log(string.format("total files size sum: %d bytes", pkg:sumsize()))
log(string.format("packaged: %d files to %d aliases", pkg.recnum, pkg.tagnum))

-- write records table, tags table and finalize wpk-file
pkg:complete()

log "done."
