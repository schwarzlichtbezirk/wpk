
Library to build and use data files packages.

# Preamble

Software often uses a lot of data files and needs effective method to manage them and get quick access. Stitch them to single package and then get access by mapped memory is a good resolution.

Package keeps all files together with warranty that no any file will be deleted, moved or changed separately from others during software is running, it's snapshot of entire development workflow. Package saves disk space due to except file system granulation, especially if many small files are packed together. Package allows to lock fragment of single file to get access to mapped memory.

# Structure

Library have root **`wpk`** module that used by any code working with `.wpk` packages. And modules to build utilities for packing/unpacking data files to package:

- **wpk**
Places data files into single package, extracts them, and gives API for access to package.

- **wpk/bulk**
Wrapper for package to hold WPK-file whole content as a slice. Actual for small packages (size is much less than the amount of RAM).

- **wpk/mmap**
Wrapper for package to get access to nested files as to memory mapped blocks. Actual for medium size packages (size correlates with the RAM amount).

- **wpk/fsys**
Wrapper for package to get access to nested files by OS files. Actual for large packages (size is much exceeds the amount of RAM) or large nested files.

- **wpk/luawpk**
Package writer with package building process scripting using [Lua 5.1]([https://www.lua.org/manual/5.1/](https://www.lua.org/manual/5.1/)). Typical script workflow is to create package for writing, setup some options, put group of files to package, and finalize it.

- **wpk/test**
Contains some Lua-scripts to test **`wpk/luawpk`** module and learn scripting API opportunities.

- **wpk/util/pack**
Small simple utility designed to pack a directory, or a list of directories into an package.

- **wpk/util/extract**
Small simple utility designed to extract all packed files from package, or list of packages to given directory.

- **wpk/util/build**
Utility for the packages programmable building, based on **`wpk/luawpk`** module.

Compiled binaries of utilities can be downloaded in [Releases](https://github.com/schwarzlichtbezirk/wpk/releases) section.

# How to use

At first, install [Golang](https://golang.org/) minimum 1.17 version for last version of this package, and get this package:

```batch
go get github.com/schwarzlichtbezirk/wpk
```

Then you can make simple package with files at [test/media](https://github.com/schwarzlichtbezirk/wpk/tree/master/test/media) directory by command:

```batch
go run github.com/schwarzlichtbezirk/wpk/util/pack --src=${GOPATH}/src/github.com/schwarzlichtbezirk/wpk/test/media --dst=${GOPATH}/bin/media.wpk
```

It's runs utility that receives source directory full path and destination package full path. `${GOPATH}` at command line directory path replaced by `GOPATH` environment variable value. To place any other environment variable `VAR` you can by `${VAR}`. In this sample package placed into `bin` directory with other compiled golang binary files.

To extract files from this `media.wpk` package run command:

```batch
go run github.com/schwarzlichtbezirk/wpk/util/extract --md --src=${GOPATH}/bin/media.wpk --dst=${GOPATH}/bin/media
```

and see files in directory `${GOPATH}/bin/media`.

To build package at development workflow you can by **`build`** utility. It can put files into package from any different paths with given names, and bind addition tags to each file, such as MIME types, keywords, CRC, MD5, SHA256 and others. Run this command to see how its work:

```batch
go run github.com/schwarzlichtbezirk/wpk/util/build ${GOPATH}/src/github.com/schwarzlichtbezirk/wpk/test/build.lua
```

and see `build.wpk` file in binary directory near compiled output.

# WPK-format

Package consist of 3 sections:

1. **Header**, constantly 64 bytes. Starts with signature (24 bytes), then follow 8 bytes with used at package types sizes, file tags table offset and table size (8+8 bytes), and bata block offset and size (8+8 bytes).

2. **Bare data files blocks**.

3. **File tags set table**. Contains list of tagset for each file alias. Each tagset must contain some requered fields: it's ID, file size, file offset in package, file name (path), creation time. Package can have common description stored as tagset with empty name. This tagset is placed as first record in file tags table.

Existing package can be opened to append new files, in this case new files blocks will be posted to *tags sets* old place.

Package can be splitted in two files: 1) file with header and tags table, `.wpt`-file, it's a short file in most common, and 2) file with data files block, typically `.wpf`-file. In this case package is able for reading during new files packing to package. If process of packing new files will be broken by any case, package remains accessible with information pointed at last header record.

# Script API

**`build`** utility receives one or more Lua-scripts that maneges package building workflow. Typical sequence is to create new package, setup common properties, put files and add aliases with some tags if it necessary, and complete package building. See whole script API documentation in header comment of [api.lua](https://github.com/schwarzlichtbezirk/wpk/blob/master/test/api.lua) script, and sample package building algorithm below.

[step1.lua](https://github.com/schwarzlichtbezirk/wpk/blob/master/test/step1.lua) and [step2.lua](https://github.com/schwarzlichtbezirk/wpk/blob/master/test/step2.lua) scripts shows sample how to create new package at *step1*:

```batch
go run github.com/schwarzlichtbezirk/wpk/util/build ${GOPATH}/src/github.com/schwarzlichtbezirk/wpk/test/step1.lua
```

and later append to exising package new files at *step2* call:

```batch
go run github.com/schwarzlichtbezirk/wpk/util/build ${GOPATH}/src/github.com/schwarzlichtbezirk/wpk/test/step2.lua
```

[packdir.lua](https://github.com/schwarzlichtbezirk/wpk/blob/master/test/packdir.lua) script has function that can be used to put to package directory with original tree hierarchy.

# WPK API usage

See [godoc](https://godoc.org/github.com/schwarzlichtbezirk/wpk) with API description, and [wpk_test.go](https://github.com/schwarzlichtbezirk/wpk/blob/master/wpk_test.go) for usage samples.

On your program initialisation open prepared wpk-package by [Package.Read](https://godoc.org/github.com/schwarzlichtbezirk/wpk#Package.Read) call. It reads tags sets of package on this call and has no any others reading of tags sets later. [File](https://godoc.org/github.com/schwarzlichtbezirk/wpk#File) structure helps you to implement custom [fs.FS](https://golang.org/pkg/io/fs/#FS) to provide local file system and route it by [http.FileServer](https://golang.org/pkg/net/http/#FileServer). **`wpk/bulk`**, **`wpk/mmap`** and **`wpk/fsys`** modules already has file system interface implementation.
