package wpk

// ToSlash brings filenames to true slashes
// without superfluous allocations if it possible.
func ToSlash(s string) string {
	var b = S2B(s)
	var bc = b
	var c bool
	for i, v := range b {
		if v == '\\' {
			if !c {
				bc, c = []byte(s), true
			}
			bc[i] = '/'
		}
	}
	return B2S(bc)
}

// ToLower is high performance function to bring filenames to lower case in ASCII
// without superfluous allocations if it possible.
func ToLower(s string) string {
	var b = S2B(s)
	var bc = b
	var c bool
	for i, v := range b {
		if v >= 'A' && v <= 'Z' {
			if !c {
				bc, c = []byte(s), true
			}
			bc[i] |= 0x20
		}
	}
	return B2S(bc)
}

// ToUpper is high performance function to bring filenames to upper case in ASCII
// without superfluous allocations if it possible.
func ToUpper(s string) string {
	var b = S2B(s)
	var bc = b
	var c bool
	for i, v := range b {
		if v >= 'a' && v <= 'z' {
			if !c {
				bc, c = []byte(s), true
			}
			bc[i] &= 0xdf
		}
	}
	return B2S(bc)
}

// ToKey is high performance function to bring filenames to lower case in ASCII
// and true slashes at once without superfluous allocations if it possible.
func ToKey(s string) string {
	var b = S2B(s)
	var bc = b
	var c bool
	for i, v := range b {
		if v >= 'A' && v <= 'Z' {
			if !c {
				bc, c = []byte(s), true
			}
			bc[i] |= 0x20
		} else if v == '\\' {
			if !c {
				bc, c = []byte(s), true
			}
			bc[i] = '/'
		}
	}
	return B2S(bc)
}
