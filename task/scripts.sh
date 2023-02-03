#!/bin/bash

echo ""
echo "*** api.lua"
$GOPATH/bin/wpkbuild.exe $(dirname $0)/../testdata/api.lua
rm $GOPATH/bin/api.wpk

echo ""
echo "*** build.lua"
$GOPATH/bin/wpkbuild.exe $(dirname $0)/../testdata/build.lua
rm $GOPATH/bin/build.wpk

echo ""
echo "*** packdir.lua"
$GOPATH/bin/wpkbuild.exe $(dirname $0)/../testdata/packdir.lua
rm $TEMP/packdir.wpk

echo ""
echo "*** split.lua"
$GOPATH/bin/wpkbuild.exe $(dirname $0)/../testdata/split.lua
rm $TEMP/build.wpt
rm $TEMP/build.wpf

echo ""
echo "*** step1.lua and step2.lua"
$GOPATH/bin/wpkbuild.exe $(dirname $0)/../testdata/step1.lua
$GOPATH/bin/wpkbuild.exe $(dirname $0)/../testdata/step2.lua
rm $TEMP/steps.wpk
