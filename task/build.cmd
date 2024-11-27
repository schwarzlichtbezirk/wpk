@echo off

for /F "tokens=*" %%g in ('git describe --tags') do (set buildvers=%%g)
for /f "tokens=2 delims==" %%g in ('wmic os get localdatetime /value') do set dt=%%g
set buildtime=%dt:~0,4%-%dt:~4,2%-%dt:~6,2%T%dt:~8,2%:%dt:~10,2%:%dt:~12,2%.%dt:~15,3%Z

set wd=%~dp0..
go build -o %GOPATH%/bin/wpkbuild.exe -v -ldflags="-X 'github.com/schwarzlichtbezirk/wpk/luawpk.BuildVers=%buildvers%' -X 'github.com/schwarzlichtbezirk/wpk/luawpk.BuildTime=%buildtime%'" %wd%/cmd/build
go build -o %GOPATH%/bin/wpkextract.exe -v %wd%/cmd/extract
go build -o %GOPATH%/bin/wpkpack.exe -v %wd%/cmd/pack
