// Copyright 2023 Charlie Vieth. All rights reserved.
// Use of this source code is governed by the MIT license.

package strcase

import (
	"runtime"
	"strings"
	"unicode/utf8"

	"github.com/charlievieth/strcase/internal/bytealg"
)

// TODO: tune these and move to bytealg package
const maxBruteForce = 16 // substring length
const maxLen = 32        // subject length

// A foldPair stores Unicode case folding pairs
type foldPair struct {
	From uint32
	To   uint32
}

// TODO: rename to "foldCase"
//
// caseFold returns the Unicode simple case-fold for r, if one exists, or r
// unmodified, if one does not exist.
func caseFold(r rune) rune {
	// TODO: check if r is ASCII here?
	u := uint32(r)
	h := (u * _CaseFoldsSeed) >> _CaseFoldsShift
	p := _CaseFolds[h]
	if p.From == u {
		r = rune(p.To)
	}
	return r
}

// TODO: rename
func foldMap(r rune) *[4]uint16 {
	u := uint32(r)
	h := (u * _FoldMapSeed) >> _FoldMapShift
	p := &_FoldMap[h]
	if uint32(p[0]) == u {
		return p
	}
	return nil
}

func foldMapExcludingUpperLower(r rune) [2]rune {
	u := uint32(r)
	h := (u * _FoldMapSeed) >> _FoldMapShift
	p := &_FoldMapExcludingUpperLower[h]
	if uint32(p.r) == u {
		return [2]rune{rune(p.a[0]), rune(p.a[1])}
	}
	return [2]rune{}
}

func clamp(n int) int {
	if n < 0 {
		return -1
	}
	if n > 0 {
		return 1
	}
	return 0
}

// Compare returns an integer comparing two strings lexicographically
// ignoring case.
// The result will be 0 if a == b, -1 if a < b, and +1 if a > b.
func Compare(s, t string) int {
	// TODO: move next to hasPrefixUnicode
	i := 0
	for ; i < len(s) && i < len(t); i++ {
		sr := s[i]
		tr := t[i]
		if (sr|tr)&utf8.RuneSelf != 0 {
			goto hasUnicode
		}
		if sr == tr || _lower[sr] == _lower[tr] {
			continue
		}
		if _lower[sr] < _lower[tr] {
			return -1
		}
		return 1
	}
	return clamp(len(s) - len(t))

hasUnicode:
	s = s[i:]
	t = t[i:]
	for _, sr := range s {
		// If t is exhausted the strings are not equal.
		if len(t) == 0 {
			return 1
		}
		// Extract first rune from second string.
		var tr rune
		if t[0] < utf8.RuneSelf {
			tr, t = rune(_lower[t[0]]), t[1:]
		} else {
			r, size := utf8.DecodeRuneInString(t)
			tr, t = caseFold(r), t[size:]
		}
		// Easy case.
		if sr == tr || caseFold(sr) == tr {
			continue
		}
		return clamp(int(caseFold(sr)) - int(tr))
	}
	if len(t) == 0 {
		return 0
	}
	return -1
}

// EqualFold reports whether s and t, interpreted as UTF-8 strings,
// are equal under simple Unicode case-folding, which is a more general
// form of case-insensitivity.
//
// EqualFold is included for symmetry with the strings package and because
// our implementation is usually 2x faster than [strings.EqualFold].
func EqualFold(s, t string) bool {
	return Compare(s, t) == 0
}

func isAlpha(c byte) bool {
	return 'A' <= c && c <= 'Z' || 'a' <= c && c <= 'z'
}

// NB: we previously used a table that only contained the 128 ASCII characters
// and a mask, but this is a about ~6% faster.
var _lower = [256]byte{
	0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20,
	21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, ' ', '!', '"', '#', '$', '%',
	'&', '\'', '(', ')', '*', '+', ',', '-', '.', '/', '0', '1', '2', '3', '4',
	'5', '6', '7', '8', '9', ':', ';', '<', '=', '>', '?', '@', 'a', 'b', 'c',
	'd', 'e', 'f', 'g', 'h', 'i', 'j', 'k', 'l', 'm', 'n', 'o', 'p', 'q', 'r',
	's', 't', 'u', 'v', 'w', 'x', 'y', 'z', '[', '\\', ']', '^', '_', '`', 'a',
	'b', 'c', 'd', 'e', 'f', 'g', 'h', 'i', 'j', 'k', 'l', 'm', 'n', 'o', 'p',
	'q', 'r', 's', 't', 'u', 'v', 'w', 'x', 'y', 'z', '{', '|', '}', '~', 127,
	128, 129, 130, 131, 132, 133, 134, 135, 136, 137, 138, 139, 140, 141, 142,
	143, 144, 145, 146, 147, 148, 149, 150, 151, 152, 153, 154, 155, 156, 157,
	158, 159, 160, 161, 162, 163, 164, 165, 166, 167, 168, 169, 170, 171, 172,
	173, 174, 175, 176, 177, 178, 179, 180, 181, 182, 183, 184, 185, 186, 187,
	188, 189, 190, 191, 192, 193, 194, 195, 196, 197, 198, 199, 200, 201, 202,
	203, 204, 205, 206, 207, 208, 209, 210, 211, 212, 213, 214, 215, 216, 217,
	218, 219, 220, 221, 222, 223, 224, 225, 226, 227, 228, 229, 230, 231, 232,
	233, 234, 235, 236, 237, 238, 239, 240, 241, 242, 243, 244, 245, 246, 247,
	248, 249, 250, 251, 252, 253, 254, 255,
}

// containsKelvin returns true if string s contains rune 'K' (Kelvin).
func containsKelvin(s string) bool {
	// TODO: it might be faster to check with IndexNonASCII first
	// then with Count.
	return len(s) > 0 && indexRuneCase(s, '\u212A') != -1
}

// HasPrefix tests whether the string s begins with prefix ignoring case.
func HasPrefix(s, prefix string) bool {
	ok, _ := hasPrefixUnicode(s, prefix)
	return ok
}

