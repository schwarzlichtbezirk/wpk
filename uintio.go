package wpk

import (
	"io"
	"unsafe"
)

func GetU16(b []byte) uint16 {
	_ = b[1]
	return *((*uint16)(unsafe.Pointer(unsafe.SliceData(b))))
}

func GetU32(b []byte) uint32 {
	_ = b[3]
	return *((*uint32)(unsafe.Pointer(unsafe.SliceData(b))))
}

func GetU64(b []byte) uint64 {
	_ = b[7]
	return *((*uint64)(unsafe.Pointer(unsafe.SliceData(b))))
}

func GetF32(b []byte) float32 {
	_ = b[3]
	return *((*float32)(unsafe.Pointer(unsafe.SliceData(b))))
}

func GetF64(b []byte) float64 {
	_ = b[7]
	return *((*float64)(unsafe.Pointer(unsafe.SliceData(b))))
}

func SetU16(b []byte, u uint16) {
	_ = b[1]
	*((*uint16)(unsafe.Pointer(unsafe.SliceData(b)))) = u
}

func SetU32(b []byte, u uint32) {
	_ = b[3]
	*((*uint32)(unsafe.Pointer(unsafe.SliceData(b)))) = u
}

func SetU64(b []byte, u uint64) {
	_ = b[7]
	*((*uint64)(unsafe.Pointer(unsafe.SliceData(b)))) = u
}

func SetF32(b []byte, f float32) {
	_ = b[3]
	*((*float32)(unsafe.Pointer(unsafe.SliceData(b)))) = f
}

func SetF64(b []byte, f float64) {
	_ = b[7]
	*((*float64)(unsafe.Pointer(unsafe.SliceData(b)))) = f
}

func ReadU16(r io.Reader) (u uint16, err error) {
	_, err = r.Read(unsafe.Slice((*byte)(unsafe.Pointer(&u)), 2))
	return
}

func ReadU32(r io.Reader) (u uint32, err error) {
	_, err = r.Read(unsafe.Slice((*byte)(unsafe.Pointer(&u)), 4))
	return
}

func ReadU64(r io.Reader) (u uint64, err error) {
	_, err = r.Read(unsafe.Slice((*byte)(unsafe.Pointer(&u)), 8))
	return
}

func ReadF32(r io.Reader) (f float32, err error) {
	_, err = r.Read(unsafe.Slice((*byte)(unsafe.Pointer(&f)), 4))
	return
}

func ReadF64(r io.Reader) (f float64, err error) {
	_, err = r.Read(unsafe.Slice((*byte)(unsafe.Pointer(&f)), 8))
	return
}

func WriteU16(w io.Writer, u uint16) (err error) {
	_, err = w.Write(unsafe.Slice((*byte)(unsafe.Pointer(&u)), 2))
	return
}

func WriteU32(w io.Writer, u uint32) (err error) {
	_, err = w.Write(unsafe.Slice((*byte)(unsafe.Pointer(&u)), 4))
	return
}

func WriteU64(w io.Writer, u uint64) (err error) {
	_, err = w.Write(unsafe.Slice((*byte)(unsafe.Pointer(&u)), 8))
	return
}

func WriteF32(w io.Writer, f float32) (err error) {
	_, err = w.Write(unsafe.Slice((*byte)(unsafe.Pointer(&f)), 4))
	return
}

func WriteF64(w io.Writer, f float64) (err error) {
	_, err = w.Write(unsafe.Slice((*byte)(unsafe.Pointer(&f)), 8))
	return
}

// The End.
