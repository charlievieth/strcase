//go:build (!amd64 && !arm64) || (arm64 && !go1.16)
// +build !amd64,!arm64 arm64,!go1.16

// //go:build s390x || ppc64 || (arm64 && !go1.16)
// // +build s390x ppc64 arm64,!go1.16

package bytealg

import "unicode/utf8"

func IndexByteNonASCII(b []byte) int {
	for i := 0; i < len(b); i++ {
		if b[i]&utf8.RuneSelf != 0 {
			return i
		}
	}
	return -1
}

func IndexNonASCII(s string) int {
	for i := 0; i < len(s); i++ {
		if s[i]&utf8.RuneSelf != 0 {
			return i
		}
	}
	return -1
}
