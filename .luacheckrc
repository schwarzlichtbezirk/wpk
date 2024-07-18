
-- configuration for `luacheck`
-- see: https://luacheck.readthedocs.io/en/stable/index.html
-- see: https://github.com/lunarmodules/luacheck

globals = {
	"wpk", "path",
	"log", "checkfile", "bin2hex", "hex2bin", "milli2time", "time2milli",
}

read_globals = {
	"buildvers", "buildtime", "bindir", "scrdir", "tmpdir",
}
