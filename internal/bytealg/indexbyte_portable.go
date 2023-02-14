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

	// TODO: calculate the optimal cutoff
	if n > 0 && len(s) >= 16 {
		s = s[:n] // limit search space
	}

	c ^= ' ' // swap case
	if o := bytes.IndexByte(s, c); n == -1 || (o != -1 && o < n) {
		n = o
	}
	return n
}

func IndexByteString(s string, c byte) int {
	n := strings.IndexByte(s, c)
	if n == 0 || !isAlpha(c) {
		return n
	}

	// TODO: calculate the optimal cutoff
	if n > 0 && len(s) >= 16 {
		s = s[:n] // limit search space
	}

	c ^= ' ' // swap case
	if o := strings.IndexByte(s, c); n == -1 || (o != -1 && o < n) {
		n = o
	}
	return n
}
