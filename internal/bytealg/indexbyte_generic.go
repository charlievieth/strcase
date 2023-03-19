//go:build !s390x && !wasm && !ppc64x && !amd64 && !arm64
// +build !s390x,!wasm,!ppc64x,!amd64,!arm64

// Simple implementations for arch's that don't supper SIMD or
// for which the stdlib doesn't use SIMD.

// NOTE(cev): the arch's deduced as having non-SIMD implementations of
// IndexByte was lazily created by me browsing the assembly in
// internal/bytealg/indexbyte_*.s for go1.20.
//
// TLDR: apart from "arm" the arch build tags here a guess and should be
// re-visited in future release / by anyone with access to the hardware.

package bytealg

import (
	"bytes"
	"strings"
)

// The arm implementations of {bytes,strings}.IndexByte do not use SIMD
// so use a simple loop for the case-insensitive search.A

// simple implementations for when the standard lib doesn't use SIMD

func isAlpha(c byte) bool {
	return 'A' <= c && c <= 'Z' || 'a' <= c && c <= 'z'
}

func IndexByte(s []byte, c byte) int {
	if !isAlpha(c) {
		return bytes.IndexByte(s, c)
	}
	c |= ' '
	for i, cc := range s {
		if cc|' ' == c {
			return i
		}
	}
	return -1
}

func IndexByteString(s string, c byte) int {
	if !isAlpha(c) {
		return strings.IndexByte(s, c)
	}
	c |= ' '
	for i := 0; i < len(s); i++ {
		if s[i]|' ' == c {
			return i
		}
	}
	return -1
}
