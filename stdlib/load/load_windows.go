package load

import "github.com/hirochachacha/plua/internal/version"

const (
	ldir        = "!\\lua\\"
	shrdir      = "!\\..\\share\\lua\\" + version.LUA_VERSION + "\\"
	defaultPath = ldir + "?.lua;" + ldir + "?\\init.lua;" + shrdir + "?.lua;" + shrdir + "?\\init.lua;" + ".\\?.lua;" + ".\\?\\init.lua"

	dsep   = "\\"
	psep   = ";"
	mark   = "?"
	edir   = "!"
	ignore = "-"

	config = dsep + "\n" + psep + "\n" + mark + "\n" + edir + "\n" + ignore + "\n"
)
