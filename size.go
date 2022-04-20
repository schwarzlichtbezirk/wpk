package wpk

import (
	"encoding/binary"
)

// Size-dependent types definitions.

type (
	// FOffset_t - nested file offset type.
	FOffset_t uint64 // can be 32, 64 bit integer.
	// FSize_t - nested file size type.
	FSize_t uint64 // can be 32, 64 bit integer.
	// FID_t - file index/identifier type.
	FID_t uint32 // can be 16, 32, 64 bit integer.
	// TID_t - tag identifier type.
	TID_t uint16 // can be 8, 16, 32 bit integer.
	// TSize_t - tag size type.
	TSize_t uint16 // can be 8, 16, 32 bit integer.
	// TSSize_t - tagset size/offset type.
	TSSize_t uint16 // can be 16, 32 bit integer.
)

const (
	// FOffset_l - nested file type length; FOffset_l >= FSize_l.
	FOffset_l = 8
	// FSize_l - nested file size type length.
	FSize_l = 8
	// FID_l - file index/identifier type length.
	FID_l = 4
	// TID_l - tag identifier type length; TID_l <= TSSize_l.
	TID_l = 2
	// TSize_l - tag size type length; TSize_l <= TSize_l.
	TSize_l = 2
	// TSSize_l - tagset size/offset type length.
	TSSize_l = 2
)

var (
	FOffset_r = func(b []byte) FOffset_t { return FOffset_t(binary.LittleEndian.Uint64(b)) }
	FSize_r   = func(b []byte) FSize_t { return FSize_t(binary.LittleEndian.Uint64(b)) }
	FID_r     = func(b []byte) FID_t { return FID_t(binary.LittleEndian.Uint32(b)) }
	TID_r     = func(b []byte) TID_t { return TID_t(binary.LittleEndian.Uint16(b)) }
	TSize_r   = func(b []byte) TSize_t { return TSize_t(binary.LittleEndian.Uint16(b)) }
	TSSize_r  = func(b []byte) TSSize_t { return TSSize_t(binary.LittleEndian.Uint16(b)) }
	FOffset_w = func(b []byte, v FOffset_t) { binary.LittleEndian.PutUint64(b, uint64(v)) }
	FSize_w   = func(b []byte, v FSize_t) { binary.LittleEndian.PutUint64(b, uint64(v)) }
	FID_w     = func(b []byte, v FID_t) { binary.LittleEndian.PutUint32(b, uint32(v)) }
	TID_w     = func(b []byte, v TID_t) { binary.LittleEndian.PutUint16(b, uint16(v)) }
	TSize_w   = func(b []byte, v TSize_t) { binary.LittleEndian.PutUint16(b, uint16(v)) }
	TSSize_w  = func(b []byte, v TSSize_t) { binary.LittleEndian.PutUint16(b, uint16(v)) }
)

// The End.