// hasPrefixUnicode returns if string s begins with prefix (ignoring case) and
// if all of s was consumed matching prefix (either before a match could be found
// or is prefix consumes all of s).
func hasPrefixUnicode(s, prefix string) (bool, bool) {
	// TODO: return an enum instead of two bools

	// The max difference in encoded lengths between cases is 2 bytes for
	// [kK] (1 byte) and Kelvin 'K' (3 bytes).
	n := len(s)
	if len(prefix) > n*3 || (len(prefix) > n*2 && !containsKelvin(prefix)) {
		return false, true
	}

	// ASCII fast path
	i := 0
	for ; i < len(s) && i < len(prefix); i++ {
		sr := s[i]
		tr := prefix[i]
		if (sr|tr)&utf8.RuneSelf != 0 {
			goto hasUnicode
		}
		if tr == sr || _lower[sr] == _lower[tr] {
			continue
		}
		return false, i == len(s)-1
	}
	// Check if we've exhausted s
	return i == len(prefix), i == len(s)

hasUnicode:
	s = s[i:]
	prefix = prefix[i:]
	for _, tr := range prefix {
		// If s is exhausted the strings are not equal.
		if len(s) == 0 {
			return false, true
		}
		var sr rune
		if s[0] < utf8.RuneSelf {
			sr, s = rune(_lower[s[0]]), s[1:]
		} else {
			r, size := utf8.DecodeRuneInString(s)
			sr, s = r, s[size:]
		}
		if tr == sr || caseFold(tr) == caseFold(sr) {
			continue
		}
		return false, len(s) == 0
	}
	return true, len(s) == 0 // s exhausted
}

// TrimPrefix returns s without the provided leading prefix string.
// If s doesn't start with prefix, s is returned unchanged.
func TrimPrefix(s, prefix string) string {
	// The max difference in encoded lengths between cases is 2 bytes for
	// [kK] (1 byte) and Kelvin 'K' (3 bytes).
	n := len(s)
	if n*3 < len(prefix) || (n*2 < len(prefix) && !containsKelvin(prefix)) {
		return s
	}

	// ASCII fast path
	i := 0
	for ; i < len(s) && i < len(prefix); i++ {
		sr := s[i]
		tr := prefix[i]
		if (sr|tr)&utf8.RuneSelf != 0 {
			goto hasUnicode
		}
		if tr == sr || _lower[sr] == _lower[tr] {
			continue
		}
		return s
	}
	return s[i:]

hasUnicode:
	ss := s
	s = s[i:]
	prefix = prefix[i:]
	for _, tr := range prefix {
		// If s is exhausted the strings are not equal.
		if len(s) == 0 {
			return ss
		}

		var sr rune
		if s[0] < utf8.RuneSelf {
			sr, s = rune(_lower[s[0]]), s[1:]
		} else {
			r, size := utf8.DecodeRuneInString(s)
			sr, s = caseFold(r), s[size:]
		}
		if tr == sr || caseFold(tr) == sr {
			continue
		}
		return ss
	}
	return s
}

// HasSuffix tests whether the string s ends with suffix.
func HasSuffix(s, suffix string) bool {
	ok, _, _ := hasSuffixUnicode(s, suffix)
	return ok
}

// hasSuffixUnicode returns if string s ends with suffix and if all of s was
// consumed matching suffix (either before a match could be found or is suffix
// consumes all of s).
func hasSuffixUnicode(s, suffix string) (bool, bool, int) {
	// The max difference in encoded lengths between cases is 2 bytes for
	// [kK] (1 byte) and Kelvin 'K' (3 bytes).
	nt := len(suffix)
	ns := len(s)
	if nt == 0 {
		return true, false, ns
	}
	if ns*3 < nt || (ns*2 < nt && !containsKelvin(suffix)) {
		return false, true, 0
	}

	t := suffix
	i := ns - 1
	j := nt - 1
	for ; i >= 0 && j >= 0; i, j = i-1, j-1 {
		sr := s[i]
		tr := t[j]
		if (sr|tr)&utf8.RuneSelf != 0 {
			goto hasUnicode
		}
		if tr == sr || _lower[sr] == _lower[tr] {
			continue
		}
		return false, i == 0, 0
	}
	return j == -1, i <= 0, i + 1

hasUnicode:
	s = s[:i+1]
	t = t[:j+1]
	for len(s) != 0 && len(t) != 0 {
		var sr, tr rune
		if n := len(s) - 1; s[n] < utf8.RuneSelf {
			sr, s = rune(_lower[s[n]]), s[:n]
		} else {
			r, size := utf8.DecodeLastRuneInString(s)
			sr, s = r, s[:len(s)-size]
		}
		if n := len(t) - 1; t[n] < utf8.RuneSelf {
			tr, t = rune(_lower[t[n]]), t[:n]
		} else {
			r, size := utf8.DecodeLastRuneInString(t)
			tr, t = r, t[:len(t)-size]
		}
		if sr == tr || caseFold(sr) == caseFold(tr) {
			continue
		}
		return false, len(s) == 0, 0
	}

	return len(t) == 0, len(s) == 0, len(s)
}

// TrimSuffix returns s without the provided trailing suffix string.
// If s doesn't end with suffix, s is returned unchanged.
func TrimSuffix(s, suffix string) string {
	if match, _, i := hasSuffixUnicode(s, suffix); match {
		return s[:i]
	}
	return s
}

// toUpperLower combines unicode.ToUpper and unicode.ToLower in one function.
func toUpperLower(r rune) (upper, lower rune, foundMapping bool) {
	if r <= utf8.RuneSelf {
		if 'A' <= r && r <= 'Z' {
			return r, r + ('a' - 'A'), true
		}
		if 'a' <= r && r <= 'z' {
			return r - ('a' - 'A'), r, true
		}
		return r, r, false
	}
	// Hash rune r and see if it's in the _UpperLower table.
	u := uint32(r)
	h := (u | u<<24) * _UpperLowerSeed
	p := &_UpperLower[h>>_UpperLowerShift]
	if p[0] == u || p[1] == u {
		return rune(p[0]), rune(p[1]), true
	}
	// Handle Unicode characters that do not equal
	// their upper and lower case forms.
	return toUpperLowerSpecial(r)
}

