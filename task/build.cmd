@echo off
set wd=%~dp0..

for /F "tokens=*" %%g in ('git describe --tags') do (set buildvers=%%g)
for /F "tokens=*" %%g in ('go run %~dp0/timenow.go') do (set buildtime=%%g)

go build -o %GOPATH%/bin/wpkbuild.exe -v -ldflags="-X 'github.com/schwarzlichtbezirk/wpk/luawpk.BuildVers=%buildvers%' -X 'github.com/schwarzlichtbezirk/wpk/luawpk.BuildTime=%buildtime%'" %wd%/util/build
go build -o %GOPATH%/bin/wpkextract.exe -v %wd%/util/extract
go build -o %GOPATH%/bin/wpkpack.exe -v %wd%/util/pack
