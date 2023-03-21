//go:build !amd64 && !arm64
// +build !amd64,!arm64

package bytealg

import (
	"math/bits"
	"unicode/utf8"
	"unsafe"
)

func IndexNonASCII(s string) int {
	// TODO: see if we should use aligned loads
	const wordSize = int(unsafe.Sizeof(uintptr(0)))

	if len(s) < 8 {
		for i := 0; i < len(s); i++ {
			if s[i] >= utf8.RuneSelf {
				return i
			}
		}
		return -1
	}

	b := *(*[]byte)(unsafe.Pointer(&s))
	n := len(b) / wordSize
	sliceHeader := struct {
		p   unsafe.Pointer
		len int
		cap int
	}{unsafe.Pointer(&b[0]), n, n}

	var i int
	if wordSize == 8 {
		const mask64 uint64 = 0x8080808080808080
		us := *(*[]uint64)(unsafe.Pointer(&sliceHeader))
		for i := 0; i < len(us); i++ {
			if m := us[i] & mask64; m != 0 {
				return i*wordSize + bits.TrailingZeros64(m)/8
			}
		}
		i *= wordSize

	} else /* wordSize == 4 */ {
		const mask32 uint32 = 0x80808080
		us := *(*[]uint32)(unsafe.Pointer(&sliceHeader))
		for i := 0; i < len(us); i++ {
			if m := us[i] & mask32; m != 0 {
				return i*wordSize + bits.TrailingZeros32(m)/8
			}
		}
		i *= wordSize
	}

	// Check remaining bytes
	for ; i < len(s); i++ {
		if s[i] >= utf8.RuneSelf {
			return i
		}
	}
	return -1
}
