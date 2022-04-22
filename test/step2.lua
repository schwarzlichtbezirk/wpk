
-- ensure package existence
local pkgpath = path.join(tmpdir, "steps.wpk") -- make package full file name on temporary directory
if not checkfile(pkgpath) then
	log "package file 'steps.wpk' to append data is not found, performs previous step"
	dofile(path.join(scrdir, "step1.lua"))
end

print ""
log "starts step 2"

-- inits new package
local pkg = wpk.new()
pkg.automime = true -- put MIME type for each file if it is not given explicit
pkg.secret = "package-private-key" -- private key to sign cryptographic hashes for each file
pkg.crc32 = true -- generate CRC32 Castagnoli code for each file
pkg.sha1 = true -- generate SHA1 hash for each file on this step

-- read records table, tags table of existing package
pkg:load(pkgpath)
-- check if files from this step are appended by test any of them
if pkg:hasfile "img2/marble.jpg" then
	log "files from step 2 already appended"
	os.exit()
end

-- define some local functions for packing workflow
local function logfmt(...) -- write to log formatted string
	log(string.format(...))
end
local function logfile(fkey) -- write record log
	logfmt("#%d file %s, crc=%s",
		pkg:gettag(fkey, "fid").uint32, fkey,
		tostring(pkg:gettag(fkey, "crc32")))
end
local function packfile(fkey, keywords) -- pack given file with common preset
	pkg:putfile(fkey, path.join(scrdir, "media", fkey))
	pkg:addtags(fkey, {keywords=keywords, author="schwarzlichtbezirk"})
	logfile(fkey)
end
local function packdata(fkey, data, keywords) -- put text file created from given string
	pkg:putdata(fkey, data)
	pkg:settags(fkey, {keywords=keywords, mime="text/plain;charset=utf-8"})
	logfile(fkey)
end
local function safealias(fname1, fname2) -- make 2 file name aliases to 1 file
	if pkg:hasfile(fname1) then
		pkg:putalias(fname1, fname2)
		logfmt("maked alias '%s' to '%s'", fname2, fname1)
	else
		logfmt("file '%s' is not found in package", fname1)
	end
end

-- prepare to append new files to existing package
pkg:append()
log("append: "..pkgpath)

-- put images with keywords and author addition tags
packfile("img2/marble.jpg", "beach")
packfile("img2/uzunji.jpg", "rock")

-- make alias to file written at step 1
safealias("img1/claustral.jpg", "jasper.jpg")

-- generate SHA384 hash for files below
pkg.sha384 = true

-- put sample text file created from string
packdata("sample.txt", "The quick brown fox jumps over the lazy dog", "fox;dog")

logfmt("packaged %d files on sum %s bytes", pkg.recnum, pkg.datasize or "N/A")

-- write records table, tags table and finalize wpk-file
pkg:finalize()

log(tostring(pkg))
log "done step 2."
