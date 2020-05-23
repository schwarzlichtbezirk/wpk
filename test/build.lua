
local pkgpath = scrdir.."build.wpk" -- make package full file name on script directory
log("starts: "..pkgpath)

-- inits new package
local pkg = wpk.new()
pkg.automime = true -- put MIME type for each file if it is not given explicit
pkg.secret = "package-private-key" -- private key to sign cryptographic hashes for each file
pkg.crc64 = true -- generate CRC64 ISO code for each file
pkg.sha256 = true -- generate SHA256 hash for each file

-- open wpk-file for write
pkg:begin(pkgpath)

-- put images with keywords and author addition tags
local mediadir = scrdir.."media/"
local auth = "schwarzlichtbezirk"
pkg:putfile({name="bounty.jpg", keywords="beach", author=auth}, mediadir.."bounty.jpg")
pkg:putfile({name="qarataslar.jpg", keywords="beach;rock", author=auth}, mediadir.."img1/qarataslar.jpg")
pkg:putfile({name="claustral.jpg", keywords="beach;rock", author=auth}, mediadir.."img1/claustral.jpg")
pkg:putfile({name="marble.jpg", keywords="beach", author=auth}, mediadir.."img2/marble.jpg")
pkg:putfile({name="uzunji.jpg", keywords="rock", author=auth}, mediadir.."img2/uzunji.jpg")

-- put file created from given string
pkg:putdata({name="sample.txt", mime="text/plain;charset=utf-8", keywords="fox;dog"}, "The quick brown fox jumps over the lazy dog")

-- make 2 file name aliases to 1 file
pkg:putalias("claustral.jpg", "jasper.jpg")

log(string.format("qarataşlar file size: %d bytes", pkg:filesize("qarataslar.jpg")))
log(string.format("total files size sum: %d bytes", pkg.datasize))
log(string.format("packaged: %d files to %d aliases", pkg.recnum, pkg.tagnum))

-- write records table, tags table and finalize wpk-file
pkg:complete()

log "done."
