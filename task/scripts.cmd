@echo off

echo.
echo *** api.lua
%GOPATH%/bin/wpkbuild.exe %~dp0../test/api.lua
del %GOPATH%\bin\api.wpk

echo.
echo *** build.lua
%GOPATH%/bin/wpkbuild.exe %~dp0../test/build.lua
del %GOPATH%\bin\build.wpk

echo.
echo *** packdir.lua
%GOPATH%/bin/wpkbuild.exe %~dp0../test/packdir.lua
del %TEMP%\packdir.wpk

echo.
echo *** split.lua
%GOPATH%/bin/wpkbuild.exe %~dp0../test/split.lua
del %TEMP%\build.wpt
del %TEMP%\build.wpf

echo.
echo *** step1.lua and step2.lua
%GOPATH%/bin/wpkbuild.exe %~dp0../test/step1.lua
%GOPATH%/bin/wpkbuild.exe %~dp0../test/step2.lua
del %TEMP%\steps.wpk
