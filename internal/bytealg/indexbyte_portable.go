//go:build !amd64 && !arm64
// +build !amd64,!arm64

package bytealg

import (
	"bytes"
	"strings"
)

func isAlpha(c byte) bool {
	return 'A' <= c && c <= 'Z' || 'a' <= c && c <= 'z'
}

func IndexByte(s []byte, c byte) int {
	n := bytes.IndexByte(s, c)
	if n == 0 || !isAlpha(c) {
		return n
	}

	c ^= ' ' // swap case
	if s[0] == c {
		return 0
	}

	// TODO: calculate the optimal cutoff
	if n > 0 && len(s) >= 16 {
		s = s[:n] // limit search space
	}

	if o := bytes.IndexByte(s, c); n == -1 || (o != -1 && o < n) {
		n = o
	}
	return n
}

func IndexByteString(s string, c byte) int {
	// WARN: this is faster but should be done in assembly
	if len(s) <= 12 {
		if !isAlpha(c) {
			for i := 0; i < len(s); i++ {
				if s[i] == c {
					return i
				}
			}
		} else {
			c |= ' '
			for i := 0; i < len(s); i++ {
				if s[i]|' ' == c {
					return i
				}
			}
		}
		return -1
	}
	n := strings.IndexByte(s, c)
	if n == 0 || !isAlpha(c) {
		return n
	}

	c ^= ' ' // swap case
	if s[0] == c {
		return 0
	}

	// TODO: calculate the optimal cutoff
	if n > 0 && len(s) >= 16 {
		s = s[:n] // limit search space
	}

	if o := strings.IndexByte(s, c); n == -1 || (o != -1 && o < n) {
		n = o
	}
	return n
}

func IndexByteString_OLD(s string, c byte) int {
	n := strings.IndexByte(s, c)
	if n == 0 || !isAlpha(c) {
		return n
	}

	c ^= ' ' // swap case
	if s[0] == c {
		return 0
	}

	// TODO: calculate the optimal cutoff
	if n > 0 && len(s) >= 16 {
		s = s[:n] // limit search space
	}

	if o := strings.IndexByte(s, c); n == -1 || (o != -1 && o < n) {
		n = o
	}
	return n
}

// TODO: check for fast IndexByte at runtime ???
//
// NOTE: as of Go 1.20 the IndexByte appears to be slow for:
// 	arm, riscv, mips, mips64, and loong64
/*
func hasFastIndexByte() bool {
	const _s32 = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	const s = _s32 + _s32
	t1 := time.Now()
	for i := 0; i < 128; i++ {
		_ = naiveIndexByte(s, 'b')
	}
	d1 := time.Since(t1)
	t2 := time.Now()
	for i := 0; i < 128; i++ {
		_ = strings.IndexByte(s, 'b')
	}
	d2 := time.Since(t2)
	fmt.Printf("%s: d1: %s d2: %s\n", time.Since(t1), d1, d2)
	return d2*2 < d1
}
*/
