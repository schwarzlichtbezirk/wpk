
log "starts"

-- inits new package
local pkg = wpk.new()
pkg.automime = true -- put MIME type for each file if it is not given explicit
pkg.secret = "package-private-key" -- private key to sign cryptographic hashes for each file
pkg.crc32 = true -- generate CRC32 Castagnoli code for each file
pkg.sha256 = true -- generate SHA256 hash for each file

-- open wpk-file for write
pkg:begin(scrdir.."packdir.wpk")

-- write to log formatted string
local function logfmt(...)
	log(string.format(...))
end
-- check file names can be included to package
local function checkname(name)
	local fc = string.sub(name, 1, 1) -- first char
	if fc == "." or fc == "~" then return false end
	name = string.lower(name)
	if name == "thumb.db" then return false end
	local ext = string.sub(name, -4, -1) -- file extension
	if ext == ".sys" or ext == ".tmp" or ext == ".bak" or ext == ".wpk" then return false end
	return true
end
-- pack given directory and add to each file name given prefix
local function packdir(dir, prefix)
	for i, name in ipairs(path.enum(dir)) do
		local fname = prefix..name
		local fpath = dir..name
		local access, isdir = checkfile(fpath)
		if access and checkname(name) then
			if isdir then
				packdir(fpath.."/", fname.."/")
			else
				pkg:putfile({name=fname, author="schwarzlichtbezirk"}, fpath)
				logfmt("packed %d file %s, crc=%s", pkg:gettag(fname, "fid").uint32, fname, tostring(pkg:gettag(fname, "crc32")))
			end
		end
	end
end

packdir(scrdir.."media/", "")

-- write records table, tags table and finalize wpk-file
pkg:complete()

log "done."
