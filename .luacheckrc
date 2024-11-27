
-- configuration for `luacheck`
-- see: https://luacheck.readthedocs.io/en/stable/index.html
-- see: https://github.com/lunarmodules/luacheck

globals = {
	-- tables
	"wpk",
}

read_globals = {
	-- variables
	"buildvers", "buildtime", "bindir", "scrdir", "tmpdir",
	-- tables
	"path",
	-- functions
	"log", "checkfile", "bin2hex", "hex2bin", "milli2time", "time2milli",
}

std = { -- Lua 5.1 & GopherLua
	read_globals = {
		-- basic functions
		"assert", "collectgarbage", "dofile", "error", "_G", "getfenv",
		"getmetatable", "ipairs", "load", "loadfile", "loadstring",
		"next", "pairs", "pcall", "print",
		"rawequal", "rawget", "rawset", "select", "setfenv", "setmetatable",
		"tonumber", "tostring", "type", "unpack", "_VERSION", "xpcall",
		"module", "require",
		"goto", -- GopherLua
		-- basic libraries
		"coroutine", "debug", "io", "math", "os", "package", "string", "table",
		"channel" -- GopherLua
	}
}
