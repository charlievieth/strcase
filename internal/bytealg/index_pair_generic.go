// Copyright 2026 Charlie Vieth. All rights reserved.
// Use of this source code is governed by the MIT license.

//go:build !arm64

package bytealg

// NativeIndexPair is true if we have a fast native (assembly)
// implementation of IndexPairFold.
const NativeIndexPair = false

// IndexPairFold returns the index of the first position i in s where s[i]
// matches c0 and s[i+1] matches c1, where the four bytes packed into pair
// are the lower and upper case forms of c0 and c1 (see PackPair), or -1 if
// there is no such position.
func IndexPairFold(s string, pair uint32) int {
	c0l := byte(pair)
	c0u := byte(pair >> 8)
	c1l := byte(pair >> 16)
	c1u := byte(pair >> 24)
	for i := 0; i+1 < len(s); i++ {
		if c := s[i]; c == c0l || c == c0u {
			if c := s[i+1]; c == c1l || c == c1u {
				return i
			}
		}
	}
	return -1
}

// LastIndexPairFold returns the index of the last position i in s where s[i]
// matches c0 and s[i+1] matches c1, where the four bytes packed into pair
// are the lower and upper case forms of c0 and c1 (see PackPair), or -1 if
// there is no such position.
func LastIndexPairFold(s string, pair uint32) int {
	c0l := byte(pair)
	c0u := byte(pair >> 8)
	c1l := byte(pair >> 16)
	c1u := byte(pair >> 24)
	for i := len(s) - 2; i >= 0; i-- {
		if c := s[i]; c == c0l || c == c0u {
			if c := s[i+1]; c == c1l || c == c1u {
				return i
			}
		}
	}
	return -1
}
