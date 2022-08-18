package wpk

import (
	"encoding/binary"
	"io"
)

// Size-dependent types definitions.

type (
	// TID_i - tag identifier type, can be 8, 16, 32 bit integer. TID <= TSSize.
	TID_i interface{ uint8 | uint16 | uint32 }
	// TSize_i - tag size type, can be 8, 16, 32 bit integer. TSize <= TSSize.
	TSize_i interface{ uint8 | uint16 | uint32 }
)

func ReadUintBuf(b []byte) uint {
	switch len(b) {
	case 1:
		return uint(b[0])
	case 2:
		return uint(binary.LittleEndian.Uint16(b))
	case 4:
		return uint(binary.LittleEndian.Uint32(b))
	case 8:
		return uint(binary.LittleEndian.Uint64(b))
	default:
		panic("undefined condition")
	}
}

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

func ReadUint(r io.Reader, l byte) (data uint, err error) {
	var buf = make([]byte, l)
	_, err = r.Read(buf)
	data = ReadUintBuf(buf)
	return
}

func WriteUint(w io.Writer, data uint, l byte) (err error) {
	var buf = make([]byte, l)
	WriteUintBuf(buf, data)
	_, err = w.Write(buf)
	return
}

// The End.