// bruteForceIndexUnicode performs a brute-force search for substr in s.
func bruteForceIndexUnicode(s, substr string) int {
	var u0, u1 rune
	var sz0, sz1 int
	if substr[0] < utf8.RuneSelf {
		u0, sz0 = rune(substr[0]), 1
	} else {
		u0, sz0 = utf8.DecodeRuneInString(substr)
	}
	if substr[sz0] < utf8.RuneSelf {
		u1, sz1 = rune(substr[sz0]), 1
	} else {
		u1, sz1 = utf8.DecodeRuneInString(substr[sz0:])
	}
	folds0 := foldMapExcludingUpperLower(u0)
	folds1 := foldMapExcludingUpperLower(u1)
	hasFolds0 := folds0[0] != 0
	hasFolds1 := folds1[0] != 0
	needle := substr[sz0+sz1:]

	// Ugly hack
	var l0, l1 rune
	if u0 != 'İ' {
		u0, l0, _ = toUpperLower(u0)
	} else {
		l0 = 'İ'
	}
	if u1 != 'İ' {
		u1, l1, _ = toUpperLower(u1)
	} else {
		l1 = 'İ'
	}

	// Limit search space.
	t := len(s) - len(substr)/3 + 2
	if t > len(s) {
		t = len(s)
	}
	switch {
	case !hasFolds0 && u0 == l0 && !hasFolds1 && u1 == l1:
		// Fast check for the first rune.
		i := strings.Index(s, substr[:sz0+sz1])
		if i < 0 {
			return -1
		}
		for i < t {
			var n0 int
			var r0 rune
			if s[i] < utf8.RuneSelf {
				r0, n0 = rune(s[i]), 1
			} else {
				r0, n0 = utf8.DecodeRuneInString(s[i:])
			}
			if r0 != u0 {
				i += n0
				continue
			}
			if i+n0 >= t {
				break
			}

			var n1 int
			var r1 rune
			if s[i+n0] < utf8.RuneSelf {
				r1, n1 = rune(s[i+n0]), 1
			} else {
				r1, n1 = utf8.DecodeRuneInString(s[i+n0:])
			}
			if r1 != u1 {
				i += n0
				if r1 != u0 {
					i += n1 // Skip 2 runes when possible
				}
				continue
			}

			match, noMore := hasPrefixUnicode(s[i+n0+n1:], needle)
			if match {
				return i
			}
			if noMore {
				break
			}
			i += n0
			if r1 != u0 {
				i += n1 // Skip 2 runes when possible
			}
		}
		return -1
	case !hasFolds0 && !hasFolds1:
		// TODO: check is adding a fast check for l0 and u0 is faster
		i := 0
		for i < t {
			var n0 int
			var r0 rune
			if s[i] < utf8.RuneSelf {
				r0, n0 = rune(s[i]), 1
			} else {
				r0, n0 = utf8.DecodeRuneInString(s[i:])
			}
			if r0 != u0 && r0 != l0 {
				i += n0
				continue
			}
			if i+n0 >= t {
				break
			}

			var n1 int
			var r1 rune
			if s[i+n0] < utf8.RuneSelf {
				r1, n1 = rune(s[i+n0]), 1
			} else {
				r1, n1 = utf8.DecodeRuneInString(s[i+n0:])
			}
			if r1 != u1 && r1 != l1 {
				i += n0
				if r1 != u0 && r1 != l0 {
					i += n1 // Skip 2 runes when possible
				}
				continue
			}

			match, noMore := hasPrefixUnicode(s[i+n0+n1:], needle)
			if match {
				return i
			}
			if noMore {
				break
			}
			i += n0
			if r1 != u0 && r1 != l0 {
				i += n1 // Skip 2 runes when possible
			}
		}
		return -1
	default:
		// TODO: see if there is a better cutoff to use
		i := 0
		for i < t {
			var n0 int
			var r0 rune
			if s[i] < utf8.RuneSelf {
				r0, n0 = rune(s[i]), 1
			} else {
				r0, n0 = utf8.DecodeRuneInString(s[i:])
			}
			if r0 != u0 && r0 != l0 {
				if !hasFolds0 || (r0 != folds0[0] && r0 != folds0[1]) {
					i += n0
					continue
				}
			}
			if i+n0 >= t {
				break
			}

			var n1 int
			var r1 rune
			if s[i+n0] < utf8.RuneSelf {
				r1, n1 = rune(s[i+n0]), 1
			} else {
				r1, n1 = utf8.DecodeRuneInString(s[i+n0:])
			}
			if r1 != u1 && r1 != l1 {
				if !hasFolds1 || (r1 != folds1[0] && r1 != folds1[1]) {
					i += n0
					if !hasFolds0 && r1 != u0 && r1 != l0 {
						i += n1 // Skip 2 runes when possible
					}
					continue
				}
			}

			match, noMore := hasPrefixUnicode(s[i+n0+n1:], needle)
			if match {
				return i
			}
			if noMore {
				break
			}
			i += n0
			if !hasFolds0 && r1 != u0 && r1 != l0 {
				i += n1 // Skip 2 runes when possible
			}
		}
		return -1
	}
}

