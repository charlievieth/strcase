// Package strings is a case-insensitive implementation of the strings package.
package strcase

//go:generate go run gen.go

import (
	"io"
	"log"
	"os"
	"runtime"
	"strings"
	"unicode"
	"unicode/utf8"
	"unsafe"
)

// TODO:
//  1. Compare using unicode.CaseRanges. That is get the case range for the
//     runes being searched for and see if the runes being searched are in it.
//  2. Consider using folding and checking if the fold is possible (a < b or
//     whatever).

// WARN: DEV ONLY
const debug = false

// const debug = true

func init() {
	log.SetFlags(log.Lshortfile)
	if debug {
		log.SetOutput(os.Stderr)
	} else {
		log.SetOutput(io.Discard)
	}
}

const maxBruteForce = 16 // substring length
const maxLen = 32        // subject length

func clamp(n int) int {
	if n < 0 {
		return -1
	}
	if n > 0 {
		return 1
	}
	return 0
}

////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////
//
//
// TODO: look up case table for 1 of the runes then compare the other to it !!!
//
////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////

func Compare(s, t string) int {
	i := 0
	for ; i < len(s) && i < len(t); i++ {
		sr := s[i]
		tr := t[i]
		if sr|tr >= utf8.RuneSelf {
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
			tr, t = rune(t[0]), t[1:]
			if 'A' <= tr && tr <= 'Z' {
				tr += 'a' - 'A'
			}
		} else {
			r, size := utf8.DecodeRuneInString(t)
			tr, t = r, t[size:]
			tr = unicode.ToLower(tr)
		}

		// Easy case.
		if sr == tr {
			continue
		}
		if sr < utf8.RuneSelf {
			if 'A' <= sr && sr <= 'Z' {
				sr += 'a' - 'A'
			}
		} else {
			sr = unicode.To(unicode.LowerCase, sr)
		}
		if sr != tr {
			return clamp(int(sr) - int(tr))
		}
	}
	if len(t) == 0 {
		return 0
	}
	return -1
}

func isAlpha(c byte) bool {
	return 'A' <= c && c <= 'Z' || 'a' <= c && c <= 'z'
}

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

// WARN: use IndexNonASCII()
func isASCII(s string) bool {
	// i := 0
	// for i := 0; i < len(s) && _asciiSafe[s[i]]; i++ {
	// }
	// return i == len(s)

	for i := 0; i < len(s); i++ {
		if s[i] >= utf8.RuneSelf {
			return false
		}
		// if !_asciiSafe[s[i]] {
		// 	return false
		// }
		// if c >= utf8.RuneSelf || c == 'k' || c == 'K' || c == 'i' || c == 'I' {
		// 	return false
		// }
	}
	return true
}

func hasUnicode(s string) bool { return !isASCII(s) }

// HasPrefix tests whether the string s begins with prefix ignoring case.
func HasPrefix(s, prefix string) bool {
	ok, _ := hasPrefixUnicode(s, prefix)
	return ok
}

func validByteDelta(s, sep string) bool {
	if len(sep) > len(s)*3 {
		return false
	}
	return false
}

// TODO: do we need this ???
//
// func maxByteDelta(s, sep string) int {
// 	return -1
// }

