@echo off
set sd=%~dp0../testdata

echo.
echo *** api.lua ***
%GOPATH%/bin/wpkbuild.exe %sd%/api.lua
del %GOPATH%\bin\api.wpk

echo.
echo *** build.lua ***
%GOPATH%/bin/wpkbuild.exe %sd%/build.lua
del %GOPATH%\bin\build.wpk

echo.
echo *** packdir.lua ***
%GOPATH%/bin/wpkbuild.exe %sd%/packdir.lua
del %TEMP%\packdir.wpk

echo.
echo *** split.lua ***
%GOPATH%/bin/wpkbuild.exe %sd%/split.lua
del %TEMP%\build.wpt
del %TEMP%\build.wpf

echo.
echo *** step1.lua and step2.lua ***
%GOPATH%/bin/wpkbuild.exe %sd%/step1.lua
%GOPATH%/bin/wpkbuild.exe %sd%/step2.lua
del %TEMP%\steps.wpk
