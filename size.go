package wpk

import (
	"encoding/binary"
	"io"
)

// Size-dependent types definitions.

type (
	FOffset_t = uint64
	FSize_t   = uint64
	FID_t     = uint32
	// FOffset_i - nested file offset type, can be 32, 64 bit integer. FOffset >= FSize.
	FOffset_i interface{ uint32 | uint64 }
	// FSize_i - nested file size type, can be 32, 64 bit integer.
	FSize_i interface{ uint32 | uint64 }
	// FID_i - file index/identifier type, can be 16, 32, 64 bit integer.
	FID_i interface{ uint16 | uint32 | uint64 }
	// TID_i - tag identifier type, can be 8, 16, 32 bit integer. TID <= TSSize.
	TID_i interface{ uint8 | uint16 | uint32 }
	// TSize_i - tag size type, can be 8, 16, 32 bit integer. TSize <= TSSize.
	TSize_i interface{ uint8 | uint16 | uint32 }
)

func Uint_l[T uint8 | uint16 | uint32 | uint64]() int {
	var v T
	switch any(v).(type) {
	case uint8:
		return 1
	case uint16:
		return 2
	case uint32:
		return 4
	case uint64:
		return 8
	default:
		panic("unreachable condition")
	}
}

func Uint_r[T uint8 | uint16 | uint32 | uint64](b []byte) (ret T) {
	switch any(ret).(type) {
	case uint8:
		return T(b[0])
	case uint16:
		return T(binary.LittleEndian.Uint16(b))
	case uint32:
		return T(binary.LittleEndian.Uint32(b))
	case uint64:
		return T(binary.LittleEndian.Uint64(b))
	default:
		panic("unreachable condition")
	}
}

func Uint_w[T uint8 | uint16 | uint32 | uint64](b []byte, v T) {
	switch v := any(v).(type) {
	case uint8:
		b[0] = v
	case uint16:
		binary.LittleEndian.PutUint16(b, v)
	case uint32:
		binary.LittleEndian.PutUint32(b, v)
	case uint64:
		binary.LittleEndian.PutUint64(b, v)
	default:
		panic("unreachable condition")
	}
}

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
	case 1:
		b[0] = byte(v)
	case 2:
		binary.LittleEndian.PutUint16(b, uint16(v))
	case 4:
		binary.LittleEndian.PutUint32(b, uint32(v))
	case 8:
		binary.LittleEndian.PutUint64(b, uint64(v))
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
