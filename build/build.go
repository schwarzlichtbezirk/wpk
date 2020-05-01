package main

import (
	"log"
	"os"

	"github.com/yuin/gopher-lua"
)

func lualog(ls *lua.LState) int {
	var s = ls.CheckString(1)

	log.Println(s)
	return 0
}

func luaexist(ls *lua.LState) int {
	var path = ls.CheckString(1)

	var err error
	if _, err = os.Stat(path); err == nil {
		ls.Push(lua.LBool(true))
		return 1
	}
	if os.IsNotExist(err) {
		ls.Push(lua.LBool(false))
		return 1
	}
	ls.Push(lua.LBool(false))
	ls.Push(lua.LString(err.Error()))
	return 2
}

func mainluavm(path string) (err error) {
	var ls = lua.NewState()
	defer ls.Close()

	OpenPath(ls)
	RegPack(ls)

	ls.SetGlobal("log", ls.NewFunction(lualog))
	ls.SetGlobal("exist", ls.NewFunction(luaexist))

	if err = ls.DoFile(path); err != nil {
		return
	}
	return
}

func main() {
	log.Println("starts")
	for _, path := range os.Args[1:] {
		log.Printf("executes: %s", path)
		if err := mainluavm(path); err != nil {
			log.Println(err.Error())
			return
		}
	}
	log.Println("done.")
}

// The End.
