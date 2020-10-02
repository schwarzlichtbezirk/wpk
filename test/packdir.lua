
-- inits new package
local pkg = wpk.new()
pkg.automime = true -- put MIME type for each file if it is not given explicit
pkg.secret = "package-private-key" -- private key to sign cryptographic hashes for each file
pkg.crc32 = true -- generate CRC32 Castagnoli code for each file
pkg.sha256 = true -- generate SHA256 hash for each file

-- open wpk-file for write
pkg:begin(path.join(tmpdir, "packdir.wpk"))
log("starts: "..pkg.path)

-- write to log formatted string
local function logfmt(...)
	log(string.format(...))
end
-- patterns for ignored files
local skippattern = {
	"^packdir%.lua$", -- script that generate this package
	"^thumb%.db$",
	"^rca%w+$",
	"^%$recycle%.bin$",
}
-- extensions of files that should not be included to package
local skipext = {
	wpk = true,
	sys = true,
	tmp = true,
	bak = true,
	-- compiler intermediate output
	log = true, tlog = true, lastbuildstate = true, unsuccessfulbuild = true,
	obj = true, lib = true, res = true,
	ilk = true, idb = true, ipdb = true, iobj = true, pdb = true, pgc = true, pgd = true,
	pch = true, ipch = true,
	cache = true,
}
-- check file names can be included to package
local function checkname(name)
	local fc = string.sub(name, 1, 1) -- first char
	if fc == "." or fc == "~" then return false end
	name = string.lower(name)
	for i, pattern in ipairs(skippattern) do
		if string.match(name, pattern) then return false end
	end
	local ext = string.match(name, "%.(%w+)$") -- file extension
	if ext and skipext[ext] then return false end
	return true
end
-- pack given directory and add to each file name given prefix
local function packdir(dir, prefix)
	for i, name in ipairs(path.enum(dir)) do
		local kpath = prefix..name
		local fpath = dir..name
		local access, isdir = checkfile(fpath)
		if access and checkname(name) then
			if isdir then
				packdir(fpath.."/", kpath.."/")
			else
				pkg:putfile(kpath, fpath)
				pkg:settag(kpath, "author", "schwarzlichtbezirk")
				logfmt("#%d %s, %d bytes, %s",
					pkg:gettag(kpath, "fid").uint32, kpath,
					pkg:filesize(kpath), pkg:gettag(kpath, "mime").string)
			end
		end
	end
end

packdir(scrdir.."media/", "")
log(tostring(pkg))

-- write records table, tags table and finalize wpk-file
pkg:complete()

log "done."
