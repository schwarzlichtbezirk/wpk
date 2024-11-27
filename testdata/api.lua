
--[[

In addition to standard Lua 5.1 library, there is registered API
to build wpk-packages.

*global registration*
	Data and functions defined in global namespace.

	variables:
	buildvers - compiled binary version, sets by compiler during compilation
		with git tags value.
	buildtime - compiled binary build date, sets by compiler during compilation.
	bindir - string value with directory to this running binary file destination.
	scrdir - string value with directory to this Lua script.
	tmpdir - string value with default directory to use for temporary files.

	functions:
	log(str) - writes to log given string with current date.
	checkfile(fpath) - checks up file existence with given full path to it.
		Returns 2 values: first - boolean file existence. If first is true,
		second value indicates whether given path is file. False second value
		means directory. If first is false, second value can be string message
		of occurred error. Also, first value can be false if file is exist but
		access to file is denied.
	bin2hex(bin) - converts 'bin' argument given as string with raw binary data
		to string with data in hexadecimal representation.
	hex2bin(hex) - converts string 'hex' argument with data in hexadecimal
		representation to string with raw binary data.
	milli2time(milli, layout) - converts time given in UNIX milliseconds to
		formatted time string with optional 'layout', by default layout ISO 8601,
		used at ECMAScript.
	time2milli(arg, layout) - converts time given as string 'arg' to UNIX
		milliseconds. Time argument can be formatted with given 'layout',
		by default layout ISO 8601, used at ECMAScript.


*path* library:
	Implements utility routines for manipulating filename paths.

	toslash(fpath) - returns the result of replacing each separator character
		in fpath with a slash ('/') character. Multiple separators are replaced
		by multiple slashes.
	clean(fpath) - returns the shortest path name equivalent to path by purely
		lexical processing.
	volume(fpath) - returns leading volume name. Given "C:\foo\bar"
		it returns "C:" on Windows. Given "\\host\share\foo" it returns
		"\\host\share". On other platforms it returns "".
	dir(fpath) - returns all but the last element of fpath, typically
		the path's directory. If the path is empty, Dir returns ".".
	name(fpath) - returns name of file in given file path without extension.
	base(fpath) - returns the last element of fpath. Trailing path separators
		are removed before extracting the last element. If the path is empty,
		base returns ".". If the path consists entirely of separators,
		base returns a single separator.
	ext(fpath) - returns the file name extension used by fpath.
	split(fpath) - splits path immediately following the final Separator,
		separating it into a directory and file name component. If there is no
		Separator in path, Split returns an empty dir and file set to path.
		The returned values have the property that path = dir+file.
	match(name, pattern) - reports whether name matches the shell file name pattern.
	join(...) - joins any number of path elements into a single path.
	glob(pattern) - returns the names of all files matching pattern or nil
		if there is no matching file. The syntax of patterns is the same as
		in 'match'. The pattern may describe hierarchical names such
		as /usr/*/bin/ed (assuming the Separator is '/').
	enum(dir, n) - enumerates all files of given directory, returns result as table.
		If n > 0, returns at most n file names. If n <= 0, returns all the
		file names from the directory. 'n' is optional parameter, -1 by default.
	envfmt(fpath) - replaces all entries "$envname" or "${envname}" or "%envname%"
		in path, where 'envname' is an environment variable, to it's value.


*wpk* userdata:
	Implements access at script to Package golang object.

	constructor:
	new() - creates new empty package object.

	properties:
	label - getter/setter for package label in package info. Getter returns
		nothing if label is absent. If setter was called, it creates package
		info table in case if it was absent before.
	pkgpath - getter only, returns path to opened single wpk-file, or to header
		part file on case of splitted package.
	datpath - getter only, returns path to opened package data part file of
		splitted package.
	recnum - getter only, counts number of unique records in file allocation table.
	tagnum - getter only, counts number of records in tags table, i.e. all aliases.
	fftsize - getter only, calculates size of file tags table.
	datasize - getter only, returns package data size from current file position.
	autofid - get/set mode to put for each new file tag with unique file ID (FID).
	automime - get/set mode to put for each new file tag with its MIME
		determined by file extension, if it does not issued explicitly.
	secret - get/set private key to sign hash MAC (MD5, SHA1, SHA224, etc).
	crc32 - get/set mode to put for each new file tag with CRC32 of file.
		Used Castagnoli's polynomial 0x82f63b78.
	crc64 - get/set mode to put for each new file tag with CRC64 of file.
		Used ISO polynomial 0xD800000000000000.
	md5 - get/set mode to put for each new file tag with MD5-hash of file,
		signed by 'secret' key.
	sha1 - get/set mode to put for each new file tag with SHA1-hash of file,
		signed by 'secret' key.
	sha224 - get/set mode to put for each new file tag with SHA224-hash of file,
		signed by 'secret' key.
	sha256 - get/set mode to put for each new file tag with SHA256-hash of file,
		signed by 'secret' key.
	sha384 - get/set mode to put for each new file tag with SHA384-hash of file,
		signed by 'secret' key.
	sha512 - get/set mode to put for each new file tag with SHA512-hash of file,
		signed by 'secret' key.

	methods:
	load(pkgpath, datpath) - read allocation table and tags table by specified
		wpk-file path. File descriptor is closed after this function call.
		'datpath' can be skipped for package in single file.
	begin(pkgpath, datpath) - start to write new empty package with given paths.
		If package should be splitten on tags table and data files, 'pkgpath' points
		to file with tags table, and 'datpath' points to data file. If package should
		be written to single file, 'pkgpath' only is givens, and 'datpath' is nil.
		Package in single file can not be used until writing will be 'finalize'.
		Splitted package can be used after each update by 'flush' during writing.
		If package with given path is already exist, it will be rewritten.
	append() - start to append new files to already existing package, opened by
		previous call to 'load'. Package can not be used until writing will be 'finalize'.
	finalize() - write allocation table and tags table, and finalize package writing.
	flush() - only for splitted package writes allocation table and tags table,
		and continue common files writing workflow.
	sumsize() - return size sum of all data records. Some files may refer to shared
		data, so sumsize can be more then datasize.
	glob(pattern) - returns the names of all files in package matching pattern or nil
		if there is no matching file.
	hasfile(fkey) - check up file name existence in tags table.
	filesize(fkey) - return record size of specified file name.
	putdata(fkey, data, tags) - write file with specified as string 'data' content,
		and insert tagset with specified fkey to tags table. Data writes as is, and
		can be in binary format. Key file name 'fkey' expected and should be unique
		for package. File creation/birth/modify times tags will be inserted to tagset.
		After file writing there is tagset adjust by add marked tags with hashes
		(MD5, SHA1, SHA224, etc). Optional table at 'tags' argument with addition tags
		will be concatenated to file tagset.
	putfile(fkey, fpath, tags) - write file with specified full path 'fpath' to package,
		and insert tagset with specified fkey to tags table. Key file name 'fkey'
		expected and must be unique for package. File creation/birth/modify times tags
		will be inserted to tagset. After file writing there is tagset adjust by add
		marked tags with hashes (MD5, SHA1, SHA224, etc). Optional table at 'tags'
		argument with addition tags will be concatenated to file tagset.
	rename(fkey1, fkey2) - rename file name with 'fkey1' to 'fkey2'. Rename is
		carried out by replace name tag in file tagset from one name to another.
		Keeps link to original file name.
	renamedir(dir1, dir2, skipexist) - renames all files in package with
		'dir1' path to 'dir2' path. Carried out by replace name tag in each
		file tagset from one directory prefix to another.
	putalias(fkey1, fkey2) - clone tagset with file name 'fkey1' and replace
		name tag in it to 'fkey2'. So, there will be two tagset referenced to
		one data block. Keeps link to original file name.
	delalias(fkey) - delete tagset with specified file name. Data block is
		still remains.
	hastag(fkey, tid) - check up tag existence in tagset for specified file,
		returns boolean value. 'tid' can be numeric ID or string representation
		of tag ID.
	gettag(fkey, tid) - returns tag with given ID for specified file.
		Returns nothing if tagset of specified file has no that tag.
		'tid' can be numeric ID or string representation of tag ID.
	settag(fkey, tid, tag) - set tag with given ID to tagset of specified file.
		'tid' can be numeric ID or string representation of tag ID. 'tag' can be
		of type described below.
	addtag(fkey, tid, tag) - add tag with given ID to tagset of specified file
		only if tagset does not have same yet. 'tid' can be numeric ID or string
		representation of tag ID. 'tag' can be of type described below.
	deltag(fkey, tid) - delete tag with given ID from tagset of specified file.
		'tid' can be numeric ID or string representation of tag ID.
	gettags(fkey) - returns table with tagset of specified file. There is keys -
		numeric tags identifiers, values - tags of types described below.
	settags(fkey, tags) - receive table with tags that will be replaced at tags
		set of specified file, or added if new. Keys of table can be numeric IDs
		or string representation of tags ID. Values - tags of types described below.
	addtags(fkey, tags) - receive table with tags that will be added to tagset
		of specified file. If file tagset already has given tags, those tags will
		be skipped. Keys of table can be numeric IDs or string representation of
		tags ID. Values - tags of types described below.
	deltags(fkey, tags) - receive table with numeric tags IDs or string
		representation of tags ID, which should be removed. Values of table does
		not matter.
	getinfo() - returns table with package info, if it present.
	setupinfo(tags) - setup given table with tags as package info.


*tags types*
	Any tags can be some of the followed types: binary data, string, boolean,
	unsigned integer, float number, time. Tag binary data represented as hexadecimal
	Lua-string, in some cases binary can be represented as integer in Lua-number.
	Tags unsigned integers can be variable length: 1, 2, 4 or 8 bytes, and can be
	represented by Lua-numbers or string with integer content. Numberic tags used
	only with 8 bytes length, and represented by Lua-numbers or strings with number
	content. Boolean tags represented by Lua-booleans, also all non-zero numbers
	interpreted as 'true', zeros interpreted as 'false'. Empty strings and strings
	with "false" content also interpreted as 'false', and all other strings
	interpreted as 'true'. Lua-userdata and tables does not converted to tag-values.

	available named tags:
	name    	ID	Lua-type
	offset  	1	number
	size    	2	number
	path    	3	string
	fid     	4	number
	mtime   	5	string time
	atime   	6	string time
	ctime   	7	string time
	btime   	8	string time
	attr    	9	number
	mime    	10	string
	crc32   	12	hex string, 4 bytes
	crc32ieee	11	hex string, 4 bytes
	crc32c  	12	hex string, 4 bytes
	crc32k  	13	hex string, 4 bytes
	crc64   	14	hex string, 8 bytes
	crc64iso	14	hex string, 8 bytes
	md5     	20	hex string, 16 bytes
	sha1    	21	hex string, 20 bytes
	sha224  	22	hex string, 28 bytes
	sha256  	23	hex string, 32 bytes
	sha384  	24	hex string, 48 bytes
	sha512  	25	hex string, 64 bytes
	tmbjpeg 	100	hex string
	tmbwebp 	101	hex string
	label   	110	string
	link    	111	string
	keywords	112	string
	category	113	string
	version 	114	string
	author  	115	string
	comment 	116	string

]]

-- define some functions for packing workflow
local function logfmt(...) -- write to log formatted string
	log(string.format(...))
end
local rfc822 = "02 Jan 06 15:04 MST" -- time reformat layout
function wpk.create(fpath)-- additional wpk-constructor
	local pkg = wpk.new()
	pkg.autofid = true -- put auto generated file ID for each file
	pkg.automime = true -- put MIME type for each file if it is not given explicit
	pkg.crc32 = true -- generate CRC32 Castagnoli code for each file
	pkg:begin(fpath) -- open wpk-file for write
	log("starts: "..pkg.pkgpath)
	return pkg
end
function wpk:logfile(fkey) -- write record log
	logfmt("#%d %s, %d bytes, %s, crc=%s",
		assert(self:gettag(fkey, "fid")),
		fkey,
		self:filesize(fkey),
		milli2time(time2milli(self:gettag(fkey, "btime")), rfc822),
		assert(self:gettag(fkey, "crc32")))
end
function wpk:safealias(fkey1, fkey2) -- make 2 file name aliases to 1 file
	if self:hasfile(fkey1) then
		self:putalias(fkey1, fkey2)
		logfmt("maked alias '%s' to '%s'", fkey2, fkey1)
	else
		logfmt("file '%s' is not found in package", fkey1)
	end
end

logfmt("version: %s, builton: %s", buildvers, buildtime)
logfmt("binary dir: %s", bindir)
logfmt("script dir: %s", scrdir)
logfmt("temporary dir: %s", tmpdir)

-- starts new package at golang binary directory
local pkg = wpk.create(path.toslash(path.envfmt"${GOPATH}/bin/api.wpk"))
pkg.label = "api-sample" -- image label
pkg.secret = "package-private-key" -- private key to sign cryptographic hashes for each file
pkg.sha224 = true -- generate SHA224 hash for each file

-- put images with keywords and author addition tags
for fkey, tags in pairs{
	["bounty.jpg"] = {keywords="beach", category="image"},
	["img1/Qarataşlar.jpg"] = {keywords="beach;rock", category="photo"},
	["img1/claustral.jpg"] = {keywords="beach;rock", category="photo"},
	["img2/marble.jpg"] = {keywords="beach", category="photo"},
	["img2/Uzuncı.jpg"] = {keywords="rock", category="photo"},
} do
	local fpath = path.join(scrdir, "media", fkey)
	tags.author = "schwarzlichtbezirk"
	tags.link = fpath
	pkg:putfile(fkey, fpath, tags)
	pkg:logfile(fkey)
end
-- make alias to file included at list
pkg:safealias("img1/claustral.jpg", "jasper.jpg")
pkg:settag("jasper.jpg", "comment", "beach between basalt cliffs")

logfmt("total package data size: %s bytes", pkg.datasize or "N/A")
logfmt("packaged: %d files to %d aliases", pkg.recnum, pkg.tagnum)

-- write records table, tags table and finalize wpk-file
pkg:finalize()
log "done."
