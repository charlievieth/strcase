package strcase

import "strings"

// hasPrefixASCII tests whether the string s begins with prefix.
func hasPrefixASCII(s, prefix string) bool {
	if len(s) >= len(prefix) {
		for i := 0; i < len(prefix); i++ {
			if _lower[s[i]] != _lower[prefix[i]] {
				return false
			}
		}
		return true
	}
	return false
}

func bruteForceIndexASCII(s, substr string) int {
	c0 := _lower[substr[0]]
	c1 := _lower[substr[1]]
	t := len(s) - len(substr) + 1
	for i := 0; i < t; i++ {
		if s0 := s[i]; s0 != c0 && _lower[s0] != c0 {
			continue
		}
		if s1 := s[i+1]; s1 != c1 && _lower[s1] != c1 {
			continue
		}
		if hasPrefixASCII(s[i+2:], substr[2:]) {
			return i
		}
	}
	return -1
}

func indexByteASCII(s string, c byte) int {
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

func shortIndexASCII(s, substr string) int {
	n := len(substr)
	c0 := _lower[substr[0]]
	c1 := _lower[substr[1]]
	i := 0
	t := len(s) - n + 1
	fails := 0
	for i < t {
		if _lower[s[i]] != c0 {
			// IndexByte is faster than bytealg.IndexString, so use it as long as
			// we're not getting lots of false positives.
			o := indexByteASCII(s[i+1:t], c0)
			if o < 0 {
				return -1
			}
			i += o + 1
		}
		if _lower[s[i+1]] == c1 && equalASCII(s[i:i+n], substr) {
			return i
		}
		fails++
		i++
		if fails > cutover(i) {
			// FIXME: use Rabin-Karp
			r := bruteForceIndexASCII(s[i:], substr)
			if r >= 0 {
				return r + i
			}
			return -1
		}
	}
	return -1
}

func indexASCII(s, substr string) int {
	n := len(substr)
	c0 := _lower[substr[0]]
	c1 := _lower[substr[1]]
	i := 0
	t := len(s) - n + 1
	fails := 0
	for i < t {
		if _lower[s[i]] != c0 {
			o := IndexByte(s[i+1:t], c0)
			if o < 0 {
				return -1
			}
			i += o + 1
		}
		if _lower[s[i+1]] == c1 && s[i:i+n] == substr {
			return i
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
		if _lower[s[i]] != _lower[t[i]] {
			return false
		}
	}
	return true
}
