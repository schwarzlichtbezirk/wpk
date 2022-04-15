@echo off
cd /d %~dp0..
go build -o %GOPATH%/bin/wpkbuild.exe -v ./util/build
go build -o %GOPATH%/bin/wpkextract.exe -v ./util/extract
go build -o %GOPATH%/bin/wpkpack.exe -v ./util/pack