// TODO: inline in Index
func indexUnicode(s, substr string) int {
	var u0, u1 rune
	var sz0, sz1 int
	if substr[0] < utf8.RuneSelf {
		u0, sz0 = rune(substr[0]), 1
	} else {
		u0, sz0 = utf8.DecodeRuneInString(substr)
	}
	if substr[sz0] < utf8.RuneSelf {
		u1, sz1 = rune(substr[sz0]), 1
	} else {
		u1, sz1 = utf8.DecodeRuneInString(substr[sz0:])
	}
	// hasFolds{0,1} should be rare so consider optimizing
	// the no folds case
	folds0 := foldMapExcludingUpperLower(u0)
	folds1 := foldMapExcludingUpperLower(u1)
	hasFolds0 := folds0[0] != 0
	hasFolds1 := folds1[0] != 0
	needle := substr[sz0+sz1:]

	// Ugly hack
	var l0, l1 rune
	if u0 != 'İ' {
		u0, l0, _ = toUpperLower(u0)
	} else {
		l0 = 'İ'
	}
	if u1 != 'İ' {
		u1, l1, _ = toUpperLower(u1)
	} else {
		l1 = 'İ'
	}

	fails := 0
	// TODO: see if we can stop sooner.
	t := len(s) - len(substr)/3 + 1
	if t > len(s) {
		t = len(s)
	}
	for i := 0; i < t; {
		var r0 rune
		var n0 int
		if s[i] < utf8.RuneSelf {
			r0, n0 = rune(s[i]), 1
		} else {
			r0, n0 = utf8.DecodeRuneInString(s[i:])
		}

		if r0 != u0 && r0 != l0 && (!hasFolds0 || (r0 != folds0[0] && r0 != folds0[1])) {
			var o, sz int
			if !hasFolds0 {
				o, sz = indexRune2(s[i+n0:], l0, u0)
			} else {
				// TODO: pass folds to indexRune so that we don't have to
				// look them up again.
				o, sz = indexRune(s[i+n0:], l0)
			}
			if o < 0 {
				return -1
			}
			i += o + n0
			n0 = sz // The rune we matched on might not be the same size as c0
		}

		if i+n0 >= t {
			return -1
		}

		var r1 rune
		var n1 int
		if s[i+n0] < utf8.RuneSelf {
			r1, n1 = rune(s[i+n0]), 1
		} else {
			r1, n1 = utf8.DecodeRuneInString(s[i+n0:])
		}

		if r1 == u1 || r1 == l1 || (hasFolds1 && (r1 == folds1[0] || r1 == folds1[1])) {
			match, exhausted := hasPrefixUnicode(s[i+n0+n1:], needle)
			if match {
				return i
			}
			if exhausted {
				return -1
			}
		}
		fails++
		i += n0

		// NB: After tuning this appears to be ideal across platforms.
		if fails >= 4+i>>4 && i < t {
			// From bytes/bytes.go:
			//
			// Give up on IndexByte, it isn't skipping ahead
			// far enough to be better than Rabin-Karp.
			// Experiments (using IndexPeriodic) suggest
			// the cutover is about 16 byte skips.
			// TODO: if large prefixes of sep are matching
			// we should cutover at even larger average skips,
			// because Equal becomes that much more expensive.
			// This code does not take that effect into account.
			//
			// NB(charlie): The above part about "Equal" being
			// more expensive is particularly relevant us since
			// our "Equal" is drastically more expensive than
			// the stdlibs.
			j := indexRabinKarpUnicode(s[i:], substr)
			if j < 0 {
				return -1
			}
			return i + j
		}
	}
	return -1
}

// nonLetterASCII checks if the first 32 bytes of s consist only of
// non-letter ASCII characters. This is used to quickly check if we
// can use strings.Index.
func nonLetterASCII(s string) bool {
	for i := 0; i < len(s); i++ {
		// NB: this is faster than using a lookup table
		c := int(s[i] | ' ') // simplify check for alpha
		if c >= utf8.RuneSelf || 'a' <= c && c <= 'z' {
			return false
		}
	}
	return true
}

// Index returns the index of the first instance of substr in s, or -1 if
// substr is not present in s.
func Index(s, substr string) int {
	n := len(substr)
	var r rune
	var size int
	if n > 0 {
		if substr[0] < utf8.RuneSelf {
			r, size = rune(substr[0]), 1
		} else {
			r, size = utf8.DecodeRuneInString(substr)
		}
	}
	switch {
	case n == 0:
		return 0
	case n == 1:
		return IndexByte(s, byte(r))
	case n == size:
		return IndexRune(s, r)
	case n >= len(s):
		if n > len(s)*3 {
			return -1
		}
		// Match here is possible due to upper/lower case runes
		// having different encoded sizes.
		//
		// Fast check to see if s contains the first character of substr.
		i := IndexRune(s, r)
		if i < 0 {
			return -1
		}
		// Reduce the search space
		s = s[i:]
		// Kelvin K is three times the size of ASCII [Kk] so we need
		// to check for it to see if the longer needle (substr) could
		// possibly match the shorter haystack (s).
		if n > len(s)*2 && !containsKelvin(substr) {
			return -1
		}
		// NB: until disproven this is sufficiently fast (and maybe fastest)
		if o := bruteForceIndexUnicode(s, substr); o != -1 {
			return o + i
		}
		return -1
	case n <= maxLen: // WARN: 32 is for arm64 (see: bytealg.MaxLen)
		// WARN: fast path if the sub-string is all non-alpha ASCII chars
		// FIXME: tune the cutoff here
		if nonLetterASCII(substr) {
			return strings.Index(s, substr)
		}
		// TODO: tune this
		if len(s) <= maxBruteForce {
			return bruteForceIndexUnicode(s, substr)
		}
		// fallthrough
	}
	return indexUnicode(s, substr)
}

// LastIndex returns the index of the last instance of substr in s, or -1 if
// substr is not present in s.
func LastIndex(s, substr string) int {
	n := len(substr)
	var u rune
	var size int
	if n > 0 {
		if substr[n-1] < utf8.RuneSelf {
			u, size = rune(substr[n-1]), 1
		} else {
			u, size = utf8.DecodeRuneInString(substr)
		}
	}
	switch {
	case n == 0:
		return len(s)
	case n == 1:
		return LastIndexByte(s, substr[0])
	case n == size:
		return lastIndexRune(s, u)
	case n >= len(s):
		if n > len(s)*3 {
			return -1
		}
		if n > len(s)*2 && !containsKelvin(substr) {
			return -1
		}
		// fallthrough
	}
	return indexRabinKarpRevUnicode(s, substr)
}

// IndexByte returns the index of the first instance of c (ignoring case) in s,
// or -1 if c is not present in s. Matching is case-insensitive and Unicode
// simple folding is used which means that bytes 'K' and 'k' match Kelvin 'K'
// bytes 'S' and 's' match 'ſ'. Therefore, string s may be scanned twice when
// c is [KkSs] (because the optimized assembly ASCII only).
//
// On amd64 and arm64 this is only ~20-25% slower than strings.IndexByte for
// small strings (<4M) and ~6% slower for larger strings.
// The slowdown for small strings is due to some additional initial overhead.
func IndexByte(s string, c byte) int {
	// TODO: the below quick check is only to improve benchmark performance for
	// small strings (where the overhead of this function and IndexByteString
	// not being inlined becomes noticeable ~1ns).
	// See if we can get this function inlined or reduce the overhead of
	// indexByte.
	//
	// Fast check for bytes that can't be folded to from Unicode chars.
	// This shaves one or two nanoseconds from IndexByte, which is
	// meaningful when s is small.
	switch c {
	case 'K', 'S', 'k', 's':
		i, _ := indexByte(s, c)
		return i
	default:
		return bytealg.IndexByteString(s, c)
	}
}

