
--[[

In addition to standard Lua 5.1 library there is registered API
to build wpk-packages.

*global registration*
	Data and functions defined in global namespace.

	bindir - string value with directory to this running binary file destination.
		Directory is splash-terminated.
	scrdir - string value with directory to this Lua script.
		Directory is splash-terminated.
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
	envfmt(fpath) - replaces all entries "$(envname)" in path, where 'envname' is
		an environment variable, to it's value.


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
	uint16 - get/set uint16 data, 2 bytes
	uint32 - get/set uint32 data, 4 bytes
	uint64 - get/set uint64 data, 8 bytes
	uint   - get unspecified size unsigned int data
	number - get/set float64 data, 8 bytes


*wpk* library:

	constructor:
	new() - creates new empty object.

	properties:
	path - getter only, returns path to opened wpk-file.
	recnum - getter only, returns number of records in file allocation table.
	tagnum - getter only, returns number of records in tags table.
	datasize - getter only, returns size sum of all data records.
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
	load(fpath) - read allocation table and tags table by specified wpk-file path.
		File descriptor is closed after this function call.
	begin(fpath) - start to write new empty package with given path.
		Package can not be used until writing will be 'complete'. If package with
		given path is already exist, it will be rewritten.
	append() - start to append new files to already existing package, opened by
		previous call to 'load'. Package can not be used until writing will be 'complete'.
	complete() - write allocation table and tags table, and finalize package writing.
	glob(pattern) - returns the names of all files in package matching pattern or nil
		if there is no matching file.
	hasfile(fname) - check up file name existence in tags table.
	filesize(fname) - return record size of specified file name.
	putfile(tags, fpath) - write file with specified full path to package file,
		and puts specified tags set to tags table. File name expected and must be
		unique for package. File id (tag ID = 0) and file creation time tag will be
		inserted to tags set. After file writing there is tags set adjust by add
		marked tags with hashes (MD5, SHA1, SHA224, etc).
	putdata(tags, data) - write file with specified as string 'data' content,
		and puts specified tags set to tags table. File name expected and must be
		unique for package. File id (tag ID = 0) and current time as creation time
		tag will be inserted to tags set. After file writing there is tags set
		adjust by add marked tags with hashes (MD5, SHA1, SHA224, etc).
	rename(fname1, fname2) - rename file name with fname1 to fname2. Rename is
		carried out by replace name tag in file tags set from one name to other.
	putalias(fname1, fname2) - clone tags set with file name fname1 and replace
		name tag in it to fname2. So, there will be two tags set referenced to
		one data block.
	delalias(fname) - delete tags set with specified file name. Data block is
		still remains.
	hastag(fname, tid) - check up tag existence in tags set for specified file,
		returns boolean value. 'tid' can be numeric ID or string representation
		of tag ID.
	gettag(fname, tid) - returns tag with given ID as userdata object for
		specified file. Returns nothing if tags set of specified file
		has no that tag. 'tid' can be numeric ID or string representation of tag ID.
	settag(fname, tid, tag) - set tag with given ID to tags set of specified file.
		'tid' can be numeric ID or string representation of tag ID. 'tag' can be
		constructed userdata object, or string, or boolean. Numeric values cannot
		be given as tag to prevent ambiguous data size interpretation.
	deltag(fname, tid) - delete tag with given ID from tags set of specified file.
		'tid' can be numeric ID or string representation of tag ID.
	gettags(fname) - returns table with tags set of specified file. There is keys -
		numeric tags identifiers, values - 'tag' userdata.
	settags(fname, tags) - receive table with tags that will be replaced at tags
		set of specified file, or added if new. Keys of table can be numeric IDs
		or string representation of tags ID. Values - can be 'tag' userdata objects,
		or strings, or boolean.
	addtags(fname, tags) - receive table with tags that will be added to tags set
		of specified file. If file tags set already has given tags, those tags will
		be skipped. Keys of table can be numeric IDs or string representation of
		tags ID. Values - can be 'tag' userdata objects, or strings, or boolean.
	deltags(fname, tags) - receive table with numeric tags IDs or string
		representation of tags ID, which should be removed. Values of table does
		not matter.

]]

log "starts"

-- define some functions for packing workflow
local function logfmt(...) -- write to log formatted string
	log(string.format(...))
end
function wpk.create(fpath)-- additional wpk-constructor
	local pkg = wpk.new()
	pkg.automime = true -- put MIME type for each file if it is not given explicit
	pkg.crc32 = true -- generate CRC32 Castagnoli code for each file
	pkg:begin(fpath) -- open wpk-file for write
	return pkg
end
function wpk:logfile(fname) -- write record log
	logfmt("packed %d file %s, crc=%s", self:gettag(fname, "fid").uint32, fname, tostring(self:gettag(fname, "crc32")))
end
function wpk:safealias(fname1, fname2) -- make 2 file name aliases to 1 file
	if self:hasfile(fname1) then
		self:putalias(fname1, fname2)
		logfmt("maked alias '%s' to '%s'", fname2, fname1)
	else
		logfmt("file '%s' is not found in package", fname1)
	end
end

-- starts new package
local pkg = wpk.create(scrdir.."api.wpk")
pkg.secret = "package-private-key" -- private key to sign cryptographic hashes for each file
pkg.sha224 = true -- generate SHA224 hash for each file

-- put images with keywords and author addition tags
for i, tags in ipairs{
	{name="bounty.jpg", keywords="beach", category="image"},
	{name="img1/qarataslar.jpg", keywords="beach;rock", category="photo"},
	{name="img1/claustral.jpg", keywords="beach;rock", category="photo"},
	{name="img2/marble.jpg", keywords="beach", category="photo"},
	{name="img2/uzunji.jpg", keywords="rock", category="photo"}
} do
	tags.author="schwarzlichtbezirk"
	pkg:putfile(tags, path.join(scrdir, "media", tags.name))
	pkg:logfile(tags.name)
end
-- make alias to file included at list
pkg:safealias("img1/claustral.jpg", "jasper.jpg")
pkg:settag("jasper.jpg", "comment", "beach between basalt cliffs")

log(string.format("total files size sum: %d bytes", pkg.datasize))
log(string.format("packaged: %d files to %d aliases", pkg.recnum, pkg.tagnum))

-- write records table, tags table and finalize wpk-file
pkg:complete()

log "done."
