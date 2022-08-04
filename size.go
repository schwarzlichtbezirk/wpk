package wpk

import (
	"encoding/binary"
)

// Size-dependent types definitions.

type (
	FOffset_t = uint64
	FSize_t   = uint64
	FID_t     = uint32
	// FOffset_i - nested file offset type, can be 32, 64 bit integer. FOffset_t >= FSize_t.
	FOffset_i interface{ uint32 | uint64 }
	// FSize_i - nested file size type, can be 32, 64 bit integer.
	FSize_i interface{ uint32 | uint64 }
	// FID_i - file index/identifier type, can be 16, 32, 64 bit integer.
	FID_i interface{ uint16 | uint32 | uint64 }
	// TID_i - tag identifier type, can be 8, 16, 32 bit integer. TID_t <= TSSize_t.
	TID_i interface{ uint8 | uint16 | uint32 }
	// TSize_i - tag size type, can be 8, 16, 32 bit integer. TSize_t <= TSSize_t.
	TSize_i interface{ uint8 | uint16 | uint32 }
	// TSSize_i - tagset size/offset type, can be 16, 32 bit integer.
	TSSize_i interface{ uint16 | uint32 }
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

func FOffset_r[T FOffset_i](b []byte) (ret T) {
	return Uint_r[T](b)
}

func FSize_r[T FSize_i](b []byte) (ret T) {
	return Uint_r[T](b)
}

func FID_r[T FID_i](b []byte) (ret T) {
	return Uint_r[T](b)
}

func TID_r[T TID_i](b []byte) (ret T) {
	return Uint_r[T](b)
}

func TSize_r[T TSize_i](b []byte) (ret T) {
	return Uint_r[T](b)
}

func TSSize_r[T TSSize_i](b []byte) (ret T) {
	return Uint_r[T](b)
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

func FOffset_w[T FOffset_i](b []byte, v T) {
	Uint_w(b, v)
}

func FSize_w[T FSize_i](b []byte, v T) {
	Uint_w(b, v)
}

func FID_w[T FID_i](b []byte, v T) {
	Uint_w(b, v)
}

func TID_w[T TID_i](b []byte, v T) {
	Uint_w(b, v)
}

func TSize_w[T TSize_i](b []byte, v T) {
	Uint_w(b, v)
}

func TSSize_w[T TSSize_i](b []byte, v T) {
	Uint_w(b, v)
}

// The End.
