package wpk

import (
	"io"
)

// ReadUintBuf reads unsigned integer from buffer of predefined size.
// Dimension of integer depended from size of buffer, size can be 1, 2, 4, 8.
func ReadUintBuf(b []byte) (r uint) {
	switch len(b) {
	case 8:
		r |= uint(b[7]) << 56
		r |= uint(b[6]) << 48
		r |= uint(b[5]) << 40
		r |= uint(b[4]) << 32
		fallthrough
	case 4:
		r |= uint(b[3]) << 24
		r |= uint(b[2]) << 16
		fallthrough
	case 2:
		r |= uint(b[1]) << 8
		fallthrough
	case 1:
		r |= uint(b[0])
	default:
		panic("undefined condition")
	}
	return
}

// WriteUintBuf writes unsigned integer into buffer with predefined size.
// Size of buffer can be 1, 2, 4, 8.
func WriteUintBuf(b []byte, v uint) {
	switch len(b) {
	case 8:
		b[7] = byte(v >> 56)
		b[6] = byte(v >> 48)
		b[5] = byte(v >> 40)
		b[4] = byte(v >> 32)
		fallthrough
	case 4:
		b[3] = byte(v >> 24)
		b[2] = byte(v >> 16)
		fallthrough
	case 2:
		b[1] = byte(v >> 8)
		fallthrough
	case 1:
		b[0] = byte(v)
	default:
		panic("undefined condition")
	}
}

// ReadUint reads from stream unsigned integer with given size in bytes.
// Size can be 1, 2, 4, 8.
func ReadUint(r io.Reader, l byte) (data uint, err error) {
	var buf = make([]byte, l)
	_, err = r.Read(buf)
	data = ReadUintBuf(buf)
	return
}

// WriteUint writes to stream given unsigned integer with given size in bytes.
// Size can be 1, 2, 4, 8.
func WriteUint(w io.Writer, data uint, l byte) (err error) {
	var buf = make([]byte, l)
	WriteUintBuf(buf, data)
	_, err = w.Write(buf)
	return
}

// The End.