// IndexByte returns the index of the first instance of c (ignoring case) in s,
// or -1 if c is not present in s. Case matching is ASCII only, unlike
// [IndexByte] which is Unicode aware.
func IndexByteASCII(s string, c byte) int {
	return bytealg.IndexByteString(s, c)
}

// indexByte returns the index of the first instance of c in s, or -1 if c is
// not present in s and the size (in bytes) of the character matched (this is
// needed to handle matching [Kk] and [Ss] to multibyte characters 'K' and 'ſ').
func indexByte(s string, c byte) (int, int) {
	if len(s) == 0 {
		return -1, 1
	}

	n := bytealg.IndexByteString(s, c)

	// Special case for Unicode characters that map to ASCII.
	var r rune
	var sz int
	switch c {
	case 'K', 'k':
		r = 'K' // Kelvin K
		sz = 3
	case 'S', 's':
		r = 'ſ' // Latin small letter long S
		sz = 2
	default:
		return n, 1
	}

	// Search for Unicode characters that map to ASCII byte 'c'
	if n > 0 {
		if n < sz {
			return n, 1 // Matched c before a possible rune 'r'
		}
		s = s[:n] // Limit search space
	}

	// IndexNonASCII is fast very on amd64 and arm64 and runs in O(n) time
	// compared to IndexRune which can get tripped up if the text contains
	// many of the same bytes as the rune being searched for.
	if runtime.GOARCH == "amd64" || runtime.GOARCH == "arm64" {
		o := IndexNonASCII(s)
		if n == 0 || o < 0 {
			// Short-circuit: c is the first byte or the text is all ASCII.
			return n, 1
		}
		s = s[o:] // Limit search space to the start of non-ASCII runes.
		if 0 < n && n < sz {
			return n, 1 // Matched c before a possible rune 'r'
		}
		// Search for Unicode characters that map to ASCII byte 'c'
		if k := indexRuneCase(s, r); n == -1 || (k != -1 && k < n) {
			if k != -1 {
				k += o
			}
			return k, sz
		}
		return n, 1
	}
	// Non amd64/arm64 code (should be elided)
	if o := indexRuneCase(s, r); n == -1 || (o != -1 && o < n) {
		return o, sz
	}
	return n, 1
}

// LastIndexByte returns the index of the last instance of c in s, or -1
// if c is not present in s.
func LastIndexByte(s string, c byte) int {
	if len(s) == 0 {
		return -1
	}
	if !isAlpha(c) {
		return strings.LastIndexByte(s, c)
	}

	// Special case for Unicode characters that map to ASCII.
	var r rune
	switch c {
	case 'K', 'k':
		r = 'K'
	case 'S', 's':
		r = 'ſ'
	default:
		c |= ' ' // convert to lower case
		for i := len(s) - 1; i >= 0; i-- {
			if s[i]|' ' == c {
				return i
			}
		}
		return -1
	}

	// Handle ASCII characters with Unicode mappings
	c |= ' ' // convert to lower case
	for i := len(s); i > 0; {
		if s[i-1] < utf8.RuneSelf {
			i--
			if s[i]|' ' == c {
				return i
			}
		} else {
			sr, size := utf8.DecodeLastRuneInString(s[:i])
			i -= size
			if sr == r {
				return i
			}
		}
	}
	return -1
}

// IndexRune returns the index of the first instance of the Unicode code point
// r, or -1 if rune is not present in s.
// If r is utf8.RuneError, it returns the first instance of any
// invalid UTF-8 byte sequence.
func IndexRune(s string, r rune) int {
	// TODO: This is faster than strings.IndexRune when r is not ASCII
	// so make a PR to go/strings.

	// TODO: consider checking if r is ASCII here so that it can be inlined
	i, _ := indexRune(s, r)
	return i
}

// indexRune returns the index of the first instance of the Unicode code point
// r and the size of the rune that matched.
func indexRune(s string, r rune) (int, int) {
	// TODO: handle invalid runes
	switch {
	case 0 <= r && r < utf8.RuneSelf:
		// TODO: Check if we can use bytealg.IndexByteString directly.
		return indexByte(s, byte(r))
	case r == utf8.RuneError:
		for i, r := range s {
			if r == utf8.RuneError {
				return i, utf8.RuneLen(r)
			}
		}
		return -1, 1
	case !utf8.ValidRune(r):
		return -1, 1
	default:
		// TODO: use a function for len(folds)
		if folds := foldMap(r); folds != nil {
			size := utf8.RuneLen(r)
			n := indexRuneCase(s, r)
			if n == 0 {
				return 0, size
			}
			if n > 0 {
				s = s[:n]
			}
			for i := 0; i < len(folds); i++ {
				rr := rune(folds[i])
				if rr == r {
					continue
				}
				if rr == 0 {
					break
				}
				o := indexRuneCase(s, rr)
				if o != -1 && (n == -1 || o < n) {
					n = o
					s = s[:n]
					size = utf8.RuneLen(rr)
				}
			}
			return n, size
		} else if u, l, ok := toUpperLower(r); ok {
			return indexRune2(s, l, u)
		}
		return indexRuneCase(s, r), utf8.RuneLen(r)
	}
}

func cutoverIndexRune(n int) int {
	// TODO: Merge this with cutover() which is identical.
	if runtime.GOARCH != "arm64" {
		return (n + 16) / 8
	}
	return 4 + n>>4
}

