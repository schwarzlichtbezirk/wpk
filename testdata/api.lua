
--[[

In addition to standard Lua 5.1 library, there is registered API
to build wpk-packages.

*global registration*
	Data and functions defined in global namespace.

	bindir - string value with directory to this running binary file destination.
	scrdir - string value with directory to this Lua script.
	tmpdir - string value with default directory to use for temporary files.
	log(str) - writes to log given string with current date.
	checkfile(fpath) - checks up file existence with given full path to it.
		Returns 2 values: first - boolean file existence. If first is true,
		second value indicates whether given path is directory. If first is false,
		second value can be string message of occurred error. Also, first value
		can be false if file is exist but access to file is denied.

*path* library:
	Implements utility routines for manipulating filename paths. Brings back slashes
	to normal slashes.

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
	envfmt(fpath) - replaces all entries "${envname}" or "$envname" or "%envname%"
		in path, where 'envname' is an environment variable, to it's value.


*tag* userdata:

	constructors:
	newhex(str) - creates tag from hexadecimal string.
	newbase64(str) - creates tag from base64 encoded binary.
	newstring(str) - creates tag from string.
	newbool(val) - creates tag from boolean value.
	newuint16(val) - convert given number to 2-bytes unsigned integer tag.
	newuint32(val) - convert given number to 4-bytes unsigned integer tag.
	newuint64(val) - convert given number to 8-bytes unsigned integer tag.
	newnumber(val) - convert given number to 8-bytes tag explicitly.

	operators:
	__tostring - returns hexadecimal encoded representation of byte slice.
	__len - returns number of bytes in byte slice.

	properties:
	hex    - get/set hexadecimal encoded representation of binary value
	base64 - get/set base64 encoded representation of binary value
	string - get/set UTF-8 string value
	bool   - get/set boolean data, 1 byte
	uint8  - get/set uint8 data, 1 byte
	uint16 - get/set uint16 data, 2 bytes
	uint32 - get/set uint32 data, 4 bytes
	uint64 - get/set uint64 data, 8 bytes
	uint   - get unspecified size unsigned int data
	number - get/set float64 data, 8 bytes


*wpk* library:

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
	automime - get/set mode to put for each new file tag with its MIME
		determined by file extension, if it does not issued explicitly.
	nolink - get/set mode to exclude link from tagset. Exclude on 'true'.
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
	putfile(fkey, fpath) - write file with specified full path (fpath) to package,
		and insert tagset with specified fkey to tags table. Key file name (fkey)
		expected and must be unique for package. File id (tag ID = 0) and file
		creation time tag will be inserted to tagset. After file writing there is
		tagset adjust by add marked tags with hashes (MD5, SHA1, SHA224, etc).
	putdata(fkey, data) - write file with specified as string 'data' content,
		and insert tagset with specified fkey to tags table. Key file name (fkey)
		expected and must be unique for package. File id (tag ID = 0) and current
		time as creation time tag will be inserted to tagset. After file writing
		there is tagset adjust by add marked tags with hashes (MD5, SHA1, SHA224, etc).
	rename(kpath1, kpath2) - rename file name with kpath1 to kpath2. Rename is
		carried out by replace name tag in file tagset from one name to another.
		Keeps link to original file name.
	renamedir(kpath1, kpath2, skipexist) - renames all files in package with
		'kpath1' path to 'kpath2' path. Carried out by replace name tag in each
		file tagset from one directory prefix to another.
	putalias(kpath1, kpath2) - clone tagset with file name kpath1 and replace
		name tag in it to kpath2. So, there will be two tagset referenced to
		one data block. Keeps link to original file name.
	delalias(fkey) - delete tagset with specified file name. Data block is
		still remains.
	hastag(fkey, tid) - check up tag existence in tagset for specified file,
		returns boolean value. 'tid' can be numeric ID or string representation
		of tag ID.
	gettag(fkey, tid) - returns tag with given ID as userdata object for
		specified file. Returns nothing if tagset of specified file
		has no that tag. 'tid' can be numeric ID or string representation of tag ID.
	settag(fkey, tid, tag) - set tag with given ID to tagset of specified file.
		'tid' can be numeric ID or string representation of tag ID. 'tag' can be
		constructed userdata object, or string, or boolean. Numeric values cannot
		be given as tag to prevent ambiguous data size interpretation.
	deltag(fkey, tid) - delete tag with given ID from tagset of specified file.
		'tid' can be numeric ID or string representation of tag ID.
	gettags(fkey) - returns table with tagset of specified file. There is keys -
		numeric tags identifiers, values - 'tag' userdata.
	settags(fkey, tags) - receive table with tags that will be replaced at tags
		set of specified file, or added if new. Keys of table can be numeric IDs
		or string representation of tags ID. Values - can be 'tag' userdata objects,
		or strings, or boolean.
	addtags(fkey, tags) - receive table with tags that will be added to tagset
		of specified file. If file tagset already has given tags, those tags will
		be skipped. Keys of table can be numeric IDs or string representation of
		tags ID. Values - can be 'tag' userdata objects, or strings, or boolean.
	deltags(fkey, tags) - receive table with numeric tags IDs or string
		representation of tags ID, which should be removed. Values of table does
		not matter.
	getinfo() - returns table with package info, if it present.

]]

