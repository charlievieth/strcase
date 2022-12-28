package strcase

import (
	"strings"
)

// WARN WARN WARN WARN WARN WARN
//
// Not significantly faster than Unicode version !!!
//
// 	benchmark                  old ns/op     new ns/op     delta
// 	BenchmarkIndexHard1-10     2114028       2070194       -2.07%
// 	BenchmarkIndexHard2-10     2795686       2541616       -9.09%
// 	BenchmarkIndexHard3-10     2412442       2170544       -10.03%
// 	BenchmarkIndexHard4-10     1746630       1735745       -0.62%
//
// WARN WARN WARN WARN WARN WARN

func CompareASCII(s, t string) int {
	i := 0
	for ; i < len(s) && i < len(t); i++ {
		sr := s[i]
		tr := t[i]
		if sr == tr || _lower[sr] == _lower[tr] {
			continue
		}
		if _lower[sr] < _lower[tr] {
			return -1
		}
		return 1
	}
	return clamp(len(s) - len(t))
}

// HasPrefixASCII tests whether the string s begins with prefix.
func HasPrefixASCII(s, prefix string) bool {
	return len(s) >= len(prefix) && equalASCII(s[0:len(prefix)], prefix)
}

func toUpperLowerASCII(c byte) (upper, lower byte) {
	if 'A' <= c && c <= 'Z' {
		return c, c + 'a' - 'A'
	}
	if 'a' <= c && c <= 'z' {
		return c - 'a' - 'A', c
	}
	return c, c
}

func bruteForceIndexASCII(s, substr string) int {
	u0, l0 := toUpperLowerASCII(substr[0])
	u1, l1 := toUpperLowerASCII(substr[1])
	needle := substr[2:]
	t := len(s) - len(substr) + 1
	if u0 == l0 && u1 == l1 {
		for i := 0; i < t; i++ {
			if s0 := s[i]; s0 != l0 {
				continue
			}
			if s1 := s[i+1]; s1 != l1 {
				if s1 != l0 {
					i++
				}
				continue
			}
			if HasPrefixASCII(s[i+2:], needle) {
				return i
			}
		}
	} else {
		for i := 0; i < t; i++ {
			if s0 := s[i]; s0 != l0 && s0 != u0 {
				continue
			}
			if s1 := s[i+1]; s1 != l1 && s1 != u1 {
				if s1 != l0 && s1 != u0 {
					i++
				}
				continue
			}
			if HasPrefixASCII(s[i+2:], needle) {
				return i
			}
		}
	}
	return -1
}

func IndexByteASCII(s string, c byte) int {
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

// IndexASCII returns the index of the first instance of substr in s, or -1 if
// substr is not present in s.
func IndexASCII(s, substr string) int {
	n := len(substr)
	switch {
	case n == 0:
		return 0
	case n == 1:
		return IndexByte(s, substr[0])
	case n == len(s):
		if equalASCII(s, substr) {
			return 0
		}
		return -1
	case n <= maxLen:
		// WARN: 32 is for arm64 (see: bytealg.MaxLen)
		if len(s) <= maxBruteForce {
			return bruteForceIndexASCII(s, substr)
		}
		u0, l0 := toUpperLowerASCII(substr[0])
		u1, l1 := toUpperLowerASCII(substr[1])
		needle := substr[2:]
		t := len(s) - n + 1
		i := 0
		fails := 0
		for i < t {
			if s0 := s[i]; s0 != l0 && s0 != u0 {
				o := IndexByteASCII(s[i+1:t], l0)
				if o < 0 {
					return -1
				}
				i += o + 1
			}
			if s1 := s[i+1]; (s1 == l1 || s1 == u1) && equalASCII(s[i+2:i+n], needle) {
				return i
			}
			fails++
			i++
			// FIXME: this needs to be tuned since the brute force
			// performance is very different than the stdlibs.
			if fails > cutover(i) {
				r := bruteForceIndexASCII(s[i:], substr)
				if r >= 0 {
					return r + i
				}
				return -1
			}
		}
		return -1
	}
	u0, l0 := toUpperLowerASCII(substr[0])
	u1, l1 := toUpperLowerASCII(substr[1])
	needle := substr[2:]
	t := len(s) - n + 1
	i := 0
	fails := 0
	for i < t {
		if s0 := s[i]; s0 != l0 && s0 != u0 {
			o := IndexByteASCII(s[i+1:t], l0)
			if o < 0 {
				return -1
			}
			i += o + 1
		}
		if s1 := s[i+1]; s1 == l1 || s1 == u1 {
			if equalASCII(s[i+2:i+n], needle) {
				return i
			}
		}
		i++
		fails++
		if fails >= 4+i>>4 && i < t {
			// See comment in ../bytes/bytes.go.
			j := indexRabinKarp(s[i:], substr)
			if j < 0 {
				return -1
			}
			return i + j
		}
	}
	return -1
}

// hashStr returns the hash and the appropriate multiplicative
// factor for use in Rabin-Karp algorithm.
func hashStr(sep string) (uint32, uint32) {
	hash := uint32(0)
	for i := 0; i < len(sep); i++ {
		hash = hash*primeRK + uint32(_lower[sep[i]])
	}
	var pow, sq uint32 = 1, primeRK
	for i := len(sep); i > 0; i >>= 1 {
		if i&1 != 0 {
			pow *= sq
		}
		sq *= sq
	}
	return hash, pow
}

// indexRabinKarp uses the Rabin-Karp search algorithm to return the index of the
// first occurrence of substr in s, or -1 if not present.
func indexRabinKarp(s, substr string) int {
	// Rabin-Karp search
	hashss, pow := hashStr(substr)
	n := len(substr)
	var h uint32
	for i := 0; i < n; i++ {
		h = h*primeRK + uint32(_lower[s[i]])
	}
	if h == hashss && equalASCII(s[:n], substr) {
		return 0
	}
	for i := n; i < len(s); {
		h *= primeRK
		h += uint32(_lower[s[i]])
		h -= pow * uint32(_lower[s[i-n]])
		i++
		if h == hashss && equalASCII(s[i-n:i], substr) {
			return i - n
		}
	}
	return -1
}

func equalASCII(s, t string) bool {
	if len(s) != len(t) {
		return false
	}
	for i := 0; i < len(s); i++ {
		// NOTE: this is generally faster than checking if s[i] != t[i]
		// before lower casing the chars.
		if _lower[s[i]] != _lower[t[i]] {
			return false
		}
	}
	return true
}
