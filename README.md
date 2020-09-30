
Build and use data files package.

# Preamble
Software often uses a lot of data files and needs effective method to manage them and get quick access. Stitch them to single package and then get access by mapped memory is a good resolution.

Package keeps all files together with warranty that no any file will be deleted, moved or changed separately from others during software is running, it's snapshot of entire development workflow. Package saves disk space due to except file system granulation, especially if many small files are packed together. Package allows to lock fragment of single file to get access to mapped memory.

# Structure
Library have root **wpk** module that used by any code working with .wpk packages. And modules to build utilities for packing/unpacking data files to package:
 - **wpk** 
Places data files into single package, extracts them, and gives API for access to package.
 - **wpk/pack**
Small utility designed to pack a directory, or a list of directories into an package.
 - **wpk/extract**
Small utility designed to extract all packed files from package, or list of packages to given directory.
 - **wpk/build**
Utility for programmable packages build. Uses [Lua 5.1]([https://www.lua.org/manual/5.1/](https://www.lua.org/manual/5.1/)) to script package building process.
 - **wpk/test**
Contains some Lua-scripts to test wpk/build utility and learn scripting API opportunities.
 - **wpk/bulk**
Wrapper for package to hold WPK-file whole content as a slice. Actual for small packages.
 - **wpk/mmap**
Wrapper for package to get access to nested files as to memory mapped blocks. Actual for large packages.

# How to use
At first, install [Golang](https://golang.org/) minimum 1.9 version, and get this package:

    go get github.com/schwarzlichtbezirk/wpk

Then you can make simple package with files at [test/media](https://github.com/schwarzlichtbezirk/wpk/tree/master/test/media) directory by command:

    go run github.com/schwarzlichtbezirk/wpk/pack -src=$(GOPATH)/src/github.com/schwarzlichtbezirk/wpk/test/media -dst=$(GOPATH)/bin/media.wpk
It's runs utility that receives source directory full path and destination package full path. $(GOPATH) at command line directory path replaced by GOPATH environment variable value. To place any other environment variable VAR you can by $(VAR). In this sample package placed into *bin* directory with other compiled golang binary files.

To extract files from this *media.wpk* package run command:

    go run github.com/schwarzlichtbezirk/wpk/extract -md -src=$(GOPATH)/bin/media.wpk -dst=$(GOPATH)/bin/media
and see files in directory *bin/media*.

To build package at development workflow you can by **build** utility. It can put files into package from any different paths with given names, and bind addition tags to each file, such as MIME types, keywords, CRC, MD5, SHA256 and others. Run this command to see how its work:

    go run github.com/schwarzlichtbezirk/wpk/build $(GOPATH)/src/github.com/schwarzlichtbezirk/wpk/test/build.lua
and see *build.wpk* file in wpk/test source directory near the script.

# WPK-format
Package consist of 3 sections:
 1. **Header**, constantly 40 bytes. Starts with signature (24 bytes), then follow tags offset, tags number, and records number. Tags offset and records number can be different values, because one file in package can have several aliases.
 2. **Bare data files blocks**.
 3. **Tags sets**. Contains list of tags set for each file alias. Each tags set must contain some requered fields: it's ID, file size, file offset in package, file name (path), creation time.

Existing package can be opened to append new files, in this case new files blocks will be posted to *tags sets* old place.

# Script API
**build** utility receives one or more Lua-scripts that maneges package building workflow. Typical sequence is to create new package, setup common properties, put files and add aliases with some tags if it necessary, and complete package building. See whole script API documentation in header comment of [api.lua](https://github.com/schwarzlichtbezirk/wpk/blob/master/test/api.lua) script, and sample package building algorithm below.

[step1.lua](https://github.com/schwarzlichtbezirk/wpk/blob/master/test/step1.lua) and [step2.lua](https://github.com/schwarzlichtbezirk/wpk/blob/master/test/step2.lua) scripts shows sample how to create new package at *step1*:

    go run github.com/schwarzlichtbezirk/wpk/build $(GOPATH)/src/github.com/schwarzlichtbezirk/wpk/test/step1.lua
and later append to exising package new files at *step2* call:

    go run github.com/schwarzlichtbezirk/wpk/build $(GOPATH)/src/github.com/schwarzlichtbezirk/wpk/test/step2.lua
[packdir.lua](https://github.com/schwarzlichtbezirk/wpk/blob/master/test/packdir.lua) script has function that can be used to put to package directory with original tree hierarchy.

# WPK API usage
See [godoc](https://godoc.org/github.com/schwarzlichtbezirk/wpk) with API description, and [wpk_test.go](https://github.com/schwarzlichtbezirk/wpk/blob/master/wpk_test.go) for usage samples.

On your program initialisation open prepared wpk-package by [Package.Read](https://godoc.org/github.com/schwarzlichtbezirk/wpk#Package.Read) call. It reads tags sets of package on this call and has no any others reading of tags sets later. [File](https://godoc.org/github.com/schwarzlichtbezirk/wpk#File) structure helps you to implement custom [http.FileSystem](https://golang.org/pkg/net/http/#FileSystem) to provide local file system and route it by [http.FileServer](https://golang.org/pkg/net/http/#FileServer). **wpk/bulk** and **wpk/mmap** modules already has file system interface implementation.
