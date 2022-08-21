%GOPATH%/bin/wpkbuild.exe %~dp0../test/api.lua
del "%GOPATH%\bin\api.wpk"

%GOPATH%/bin/wpkbuild.exe %~dp0../test/build.lua
del "%GOPATH%\bin\build.wpk"

%GOPATH%/bin/wpkbuild.exe %~dp0../test/packdir.lua
del "%TEMP%\packdir.wpk"

%GOPATH%/bin/wpkbuild.exe %~dp0../test/split.lua
del "%GOPATH%\bin\build.wpt"
del "%GOPATH%\bin\build.wpd"

%GOPATH%/bin/wpkbuild.exe %~dp0../test/step1.lua
%GOPATH%/bin/wpkbuild.exe %~dp0../test/step2.lua
del "%TEMP%\steps.wpk"
