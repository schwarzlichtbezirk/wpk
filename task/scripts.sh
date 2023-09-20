#!/bin/bash
sd=$(realpath -s "$(dirname $0)/../testdata")

echo ""
echo "*** api.lua ***"
$GOPATH/bin/wpkbuild.exe $sd/api.lua
rm $GOPATH/bin/api.wpk

echo ""
echo "*** build.lua ***"
$GOPATH/bin/wpkbuild.exe $sd/build.lua
rm $GOPATH/bin/build.wpk

echo ""
echo "*** packdir.lua ***"
$GOPATH/bin/wpkbuild.exe $sd/packdir.lua
rm $TEMP/packdir.wpk

echo ""
echo "*** split.lua ***"
$GOPATH/bin/wpkbuild.exe $sd/split.lua
rm $TEMP/build.wpt
rm $TEMP/build.wpf

echo ""
echo "*** step1.lua and step2.lua ***"
$GOPATH/bin/wpkbuild.exe $sd/step1.lua
$GOPATH/bin/wpkbuild.exe $sd/step2.lua
rm $TEMP/steps.wpk
