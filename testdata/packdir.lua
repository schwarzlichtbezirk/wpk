
--[[
Script with sample that shows how can be packed whole directory with
subdirectories. There is present checkup-function that checks name of
any enumerated file on skip condition. File will be skipped if it has
extension "tmp", "bak", "log" and so on.
]]

-- inits new package
local pkg = wpk.new()
pkg.autofid = true -- put auto generated file ID for each file
pkg.automime = true -- put MIME type for each file if it is not given explicit
pkg.secret = "package-private-key" -- private key to sign cryptographic hashes for each file
pkg.crc32 = true -- generate CRC32 Castagnoli code for each file
pkg.sha256 = true -- generate SHA256 hash for each file
pkg:setupinfo{ -- setup package info
	label="packed-directory",
	link="github.com/schwarzlichtbezirk/wpk",
	author="schwarzlichtbezirk"
}

-- open wpk-file for write
pkg:begin(path.join(tmpdir, "packdir.wpk"))
log("starts: "..pkg.pkgpath)

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
	for _, pattern in ipairs(skippattern) do
		if string.match(name, pattern) then return false end
	end
	local ext = string.match(name, "%.(%w+)$") -- file extension
	if ext and skipext[ext] then return false end
	return true
end
-- pack given directory and add to each file name given prefix
local ansic = "Mon Jan _2 15:04:05 2006" -- time reformat layout
local function packdir(prefix, dir)
	for _, name in ipairs(path.enum(dir)) do
		local fkey = prefix..name
		local fpath = dir..name
		local access, isfile = checkfile(fpath)
		if access and checkname(name) then
			if isfile then
				pkg:putfile(fkey, fpath, {
					link = fpath,
					author = "schwarzlichtbezirk",
				})
				logfmt("#%d %s, %d bytes, %s, %s",
					assert(pkg:gettag(fkey, "fid")), fkey,
					pkg:filesize(fkey),
					milli2time(time2milli(pkg:gettag(fkey, "mtime")), ansic),
					assert(pkg:gettag(fkey, "mime"))
				)
			else
				packdir(fkey.."/", fpath.."/")
			end
		end
	end
end

packdir("", path.join(scrdir, "media").."/")

-- write records table, tags table and finalize wpk-file
pkg:finalize()

log(tostring(pkg))
log "done."