// hasPrefixUnicode returns if string s begins with prefix (ignoring case) and
// if s was exhausted before a match was found.
func hasPrefixUnicode(s, prefix string) (bool, bool) {
	// The max difference in encoded lengths between cases is 2 bytes for
	// [kK] (Latin - 1 byte) and 'K' (Kelvin - 3 bytes).
	n := len(s)
	if n*3 < len(prefix) || (n*2 < len(prefix) && !strings.ContainsRune(prefix, 'K')) {
		return false, true
	}

	// TODO: see if this is faster in some cases
	//
	// if strings.HasPrefix(s, prefix) {
	// 	return true, len(s) == len(prefix)
	// }

	// ASCII fast path
	i := 0
	for ; i < len(s) && i < len(prefix); i++ {
		sr := s[i]
		tr := prefix[i]
		if sr|tr >= utf8.RuneSelf {
			goto hasUnicode
		}

		// Easy case.
		if tr == sr {
			continue
		}

		// Make sr < tr to simplify what follows.
		if tr < sr {
			tr, sr = sr, tr
		}
		// ASCII only, sr/tr must be upper/lower case
		if 'A' <= sr && sr <= 'Z' && tr == sr+'a'-'A' {
			continue
		}
		return false, i == len(s)-1
	}
	// Check if we've exhausted the prefix
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
			sr, s = rune(s[0]), s[1:]
		} else {
			r, size := utf8.DecodeRuneInString(s)
			sr, s = r, s[size:]
		}

		// If they match, keep going; if not, return false.

		// Easy case.
		if tr == sr {
			continue
		}

		// Make sr < tr to simplify what follows.
		if tr < sr {
			tr, sr = sr, tr
		}
		// Fast check for ASCII.
		if tr < utf8.RuneSelf {
			// ASCII only, sr/tr must be upper/lower case
			if 'A' <= sr && sr <= 'Z' && tr == sr+'a'-'A' {
				continue
			}
			return false, len(s) == 0
		}

		// General case. SimpleFold(x) returns the next equivalent rune > x
		// or wraps around to smaller values.
		r := unicode.SimpleFold(sr)
		for r != sr && r < tr {
			r = unicode.SimpleFold(r)
		}
		if r == tr {
			continue
		}
		return false, len(s) == 0
	}

	return true, len(s) == 0 // Prefix exhausted
}

// toUpperLower combines unicode.ToUpper and unicode.ToLower in one function.
func toUpperLower(r rune) (upper, lower rune, foundMapping bool) {
	if r <= unicode.MaxASCII {
		if 'A' <= r && r <= 'Z' {
			return r, r + ('a' - 'A'), true
		}
		if 'a' <= r && r <= 'z' {
			return r - ('a' - 'A'), r, true
		}
		return r, r, false
	}

	// binary search over ranges
	caseRange := unicode.CaseRanges
	lo := 0
	hi := len(caseRange)
	for lo < hi {
		m := lo + (hi-lo)/2
		cr := caseRange[m]
		if rune(cr.Lo) <= r && r <= rune(cr.Hi) {
			if delta := cr.Delta[unicode.UpperCase]; delta > unicode.MaxRune {
				// In an Upper-Lower sequence, which always starts with
				// an UpperCase letter, the real deltas always look like:
				//	{0, 1, 0}    UpperCase (Lower is next)
				//	{-1, 0, -1}  LowerCase (Upper, Title are previous)
				// The characters at even offsets from the beginning of the
				// sequence are upper case; the ones at odd offsets are lower.
				// The correct mapping can be done by clearing or setting the low
				// bit in the sequence offset.
				// The constants UpperCase and TitleCase are even while LowerCase
				// is odd so we take the low bit from _case.
				upper = rune(cr.Lo) + ((r-rune(cr.Lo))&^1 | rune(unicode.UpperCase&1))
			} else {
				upper = r + delta
			}
			if delta := cr.Delta[unicode.LowerCase]; delta > unicode.MaxRune {
				lower = rune(cr.Lo) + ((r-rune(cr.Lo))&^1 | rune(unicode.LowerCase&1))
			} else {
				lower = r + delta
			}
			return upper, lower, true
		}
		if r < rune(cr.Lo) {
			hi = m
		} else {
			lo = m + 1
		}
	}
	return r, r, false
}

