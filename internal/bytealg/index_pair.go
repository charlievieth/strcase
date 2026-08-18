// Copyright 2026 Charlie Vieth. All rights reserved.
// Use of this source code is governed by the MIT license.

package bytealg

// PackPair packs the lower and upper case forms of ASCII bytes c0 and c1
// for use with IndexPairFold.
func PackPair(c0, c1 byte) uint32 {
	c0l, c0u := caseForms(c0)
	c1l, c1u := caseForms(c1)
	return uint32(c0l) | uint32(c0u)<<8 | uint32(c1l)<<16 | uint32(c1u)<<24
}

func caseForms(c byte) (lower, upper byte) {
	if 'A' <= c && c <= 'Z' {
		return c + 'a' - 'A', c
	}
	if 'a' <= c && c <= 'z' {
		return c, c - ('a' - 'A')
	}
	return c, c
}
