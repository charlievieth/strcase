//go:build amd64
// +build amd64

package bytealg

import (
	"bytes"
	"strings"
)

// Make golangci-lint think these functions are accessed since it
// cannot see accesses in assembly.
var _ = countGeneric
var _ = countGenericString

// A backup implementation to use by assembly.
func countGeneric(b []byte, c byte) int {
	if !('A' <= c && c <= 'Z' || 'a' <= c && c <= 'z') {
		return bytes.Count(b, []byte{c})
	}
	n := 0
	c |= ' '
	for _, cc := range b {
		if cc|' ' == c {
			n++
		}
	}
	return n
}

func countGenericString(s string, c byte) int {
	if !('A' <= c && c <= 'Z' || 'a' <= c && c <= 'z') {
		return strings.Count(s, string(c))
	}
	n := 0
	c |= ' '
	for i := 0; i < len(s); i++ {
		if s[i]|' ' == c {
			n++
		}
	}
	return n
}