// indexRuneCase is a *case-sensitive* version of strings.IndexRune that is
// generally faster for Unicode characters since it searches for the rune's
// second byte, which is generally more unique, instead of the first byte,
// like strings.IndexRune.
func indexRuneCase(s string, r rune) int {
	// TODO: consider searching for the first byte if it is not one of:
	// 240, 243, or 244 (which are the first byte of ~78% of multi-byte
	// Unicode characters).
	//
	// TODO: remove check for invalid runes, if possible
	switch {
	case 0 <= r && r < utf8.RuneSelf:
		return strings.IndexByte(s, byte(r))
	case r == utf8.RuneError:
		for i, r := range s {
			if r == utf8.RuneError {
				return i
			}
		}
		return -1
	case !utf8.ValidRune(r):
		return -1
	default:
		var n int
		var c0, c1, c2, c3 byte
		// Inlined version of utf8.EncodeRune
		{
			const (
				tx       = 0b10000000
				t2       = 0b11000000
				t3       = 0b11100000
				t4       = 0b11110000
				maskx    = 0b00111111
				rune2Max = 1<<11 - 1
				rune3Max = 1<<16 - 1
			)
			switch i := uint32(r); {
			case i <= rune2Max:
				c0 = t2 | byte(r>>6)
				c1 = tx | byte(r)&maskx
				n = 2
			// NB: removed the invalid rune check since that is
			// performed above.
			case i <= rune3Max:
				c0 = t3 | byte(r>>12)
				c1 = tx | byte(r>>6)&maskx
				c2 = tx | byte(r)&maskx
				n = 3
			default:
				c0 = t4 | byte(r>>18)
				c1 = tx | byte(r>>12)&maskx
				c2 = tx | byte(r>>6)&maskx
				c3 = tx | byte(r)&maskx
				n = 4
			}
		}
		if n >= len(s) {
			// WARN: test if this is faster
			// my guess is that it actually makes things slower
			if n == len(s) && string(r) == s {
				return 0
			}
			return -1
		}
		// NOTE: searching for the last byte was not always faster (so maybe
		// not worth investigating in the future).
		//
		// Search for r using the second byte of its UTF-8 encoded form
		// since it is more unique than the first byte. This 4-5x faster
		// when all the text is Unicode.
		switch n {
		case 2:
			fails := 0
			i := 1
			t := len(s)
			for i < t {
				if s[i] != c1 {
					o := strings.IndexByte(s[i+1:t], c1)
					if o < 0 {
						return -1
					}
					i += o + 1
				}
				if s[i-1] == c0 {
					return i - 1
				}
				fails++
				i++
				if fails > cutoverIndexRune(i) {
					if j := strings.Index(s[i:], string(r)); j != -1 {
						return i + j
					}
					return -1
				}
			}
		case 3:
			fails := 0
			i := 1
			t := len(s) - 1
			for i < t {
				if s[i] != c1 {
					o := strings.IndexByte(s[i+1:t], c1)
					if o < 0 {
						return -1
					}
					i += o + 1
				}
				if s[i-1] == c0 && s[i+1] == c2 {
					return i - 1
				}
				fails++
				i++
				if fails > cutoverIndexRune(i) {
					if j := strings.Index(s[i:], string(r)); j != -1 {
						return i + j
					}
					return -1
				}
			}
		case 4:
			fails := 0
			i := 1
			t := len(s) - 2
			for i < t {
				if s[i] != c1 {
					o := strings.IndexByte(s[i+1:t], c1)
					if o < 0 {
						return -1
					}
					i += o + 1
				}
				if s[i-1] == c0 && s[i+1] == c2 && s[i+2] == c3 {
					return i - 1
				}
				fails++
				i++
				if fails > cutoverIndexRune(i) {
					if j := strings.Index(s[i:], string(r)); j != -1 {
						return i + j
					}
					return -1
				}
			}
		}
		return -1
	}
}

// indexRune2 returns the index of the first instance of the Unicode code point
// lower or upper. The search is *case-sensitive*.
func indexRune2(s string, lower, upper rune) (int, int) {
	// TODO:
	//   - do we need this now that we don't search for the first byte?
	//   - check if any of the rune bytes are equal and search by that
	//   - NB: this ^^^ was slower when I tried that
	//
	// Percentage of uppper/lower case runes that share bytes (at index):
	//
	//	0: 80.65%
	//	1: 46.58%
	//	2: 11.57%
	//	3: 3.83%
	//
	// Based on the above we could combine searches using the second byte.
	if lower|upper < utf8.RuneSelf {
		return indexByte(s, byte(lower&0x7F))
	}
	n := indexRuneCase(s, lower)
	sz := utf8.RuneLen(lower)
	if n != 0 && lower != upper {
		if 0 <= n && n < len(s) {
			s = s[:n] // limit the search space
		}
		if o := indexRuneCase(s, upper); n == -1 || (0 <= o && o < n) {
			n = o
			sz = utf8.RuneLen(upper)
		}
	}
	return n, sz
}

// lastIndexRune returns the last index of the first instance of the Unicode
// code point r, or -1 if rune is not present in s.
// If r is utf8.RuneError, it returns the last instance of any
// invalid UTF-8 byte sequence.
func lastIndexRune(s string, r rune) int {
	switch {
	case r == utf8.RuneError:
		for i := len(s); i > 0; {
			sr, size := utf8.DecodeLastRuneInString(s[:i])
			i -= size
			if sr == utf8.RuneError {
				return i
			}
		}
		return -1
	case !utf8.ValidRune(r):
		return -1
	default:
		if folds := foldMap(r); folds != nil {
			for i := len(s); i > 0; {
				var sr rune
				if sr = rune(s[i-1]); sr < utf8.RuneSelf {
					i--
				} else {
					var size int
					sr, size = utf8.DecodeLastRuneInString(s[:i])
					i -= size
				}
				for j := 0; j < len(folds) && folds[j] != 0; j++ {
					if sr == rune(folds[j]) {
						return i
					}
				}
			}
		} else {
			u, l, _ := toUpperLower(r)
			for i := len(s); i > 0; {
				var sr rune
				if sr = rune(s[i-1]); sr < utf8.RuneSelf {
					i--
				} else {
					var size int
					sr, size = utf8.DecodeLastRuneInString(s[:i])
					i -= size
				}
				if sr == u || sr == l {
					return i
				}
			}
		}
		return -1
	}
}

// primeRK is the prime base used in Rabin-Karp algorithm.
const primeRK = 16777619

// hashStrUnicode returns the hash and the appropriate multiplicative
// factor for use in Rabin-Karp algorithm, and the number of runes
// in sep.
func hashStrUnicode(sep string) (uint32, uint32, int) {
	hash := uint32(0)
	n := 0
	for _, r := range sep {
		if r < utf8.RuneSelf {
			r = rune(_lower[r])
		} else {
			r = caseFold(r)
		}
		hash = hash*primeRK + uint32(r)
		n++
	}
	var pow, sq uint32 = 1, primeRK
	for i := n; i > 0; i >>= 1 {
		if i&1 != 0 {
			pow *= sq
		}
		sq *= sq
	}
	return hash, pow, n
}

