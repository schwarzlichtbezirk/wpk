package main

import (
	"log"
	"os"

	lw "github.com/schwarzlichtbezirk/wpk/luawpk"
	"github.com/schwarzlichtbezirk/wpk/util"
)

func main() {
	for _, fpath := range os.Args[1:] {
		if err := lw.RunLuaVM(util.Envfmt(fpath, nil)); err != nil {
			log.Println(err.Error())
			return
		}
	}
}

// The End.
