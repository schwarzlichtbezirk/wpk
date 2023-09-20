#!/bin/bash
wd=$(realpath -s "$(dirname "$0")/..")

buildvers=$(git describe --tags)
buildtime=$(go run "$wd/task/timenow.go") # $(date -u +'%FT%TZ')

go build -o $GOPATH/bin/wpkbuild.exe -v -ldflags="-X 'github.com/schwarzlichtbezirk/wpk/luawpk.BuildVers=$buildvers' -X 'github.com/schwarzlichtbezirk/wpk/luawpk.BuildTime=$buildtime'" $wd/util/build
go build -o $GOPATH/bin/wpkextract.exe -v ./util/extract
go build -o $GOPATH/bin/wpkpack.exe -v ./util/pack