// hashStrRevUnicode returns the hash of the reverse of sep and the
// appropriate multiplicative factor for use in Rabin-Karp algorithm,
// and the number of runes in sep.
func hashStrRevUnicode(sep string) (uint32, uint32, int) {
	hash := uint32(0)
	n := 0
	for i := len(sep); i > 0; {
		var r rune
		var size int
		if sep[i-1] < utf8.RuneSelf {
			r, size = rune(_lower[sep[i-1]]), 1
		} else {
			r, size = utf8.DecodeLastRuneInString(sep[:i])
			r = caseFold(r)
		}
		hash = hash*primeRK + uint32(r)
		i -= size
		n++
	}
	var pow, sq uint32 = 1, primeRK
	for i := n; i > 0; i >>= 1 {
		if i&1 != 0 {
			pow *= sq
		}
		sq *= sq
	}
	return hash, pow, n
}

// indexRabinKarpRevUnicode uses the Rabin-Karp search algorithm to return the
// index of the last occurrence of substr in s, or -1 if not present.
func indexRabinKarpRevUnicode(s, substr string) int {
	// Reverse Rabin-Karp search
	hashss, pow, n := hashStrRevUnicode(substr)
	var h uint32
	i := len(s)
	for i > 0 {
		var r rune
		var size int
		if s[i-1] < utf8.RuneSelf {
			r, size = rune(_lower[s[i-1]]), 1
		} else {
			r, size = utf8.DecodeLastRuneInString(s[:i])
			r = caseFold(r)
		}
		h = h*primeRK + uint32(r)
		i -= size
		n--
		if n == 0 {
			break
		}
	}
	if n > 0 {
		return -1
	}
	if h == hashss && HasSuffix(s, substr) {
		return i // WARN
	}
	j := len(s)
	for i > 0 {
		var r0 rune
		var n0 int
		if s[i-1] < utf8.RuneSelf {
			r0, n0 = rune(_lower[s[i-1]]), 1
		} else {
			r0, n0 = utf8.DecodeLastRuneInString(s[:i])
			r0 = caseFold(r0)
		}
		var r1 rune
		var n1 int
		if s[j-1] < utf8.RuneSelf {
			r1, n1 = rune(_lower[s[j-1]]), 1
		} else {
			r1, n1 = utf8.DecodeLastRuneInString(s[:j])
			r1 = caseFold(r1)
		}
		h *= primeRK
		h += uint32(r0)
		h -= pow * uint32(r1)
		i -= n0
		j -= n1
		if h == hashss && HasSuffix(s[i:j], substr) {
			return i
		}
	}
	return -1
}

// indexRabinKarpUnicode uses the Rabin-Karp search algorithm to return the
// index of the first occurrence of substr in s, or -1 if not present.
func indexRabinKarpUnicode(s, substr string) int {
	// Rabin-Karp search
	hashss, pow, n := hashStrUnicode(substr)
	var h uint32
	sz := 0 // byte size of 'n' runes
	for i, r := range s {
		orig := r
		if r < utf8.RuneSelf {
			r = rune(_lower[r])
		} else {
			r = caseFold(r)
		}
		h = h*primeRK + uint32(r)
		n--
		if n == 0 {
			sz = i + utf8.RuneLen(orig)
			break
		}
	}
	if h == hashss && HasPrefix(s, substr) {
		return 0
	}
	i := 0 // start of rolling hash
	for j := sz; j < len(s); {
		h *= primeRK
		var s0, s1 rune
		var n0, n1 int
		if s[j] < utf8.RuneSelf {
			s0, n0 = rune(_lower[s[j]]), 1
		} else {
			s0, n0 = utf8.DecodeRuneInString(s[j:])
			s0 = caseFold(s0)
		}
		if s[i] < utf8.RuneSelf {
			s1, n1 = rune(_lower[s[i]]), 1
		} else {
			s1, n1 = utf8.DecodeRuneInString(s[i:])
			s1 = caseFold(s1)
		}
		h += uint32(s0)
		h -= pow * uint32(s1)
		j += n0
		i += n1
		if h == hashss && HasPrefix(s[i:j], substr) {
			return i
		}
	}
	return -1
}

// Count counts the number of non-overlapping instances of substr in s.
// If substr is an empty string, Count returns 1 + the number of Unicode
// code points in s.
func Count(s, substr string) int {
	// special case
	if len(substr) == 0 {
		return utf8.RuneCountInString(s) + 1
	}
	if len(substr) == 1 {
		c := substr[0]
		n := bytealg.CountString(s, c)
		switch c {
		case 'K', 'k':
			// TODO: move this shared logic to a function
			for {
				i := indexRuneCase(s, 'K')
				if i == -1 {
					return n
				}
				n++
				s = s[i+len("K"):]
			}
		case 'S', 's':
			for {
				i := indexRuneCase(s, 'ſ')
				if i == -1 {
					return n
				}
				n++
				s = s[i+len("ſ"):]
			}
		}
		return n
	}
	n := 0
	runeCount := utf8.RuneCountInString(substr)
	for {
		i := Index(s, substr)
		if i == -1 {
			return n
		}
		n++
		o := runeCount
		s = s[i:]
		// Trim substr prefix from s.
		for j, r := range s {
			o--
			if o == 0 {
				s = s[j+utf8.RuneLen(r):]
				break
			}
		}
	}
}

// Contains reports whether substr is within s.
func Contains(s, substr string) bool {
	return Index(s, substr) >= 0
}

// ContainsAny reports whether any Unicode code points in chars are within s.
func ContainsAny(s, chars string) bool {
	return IndexAny(s, chars) >= 0
}

// ContainsRune reports whether the Unicode code point r is within s.
func ContainsRune(s string, r rune) bool {
	return IndexRune(s, r) >= 0
}

// asciiSet is a 32-byte value, where each bit represents the presence of a
// given ASCII character in the set. The 128-bits of the lower 16 bytes,
// starting with the least-significant bit of the lowest word to the
// most-significant bit of the highest word, map to the full range of all
// 128 ASCII characters. The 128-bits of the upper 16 bytes will be zeroed,
// ensuring that any non-ASCII character will be reported as not in the set.
// This allocates a total of 32 bytes even though the upper half
// is unused to avoid bounds checks in asciiSet.contains.
type asciiSet [8]uint32

