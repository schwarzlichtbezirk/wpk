#!/bin/bash

buildvers=$(git describe --tags)
# See https://tc39.es/ecma262/#sec-date-time-string-format
# time format acceptable for Date constructors.
buildtime=$(date +'%FT%T.%3NZ')

wd=$(realpath -s "$(dirname "$0")/..")
go build -o $GOPATH/bin/wpkbuild.exe -v -ldflags="\
 -X 'github.com/schwarzlichtbezirk/wpk/luawpk.BuildVers=$buildvers'\
 -X 'github.com/schwarzlichtbezirk/wpk/luawpk.BuildTime=$buildtime'"\
 $wd/cmd/build
go build -o $GOPATH/bin/wpkextract.exe -v $wd/cmd/extract
go build -o $GOPATH/bin/wpkpack.exe -v $wd/cmd/pack
