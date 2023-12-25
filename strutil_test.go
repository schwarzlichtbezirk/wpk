package wpk_test

import (
	"fmt"
	"os"
	"testing"
	"unsafe"

	"github.com/schwarzlichtbezirk/wpk"
)

func TestS2B(t *testing.T) {
	var s = "some string"
	var ps = unsafe.Pointer(unsafe.StringData(s))
	var b = []byte(s)
	var pb = unsafe.Pointer(unsafe.SliceData(b))
	if ps == pb {
		t.Error("string pointer is equal to pointer on new allocated bytes slice")
	}
	b = wpk.S2B(s)
	pb = unsafe.Pointer(unsafe.SliceData(b))
	if ps != pb {
		t.Error("string pointer is not equal to pointer on same bytes slice")
	}
}

func TestB2S(t *testing.T) {
	var b = []byte("some string")
	var pb = unsafe.Pointer(unsafe.SliceData(b))
	var s = string(b)
	var ps = unsafe.Pointer(unsafe.StringData(s))
	if pb == ps {
		t.Error("bytes slice pointer is equal to pointer on new allocated string")
	}
	s = wpk.B2S(b)
	ps = unsafe.Pointer(unsafe.StringData(s))
	if pb != ps {
		t.Error("bytes slice pointer is not equal to pointer on same string")
	}
}

func ExampleToSlash() {
	fmt.Println(wpk.ToSlash("C:\\Windows\\Temp"))
	// Output: C:/Windows/Temp
}

func ExampleToLower() {
	fmt.Println(wpk.ToLower("C:\\Windows\\Temp"))
	// Output: c:\windows\temp
}

func ExampleToUpper() {
	fmt.Println(wpk.ToUpper("C:\\Windows\\Temp"))
	// Output: C:\WINDOWS\TEMP
}

func ExampleToKey() {
	fmt.Println(wpk.ToKey("C:\\Windows\\Temp"))
	// Output: c:/windows/temp
}

func ExampleJoinPath() {
	fmt.Println(wpk.JoinPath("dir", "base.ext"))
	fmt.Println(wpk.JoinPath("dir/", "base.ext"))
	fmt.Println(wpk.JoinPath("dir", "/base.ext"))
	fmt.Println(wpk.JoinPath("dir/", "/base.ext"))
	// Output:
	// dir/base.ext
	// dir/base.ext
	// dir/base.ext
	// dir/base.ext
}

func ExampleJoinFilePath() {
	fmt.Println(wpk.JoinFilePath("dir/", "base.ext"))
	fmt.Println(wpk.JoinFilePath("dir", "/base.ext"))
	fmt.Println(wpk.JoinFilePath("dir/", "/base.ext"))
	// Output:
	// dir/base.ext
	// dir/base.ext
	// dir/base.ext
}

func ExampleBaseName() {
	fmt.Println(wpk.PathName("C:\\Windows\\system.ini"))
	fmt.Println(wpk.PathName("/go/bin/wpkbuild_win_x64.exe"))
	fmt.Println(wpk.PathName("wpkbuild_win_x64.exe"))
	fmt.Println(wpk.PathName("/go/bin/wpkbuild_linux_x64"))
	fmt.Println(wpk.PathName("wpkbuild_linux_x64"))
	fmt.Printf("'%s'\n", wpk.PathName("/go/bin/"))
	// Output:
	// system
	// wpkbuild_win_x64
	// wpkbuild_win_x64
	// wpkbuild_linux_x64
	// wpkbuild_linux_x64
	// ''
}

func ExampleEnvfmt() {
	os.Setenv("VAR", "/go")
	// successful patterns
	fmt.Println(wpk.Envfmt("$VAR/bin/", nil))
	fmt.Println(wpk.Envfmt("${VAR}/bin/", nil))
	fmt.Println(wpk.Envfmt("%VAR%/bin/", nil))
	fmt.Println(wpk.Envfmt("/home$VAR", nil))
	fmt.Println(wpk.Envfmt("/home%VAR%", map[string]string{"VAR": "/any/path"}))
	fmt.Println(wpk.Envfmt("$VAR%VAR%${VAR}", nil))
	// patterns with unknown variable
	fmt.Println(wpk.Envfmt("$VYR/bin/", nil))
	fmt.Println(wpk.Envfmt("${VAR}/${_foo_}", nil))
	// patterns with errors
	fmt.Println(wpk.Envfmt("$VAR$/bin/", nil))
	fmt.Println(wpk.Envfmt("${VAR/bin/", nil))
	fmt.Println(wpk.Envfmt("%VAR/bin/", nil))
	fmt.Println(wpk.Envfmt("/home${VAR", nil))
	// Output:
	// /go/bin/
	// /go/bin/
	// /go/bin/
	// /home/go
	// /home/any/path
	// /go/go/go
	// $VYR/bin/
	// /go/${_foo_}
	// /go$/bin/
	// ${VAR/bin/
	// %VAR/bin/
	// /home${VAR
}