// makeASCIISet creates a set of ASCII characters and reports whether all
// characters in chars are ASCII.
func makeASCIISet(s, chars string) (as asciiSet, ok bool) {
	i := 0
	for ; i < len(chars); i++ {
		c := chars[i]
		if c >= utf8.RuneSelf {
			return as, false
		}
		as[c/32] |= 1 << (c % 32)
		if isAlpha(c) {
			c ^= ' ' // swap case
			as[c/32] |= 1 << (c % 32)
			// Can't use ASCII when non-ASCII chars fold to ASCII chars.
			switch c {
			case 'K', 'k', 'S', 's':
				// Checking if s contains only ASCII and using asciiSet is
				// faster than falling back to the Unicode aware search.
				// This holds true even on systems where ContainsNonASCII
				// does not use SIMD (tested on arm).
				if !ContainsNonASCII(s) {
					i++
					goto ascii // ASCII fast path that elides this check
				}
				return as, false
			}
		}
	}

ascii:
	// ASCII fast path for when we know s does not contain Unicode.
	for ; i < len(chars); i++ {
		c := chars[i]
		if c >= utf8.RuneSelf {
			return as, false
		}
		as[c/32] |= 1 << (c % 32)
		if isAlpha(c) {
			c ^= ' ' // swap case
			as[c/32] |= 1 << (c % 32)
		}
	}
	return as, true
}

// contains reports whether c is inside the set.
func (as *asciiSet) contains(c byte) bool {
	return (as[c/32] & (1 << (c % 32))) != 0
}

// IndexAny returns the index of the first instance of any Unicode code point
// from chars in s, or -1 if no Unicode code point from chars is present in s.
func IndexAny(s, chars string) int {
	if chars == "" {
		// Avoid scanning all of s.
		return -1
	}
	if len(chars) == 1 {
		// Avoid scanning all of s.
		r := rune(chars[0])
		if r >= utf8.RuneSelf {
			r = utf8.RuneError
		}
		return IndexRune(s, r)
	}
	if len(s) > 8 {
		if as, isASCII := makeASCIISet(s, chars); isASCII {
			// TODO: should we convert Kelvin and Small Long S to ASCII here?
			for i := 0; i < len(s); i++ {
				if as.contains(s[i]) {
					return i
				}
			}
			return -1
		}
	}
	if len(s) > len(chars)*2 {
		// Avoid the overhead of repeatedly calling IndexRune
		// if s significantly longer than chars (IndexRune is
		// also quite fast for long strings).
		//
		// This cutover was empirically found via internal/benchtest.
		n := -1
		for _, r := range chars {
			i := IndexRune(s, r)
			if i != -1 && (n == -1 || i < n) {
				n = i
				if n == 0 {
					break
				}
				s = s[:n]
			}
		}
		return n
	}
	for i, c := range s {
		if IndexRune(chars, c) >= 0 {
			return i
		}
	}
	return -1
}

// LastIndexAny returns the index of the last instance of any Unicode code
// point from chars in s, or -1 if no Unicode code point from chars is
// present in s.
func LastIndexAny(s, chars string) int {
	if chars == "" {
		return -1
	}
	if len(s) == 1 {
		rc := rune(s[0])
		if rc >= utf8.RuneSelf {
			rc = utf8.RuneError
		}
		if IndexRune(chars, rc) >= 0 {
			return 0
		}
		return -1
	}
	if len(s) > 8 {
		if as, isASCII := makeASCIISet(s, chars); isASCII {
			for i := len(s) - 1; i >= 0; i-- {
				if as.contains(s[i]) {
					return i
				}
			}
			return -1
		}
	}
	if len(chars) == 1 {
		if c := chars[0]; c < utf8.RuneSelf {
			return LastIndexByte(s, c)
		}
		for i := len(s); i > 0; {
			r, size := utf8.DecodeLastRuneInString(s[:i])
			i -= size
			if r == utf8.RuneError {
				return i
			}
		}
		return -1
	}
	for i := len(s); i > 0; {
		r, size := utf8.DecodeLastRuneInString(s[:i])
		i -= size
		if IndexRune(chars, r) >= 0 {
			return i
		}
	}
	return -1
}

// Cut slices s around the first instance of sep,
// returning the text before and after sep.
// The found result reports whether sep appears in s.
// If sep does not appear in s, cut returns s, "", false.
func Cut(s, sep string) (before, after string, found bool) {
	// TODO: both Cut and Count would benefit from Index returning the
	// number of bytes consumed from sep - consider adding this.
	if i := Index(s, sep); i >= 0 {
		after = s[i:]
		// trim sep from s
		for range sep {
			if after[0] < utf8.RuneSelf {
				after = after[1:]
			} else {
				_, n := utf8.DecodeRuneInString(after)
				after = after[n:]
			}
		}
		return s[:i], after, true
	}
	return s, "", false
}

// CutPrefix returns s without the provided leading prefix string
// and reports whether it found the prefix.
// If s doesn't start with prefix, CutPrefix returns s, false.
// If prefix is the empty string, CutPrefix returns s, true.
func CutPrefix(s, prefix string) (after string, found bool) {
	if prefix == "" {
		return s, true
	}
	if ss := TrimPrefix(s, prefix); len(ss) != len(s) {
		return ss, true
	}
	return s, false
}

// CutSuffix returns s without theI provided ending suffix string
// and reports whether it found the suffix.
// If s doesn't end with suffix, CutSuffix returns s, false.
// If suffix is the empty string, CutSuffix returns s, true.
func CutSuffix(s, suffix string) (before string, found bool) {
	if suffix == "" {
		return s, true
	}
	if match, _, i := hasSuffixUnicode(s, suffix); match {
		return s[:i], true
	}
	return s, false
}

// IndexNonASCII returns the index of first non-ASCII rune in s, or -1 if s
// consists only of ASCII characters.
//
// On arm64 and amd64, IndexNonASCII is an order of magnitude faster than
// using a for loop and checking each byte of s and should be close to that
// of strings.IndexByte (54GBi/s on arm64).
//
// IndexNonASCII is up to 17 times faster on arm64 and 12 times faster on
// amd64 compared to using a for loop and checking each byte of s.
func IndexNonASCII(s string) int {
	return bytealg.IndexNonASCII(s)
}

// ContainsNonASCII returns true if s contains any non-ASCII characters.
func ContainsNonASCII(s string) bool {
	return bytealg.IndexNonASCII(s) >= 0
}
