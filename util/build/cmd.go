package main

import (
	"log"
	"os"

	"github.com/schwarzlichtbezirk/wpk"
	lw "github.com/schwarzlichtbezirk/wpk/luawpk"
)

func main() {
	for _, fpath := range os.Args[1:] {
		if err := lw.RunLuaVM(wpk.Envfmt(fpath, nil)); err != nil {
			log.Println(err.Error())
			return
		}
	}
}

// The End.
