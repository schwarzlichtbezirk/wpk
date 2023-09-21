#!/bin/bash

buildvers=$(git describe --tags)
buildtime=$(go run "$(dirname "$0")/timenow.go") # $(date -u +'%FT%TZ')

wd=$(realpath -s "$(dirname "$0")/..")
go build -o $GOPATH/bin/wpkbuild.exe -v -ldflags="-X 'github.com/schwarzlichtbezirk/wpk/luawpk.BuildVers=$buildvers' -X 'github.com/schwarzlichtbezirk/wpk/luawpk.BuildTime=$buildtime'" $wd/util/build
go build -o $GOPATH/bin/wpkextract.exe -v $wd/util/extract
go build -o $GOPATH/bin/wpkpack.exe -v $wd/util/pack