-- define some functions for packing workflow
local function logfmt(...) -- write to log formatted string
	log(string.format(...))
end
function wpk.create(fpath)-- additional wpk-constructor
	local pkg = wpk.new()
	pkg.automime = true -- put MIME type for each file if it is not given explicit
	pkg.nolink = true -- exclude links
	pkg.crc32 = true -- generate CRC32 Castagnoli code for each file
	pkg:begin(fpath) -- open wpk-file for write
	log("starts: "..pkg.pkgpath)
	return pkg
end
function wpk:logfile(fkey) -- write record log
	logfmt("#%d %s, %d bytes, crc=%s",
		self:gettag(fkey, "fid").uint, fkey,
		self:filesize(fkey), self:gettag(fkey, "crc32").hex)
end
function wpk:safealias(fname1, fname2) -- make 2 file name aliases to 1 file
	if self:hasfile(fname1) then
		self:putalias(fname1, fname2)
		logfmt("maked alias '%s' to '%s'", fname2, fname1)
	else
		logfmt("file '%s' is not found in package", fname1)
	end
end

log("binary dir: "..bindir)
log("script dir: "..scrdir)
log("temporary dir: "..tmpdir)

-- starts new package at golang binary directory
local pkg = wpk.create(path.envfmt"${GOPATH}/bin/api.wpk")
pkg.label = "api-sample" -- image label
pkg.secret = "package-private-key" -- private key to sign cryptographic hashes for each file
pkg.sha224 = true -- generate SHA224 hash for each file

-- put images with keywords and author addition tags
for name, tags in pairs{
	["bounty.jpg"] = {fid=1, keywords="beach", category="image"},
	["img1/Qarataşlar.jpg"] = {fid=2, keywords="beach;rock", category="photo"},
	["img1/claustral.jpg"] = {fid=3, keywords="beach;rock", category="photo"},
	["img2/marble.jpg"] = {fid=4, keywords="beach", category="photo"},
	["img2/Uzuncı.jpg"] = {fid=5, keywords="rock", category="photo"},
} do
	tags.author="schwarzlichtbezirk"
	pkg:putfile(name, path.join(scrdir, "media", name))
	pkg:addtags(name, tags)
	pkg:logfile(name)
end
-- make alias to file included at list
pkg:safealias("img1/claustral.jpg", "jasper.jpg")
pkg:settag("jasper.jpg", "comment", "beach between basalt cliffs")

logfmt("total package data size: %s bytes", pkg.datasize or "N/A")
logfmt("packaged: %d files to %d aliases", pkg.recnum, pkg.tagnum)

-- write records table, tags table and finalize wpk-file
pkg:finalize()
log "done."