func bruteForceIndexUnicode(s, substr string) int {
	// NB: substr must contain at least 2 characters.

	u0, sz0 := utf8.DecodeRuneInString(substr)
	u1, sz1 := utf8.DecodeRuneInString(substr[sz0:])
	folds0, hasFolds0 := _FoldMapExcludingUpperLower[u0]
	folds1, hasFolds1 := _FoldMapExcludingUpperLower[u1]
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

	switch {
	case !hasFolds0 && u0 == l0 && !hasFolds1 && u1 == l1:
		i := 0
		if u0 != u1 {
			ii := strings.IndexRune(s, u1)
			if ii < 0 {
				return -1
			}
			if ii-sz0-1 > 0 {
				i = ii - sz0 - 1
			}
		}
		for i < len(s) {
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
			if i+n0 == len(s) {
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
		i := 0
		if u0 != u1 && l0 != l1 {
			ii, _ := indexRune2(s, l1, u1)
			if ii == -1 {
				return -1
			}
			if ii-sz0-1 > 0 {
				i = ii - sz0 - 1
			}
		}
		for i < len(s) {
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
			if i+n0 == len(s) {
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
		for i := 0; i < len(s); {
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
			if i+n0 == len(s) {
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

func cutover(n int) int {
	// FIXME: our cutoff is probably different since our algo is not optimized

	// WARN: this is much slower on arm64 - test on amd64
	if runtime.GOARCH == "amd64" {
		// 1 error per 8 characters, plus a few slop to start.
		return (n + 16) / 8

	}
	// arm64
	// 1 error per 16 characters, plus a few slop to start.

	return 4 + n>>4
	// return (n + 16) / 8
}

func shortIndexUnicode(s, substr string) int {
	u0, sz0 := utf8.DecodeRuneInString(substr)
	u1, sz1 := utf8.DecodeRuneInString(substr[sz0:])
	folds0, hasFolds0 := _FoldMapExcludingUpperLower[u0]
	folds1, hasFolds1 := _FoldMapExcludingUpperLower[u1]
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
	i := 0
	// t := len(s) - (len(substr) / 3) + 1
	// for i < t {
	for i < len(s) {
		var r0 rune
		var n0 int
		if s[i] < utf8.RuneSelf {
			r0, n0 = rune(s[i]), 1
		} else {
			r0, n0 = utf8.DecodeRuneInString(s[i:])
		}

		// TODO: test with '\0' (NULL) because empty folds are 0

		if r0 != u0 && r0 != l0 && (!hasFolds0 || (r0 != folds0[0] && r0 != folds0[1])) {
			var o, sz int
			if !hasFolds0 {
				o, sz = indexRune2(s[i+n0:], l0, u0)
			} else {
				o, sz = indexRune(s[i+n0:], l0)
			}
			if o < 0 {
				return -1
			}
			i += o + n0
			n0 = sz // The rune we matched on might not be the same size as c0
		}

		// FIXME: take len(substr) into accout
		if i+n0 >= len(s) {
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

		// FIXME: this needs to be tuned since the brute force
		// performance is very different than the stdlibs.
		if fails > cutover(i) {
			r := bruteForceIndexUnicode(s[i:], substr)
			if r >= 0 {
				return r + i
			}
			return -1
		}
	}
	return -1
}

func indexUnicode(s, substr string) int {
	u0, sz0 := utf8.DecodeRuneInString(substr)
	u1, sz1 := utf8.DecodeRuneInString(substr[sz0:])
	folds0, hasFolds0 := _FoldMapExcludingUpperLower[u0]
	folds1, hasFolds1 := _FoldMapExcludingUpperLower[u1]
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
	i := 0
	// WARN WARN WARN
	// t := len(s) - (len(substr) / 2) + 1
	t := len(s) - (len(substr) / 3) + 1
	for i < len(s) {
		var r0 rune
		var n0 int
		if s[i] < utf8.RuneSelf {
			r0, n0 = rune(s[i]), 1
		} else {
			r0, n0 = utf8.DecodeRuneInString(s[i:])
		}

		// TODO: test with '\0' (NULL) because empty folds are 0

		if r0 != u0 && r0 != l0 && (!hasFolds0 || (r0 != folds0[0] && r0 != folds0[1])) {
			var o, sz int
			if !hasFolds0 {
				o, sz = indexRune2(s[i+n0:], l0, u0)
			} else {
				o, sz = indexRune(s[i+n0:], l0)
			}
			if o < 0 {
				return -1
			}
			i += o + n0
			n0 = sz // The rune we matched on might not be the same size as c0
		}

		// WARN WARN WARN WARN WARN
		// if i+n0 >= t {
		if i+n0 >= len(s) {
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

		// WARN: this needs to be tuned since the brute force performance
		// is very different than the stdlibs.
		if fails >= 4+i>>4 && i < t {
			j := -2
			// Attempt to use Rabin-Karp
			if !hasFolds0 && !hasFolds1 {
				j = indexRabinKarpUnicode(s[i:], substr)
			}
			switch j {
			case -2:
				// Fallback to brute-force if we have to deal with folds
				o := bruteForceIndexUnicode(s[i:], substr)
				if o >= 0 {
					return o + i
				}
				return -1
			case -1:
				return -1
			default:
				return i + j
			}
		}
	}
	return -1
}

func Index(s, substr string) int {
	n := len(substr)
	r, size := utf8.DecodeRuneInString(substr)
	switch {
	case n == 0:
		return 0
	case n == 1:
		return IndexByte(s, substr[0])
	case n == size:
		return IndexRune(s, r)
	case n >= len(s):
		if n > len(s)*3 {
			return -1
		}
		if n > len(s)*2 && !strings.Contains(substr, "K") {
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
		if o := bruteForceIndexUnicode(s[i:], substr); o != -1 {
			return o + i
		}
		return -1

	case n <= maxLen: // WARN: 32 is for arm64 (see: bytealg.MaxLen)
		// WARN: profile bruteForceIndexUnicode
		if len(s) <= maxBruteForce {
			return bruteForceIndexUnicode(s, substr)
		}
		return shortIndexUnicode(s, substr)
	}
	return indexUnicode(s, substr)
}

// WARN: this breaks if there is a non-ASCII form of byte c
func IndexByte(s string, c byte) int {
	i, _ := indexByte(s, c)
	return i
}

func indexByte(s string, c byte) (int, int) {
	n := strings.IndexByte(s, c)
	if n == 0 || !isAlpha(c) {
		return n, 1
	}

	// TODO: calculate the optimal cutoff
	if n != -1 && len(s)-n >= 128 {
		s = s[:n] // limit search space
	}

	c ^= ' ' // swap case
	if o := strings.IndexByte(s, c); n == -1 || (o != -1 && o < n) {
		n = o
	}

	// Special case for Unicode characters that map to ASCII.
	var r rune
	var sz int
	switch c {
	// TODO: remove this block
	// case 'I', 'i':
	// 	r = 'İ'
	// 	sz = 2
	case 'K', 'k':
		r = 'K'
		sz = 3
	case 'S', 's':
		r = 'ſ'
		sz = 2
	default:
		return n, 1
	}

	if 0 <= n && n < len(s) {
		s = s[:n]
	}
	// strings.IndexRune uses strings.Index and we know the rune is valid.
	if o := strings.Index(s, string(r)); n == -1 || (o != -1 && o < n) {
		return o, sz
	}
	return n, 1
}

// IndexRune returns the index of the first instance of the Unicode code point
// r, or -1 if rune is not present in s.
// If r is utf8.RuneError, it returns the first instance of any
// invalid UTF-8 byte sequence.
func IndexRune(s string, r rune) int {
	i, _ := indexRune(s, r)
	return i
}

// indexRune returns the index of the first instance of the Unicode code point
// r (ignoring case) and the size of the rune that matched.
// If r is utf8.RuneError, it returns the first instance of any
// invalid UTF-8 byte sequence and 1.
func indexRune(s string, r rune) (int, int) {
	switch {
	case 0 <= r && r < utf8.RuneSelf:
		return indexByte(s, byte(r))
	case r == utf8.RuneError:
		for i, r := range s {
			if r == utf8.RuneError {
				return i, 1
			}
		}
		return -1, 1
	case !utf8.ValidRune(r):
		return -1, 1
	default:
		size := utf8.RuneLen(r)
		n := strings.Index(s, string(r))
		if n == 0 {
			return 0, size
		}
		if n > 0 {
			s = s[:n]
		}
		if folds, hasFolds := _FoldMap[r]; hasFolds {
			for i := 0; i < len(folds); i++ {
				rr := folds[i]
				if rr == r {
					continue
				}
				o := strings.Index(s, string(rr))
				if o != -1 && (n == -1 || o < n) {
					n = o
					s = s[:n]
					size = utf8.RuneLen(rr)
				}
			}
		} else {
			u, l, ok := toUpperLower(r)
			if ok {
				for _, rr := range []rune{u, l} {
					if rr == r {
						continue
					}
					o := strings.Index(s, string(rr))
					if o != -1 && (n == -1 || o < n) {
						n = o
						s = s[:n]
						size = utf8.RuneLen(rr)
					}
				}
			}
		}
		return n, size
	}
}

func indexRune2(s string, lower, upper rune) (n, sz int) {
	n = strings.IndexRune(s, lower)
	sz = utf8.RuneLen(lower)
	if n != 0 && lower != upper {
		if 0 <= n && n < len(s) {
			s = s[:n] // limit the search space
		}
		if o := strings.IndexRune(s, upper); n == -1 || (o != -1 && o < n) {
			n = o
			sz = utf8.RuneLen(upper)
		}
	}
	return n, sz
}

// primeRK is the prime base used in Rabin-Karp algorithm.
const primeRK = 16777619

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

// TODO: do we need separate funcs for this???
func hashStrUnicode(sep string) (uint32, uint32, int) {
	hash := uint32(0)
	n := 0
	for _, r := range sep {
		if r < utf8.RuneSelf {
			if 'A' <= r && r <= 'Z' {
				r += 'a' - 'A'
			}
		} else {
			if _, ok := _MustFold[r]; ok {
				return 0, 0, -2
			}
			r = unicode.To(unicode.LowerCase, r)
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

func indexRabinKarpUnicode(s, substr string) int {
	// Rabin-Karp search
	hashss, pow, n := hashStrUnicode(substr)
	if n == -2 {
		return -2
	}
	var h uint32
	sz := 0 // byte size of 'o' runes
	for i, r := range s {
		if _, ok := _MustFold[r]; ok {
			return -2
		}
		h = h*primeRK + uint32(unicode.ToLower(r))
		n--
		if n == 0 {
			sz = i + utf8.RuneLen(r)
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
			if _, ok := _MustFold[s0]; ok {
				return -2
			}
			s0 = unicode.To(unicode.LowerCase, s0)
		}
		if s[i] < utf8.RuneSelf {
			s1, n1 = rune(_lower[s[i]]), 1
		} else {
			s1, n1 = utf8.DecodeRuneInString(s[i:])
			if _, ok := _MustFold[s1]; ok {
				return -2
			}
			s1 = unicode.To(unicode.LowerCase, s1)
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

// WARN: rename
// TODO: the returned index is inaccurate
// TODO: use or remove
func IndexNonASCII(s string) int {
	const wordSize = int(unsafe.Sizeof(uintptr(0)))

	i := 0
	n := len(s) % wordSize
	for ; i < n && i < len(s); i++ {
		if s[i] >= utf8.RuneSelf {
			return i
		}
	}
	if i == len(s) {
		return -1
	}

	s = s[i:]
	b := *(*[]byte)(unsafe.Pointer(&s))
	sliceHeader := struct {
		p   unsafe.Pointer
		len int
		cap int
	}{unsafe.Pointer(&b[0]), len(b) / wordSize, len(b) / wordSize}

	if wordSize == 8 {
		const mask64 uint64 = 0x8080808080808080
		us := *(*[]uint64)(unsafe.Pointer(&sliceHeader))
		for j, u := range us {
			if m := u & mask64; m != 0 {
				return i + j*wordSize
			}
		}

	} else /* wordSize == 4 */ {
		const mask32 uint32 = 0x80808080
		us := *(*[]uint32)(unsafe.Pointer(&sliceHeader))
		for j, u := range us {
			if m := u & mask32; m != 0 {
				return i + j*wordSize
			}
		}
	}

	runtime.KeepAlive(&b[0])
	return -1
}
