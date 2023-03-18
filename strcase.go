// Package strings is a case-insensitive implementation of the strings package.
// Except where noted, simple Unicode case-folding is used to determine equality.
package strcase

// TODO: make sure package doc is accurate.

// BUG(cvieth): There is no mechanism for full case folding, that is, for
// characters that involve multiple runes in the input or output
// (see: https://pkg.go.dev/unicode#pkg-note-BUG).

//go:generate go run gen.go

import (
	"math/bits"
	"runtime"
	"strings"
	"unicode"
	"unicode/utf8"
	"unsafe"

	"github.com/charlievieth/strcase/internal/bytealg"
)

// TODO: tune these
const maxBruteForce = 16 // substring length
const maxLen = 32        // subject length

// A foldPair stores Unicode case folding pairs
type foldPair struct {
	From uint32
	To   uint32
}

// caseFold returns the Unicode simple case-fold for r, if one exists, or r
// unmodified, if one does not exist.
func caseFold(r rune) rune {
	u := uint32(r)
	h := (u * _CaseFoldsSeed) >> _CaseFoldsShift
	p := _CaseFolds[h]
	if p.From == u {
		r = rune(p.To)
	}
	return r
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

// TODO: add a version of strings.EqualFold() since this is faster
// also mention that in the README.
//
// TODO: move next to hasPrefixUnicode
//
// Compare returns an integer comparing two strings lexicographically
// ignoring case.
// The result will be 0 if a == b, -1 if a < b, and +1 if a > b.
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

func isAlpha(c byte) bool {
	return 'A' <= c && c <= 'Z' || 'a' <= c && c <= 'z'
}

// TODO: change to 128 bytes and use a mask
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
	// return strings.Contains(s, "\u212A")
	return indexRuneCase(s, '\u212A') != -1
}

// HasPrefix tests whether the string s begins with prefix ignoring case.
func HasPrefix(s, prefix string) bool {
	ok, _ := hasPrefixUnicode(s, prefix)
	return ok
}

// TODO: return an enum instead of two bools
//
// hasPrefixUnicode returns if string s begins with prefix (ignoring case) and
// if s was exhausted before a match was found.
func hasPrefixUnicode(s, prefix string) (bool, bool) {
	// The max difference in encoded lengths between cases is 2 bytes for
	// [kK] (Latin - 1 byte) and 'K' (Kelvin - 3 bytes).
	n := len(s)
	if n*3 < len(prefix) || (n*2 < len(prefix) && !containsKelvin(prefix)) {
		return false, true
	}

	// TODO: This is significantly faster when the strings match,
	// but needs more benchmarking to see how it impacts Index
	// performance.
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
			sr, s = rune(s[0]), s[1:]
			if 'A' <= sr && sr <= 'Z' {
				sr += 'a' - 'A'
			}
		} else {
			r, size := utf8.DecodeRuneInString(s)
			sr, s = caseFold(r), s[size:]
		}
		if tr == sr || caseFold(tr) == sr {
			continue
		}
		return false, len(s) == 0
	}
	return true, len(s) == 0 // s exhausted
}

// FIXME: comment
func HasSuffix(s, suffix string) bool {
	ok, _, _ := hasSuffixUnicode(s, suffix)
	return ok
}

// FIXME: comment
//
// Returns: match, exhausted s, byte index of the match in s
func hasSuffixUnicode(s, suffix string) (bool, bool, int) {
	// The max difference in encoded lengths between cases is 2 bytes for
	// [kK] (Latin - 1 byte) and 'K' (Kelvin - 3 bytes).
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
		return false, i == 0, 0
	}
	return j == -1, i <= 0, i + 1

hasUnicode:
	s = s[:i+1]
	t = t[:j+1]
	for len(s) != 0 && len(t) != 0 {
		var sr, tr rune
		if n := len(s) - 1; s[n] < utf8.RuneSelf {
			sr, s = rune(s[n]), s[:n]
			if 'A' <= sr && sr <= 'Z' {
				sr += 'a' - 'A'
			}
		} else {
			r, size := utf8.DecodeLastRuneInString(s)
			sr, s = r, s[:len(s)-size]
		}
		if n := len(t) - 1; t[n] < utf8.RuneSelf {
			tr, t = rune(t[n]), t[:n]
			if 'A' <= tr && tr <= 'Z' {
				tr += 'a' - 'A'
			}
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
		// Fast check for the first rune.
		// WARN
		// i := strings.Index(s, string(u0))
		i := indexRuneCase(s, u0)
		if i < 0 {
			return -1
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
		// if l0|u0 < utf8.RuneSelf {
		// 	i = IndexByte(s, byte(l0))
		// 	if i == -1 {
		// 		return -1
		// 	}
		// }
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

func bruteForceLastIndexUnicode(s, substr string) int {
	// NB: substr must contain at least 2 characters.

	u0, sz0 := utf8.DecodeLastRuneInString(substr)
	u1, sz1 := utf8.DecodeLastRuneInString(substr[:len(substr)-sz0])
	folds0, hasFolds0 := _FoldMapExcludingUpperLower[u0]
	folds1, hasFolds1 := _FoldMapExcludingUpperLower[u1]
	needle := substr[:len(substr)-sz0-sz1]

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
		for i := len(s); i > 0; {
			var n0 int
			var r0 rune
			if r0 = rune(s[i-1]); r0 < utf8.RuneSelf {
				n0 = 1
			} else {
				r0, n0 = utf8.DecodeLastRuneInString(s[:i])
			}
			if r0 != u0 {
				i -= n0
				continue
			}
			if i-n0 == 0 {
				break
			}

			var n1 int
			var r1 rune
			if r1 = rune(s[i-n0-1]); r1 < utf8.RuneSelf {
				n1 = 1
			} else {
				r1, n1 = utf8.DecodeLastRuneInString(s[:i-n0])
			}
			if r1 != u1 {
				i -= n0
				if r1 != u0 {
					i -= n1 // Skip 2 runes when possible
				}
				continue
			}
			if match, _, j := hasSuffixUnicode(s[:i-n0-n1], needle); match {
				return j
			}
			i -= n0
			if r1 != u0 {
				i -= n1 // Skip 2 runes when possible
			}
		}
		return -1

	case !hasFolds0 && !hasFolds1:
		for i := len(s); i > 0; {
			var n0 int
			var r0 rune
			if r0 = rune(s[i-1]); r0 < utf8.RuneSelf {
				n0 = 1
			} else {
				r0, n0 = utf8.DecodeLastRuneInString(s[:i])
			}
			if r0 != u0 && r0 != l0 {
				i -= n0
				continue
			}
			if i-n0 == 0 {
				break
			}

			var n1 int
			var r1 rune
			if r1 = rune(s[i-n0-1]); r1 < utf8.RuneSelf {
				n1 = 1
			} else {
				r1, n1 = utf8.DecodeLastRuneInString(s[:i-n0])
			}
			if r1 != u1 && r1 != l1 {
				i -= n0
				if r1 != u0 && r1 != l0 {
					i -= n1 // Skip 2 runes when possible
				}
				continue
			}
			if match, _, j := hasSuffixUnicode(s[:i-n0-n1], needle); match {
				return j
			}
			i -= n0
			if r1 != u0 && r1 != l0 {
				i -= n1 // Skip 2 runes when possible
			}
		}
		return -1

	default:
		// TODO: see if there is a better cutoff to use
		for i := len(s); i > 0; {
			var n0 int
			var r0 rune
			if r0 = rune(s[i-1]); r0 < utf8.RuneSelf {
				n0 = 1
			} else {
				r0, n0 = utf8.DecodeLastRuneInString(s[:i])
			}
			if r0 != u0 && r0 != l0 {
				if !hasFolds0 || (r0 != folds0[0] && r0 != folds0[1]) {
					i -= n0
					continue
				}
			}
			if i-n0 == 0 {
				break
			}

			var n1 int
			var r1 rune
			if r1 = rune(s[i-n0-1]); r1 < utf8.RuneSelf {
				n1 = 1
			} else {
				r1, n1 = utf8.DecodeLastRuneInString(s[:i-n0])
			}
			if r1 != u1 && r1 != l1 {
				if !hasFolds1 || (r1 != folds1[0] && r1 != folds1[1]) {
					i -= n0
					if !hasFolds0 && r1 != u0 && r1 != l0 {
						i -= n1 // Skip 2 runes when possible
					}
					continue
				}
			}
			if match, _, j := hasSuffixUnicode(s[:i-n0-n1], needle); match {
				return j
			}
			i -= n0
			if !hasFolds0 && r1 != u0 && r1 != l0 {
				i -= n1 // Skip 2 runes when possible
			}
		}
		return -1
	}
}

func cutover(n int) int {
	// FIXME: our cutoff is probably different since our algo is not optimized

	// WARN: this is much slower on arm64 - test on amd64
	if runtime.GOARCH != "arm64" {
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

	// WARN: resolve and benchmark this
	// TOOD: see if we can stop earlier: `t := len(s) - (len(substr) / 3) + 1`
	//
	// t := len(s) - (utf8.RuneCountInString(substr) + 1)
	t := len(s) - 1
	// t := len(s) - (len(substr) / 3) + 1

	for i < t {
		var r0 rune
		var n0 int
		if s[i] < utf8.RuneSelf {
			r0, n0 = rune(s[i]), 1
		} else {
			r0, n0 = utf8.DecodeRuneInString(s[i:])
		}

		// TODO: See if we can use our own version of strings.Index(Rune)?
		// that is faster. Could pre-compute the Rabin-Karp table, only use
		// strings.IndexByte, etc...

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
		// TODO: see if we can stop at `t`
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
	t := len(s) - (len(substr) / 3) + 1
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

		// WARN: this needs to be tuned since the brute force performance
		// is very different than the stdlibs.
		if fails >= 4+i>>4 && i < t {
			j := indexRabinKarpUnicode(s[i:], substr)
			if j < 0 {
				return -1
			}
			return i + j
		}
	}
	return -1
}

func nonLetterASCII(s string) bool {
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= utf8.RuneSelf || isAlpha(c) {
			return false
		}
	}
	return true
}

// Index returns the index of the first instance of substr in s, or -1 if
// substr is not present in s.
func Index(s, substr string) int {
	// TODO: check if short sub-strings contain only non-Alpha ASCII
	// chars
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
		if n > len(s)*2 && !containsKelvin(substr) {
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
		// WARN: fast path if the sub-string is all non-alpha ASCII chars
		// TODO: tune the cutoff here
		if nonLetterASCII(substr) {
			return strings.Index(s, substr)
		}
		if len(s) <= maxBruteForce {
			return bruteForceIndexUnicode(s, substr)
		}
		return shortIndexUnicode(s, substr)
	}
	return indexUnicode(s, substr)
}

// LastIndex returns the index of the last instance of substr in s, or -1 if
// substr is not present in s.
func LastIndex(s, substr string) int {
	n := len(substr)
	u, size := utf8.DecodeRuneInString(substr)
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
		// TODO: calculate the cutoff for brute-force vs. Rabin-Karp
		return bruteForceLastIndexUnicode(s, substr)
	}
	return indexRabinKarpRevUnicode(s, substr)
}

// FIXME: document
func IndexByte(s string, c byte) int {
	i, _ := indexByte(s, c)
	return i
}

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
		r = 'K'
		sz = 3
	case 'S', 's':
		r = 'ſ'
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
		// TODO:
		// 	* benchmark this on amd64
		// 	* see if checking for 'r' with indexRune is faster than
		// 	  bailing to the slow UTF8 loop
		//
		c |= ' ' // convert to lower case
		for i := len(s) - 1; i >= 0; i-- {
			if s[i]|' ' == c {
				return i
			}
		}
		return -1
	}

	// Handle ASCII characters with Unicode mappings
	c0 := c
	c1 := c ^ ' ' // swap case
	for i := len(s); i > 0; {
		if s[i-1] < utf8.RuneSelf {
			sr := s[i-1]
			i--
			if sr == c0 || sr == c1 {
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
	i, _ := indexRune(s, r)
	return i
}

// indexRune returns the index of the first instance of the Unicode code point
// r (ignoring case) and the size of the rune that matched.
func indexRune(s string, r rune) (int, int) {
	// TODO: search using the second or last byte of the rune
	// since it is more unique.
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
		// TODO: benchmark this against iterating through s once.
		if folds, hasFolds := _FoldMap[r]; hasFolds {
			size := utf8.RuneLen(r)
			n := indexRuneCase(s, r)
			if n == 0 {
				return 0, size
			}
			if n > 0 {
				s = s[:n]
			}
			for i := 0; i < len(folds); i++ {
				rr := folds[i]
				if rr == r {
					continue
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

// WARN: case-sensitive
// WARN: second bit should only be used if the first bit is one of:
//
//	240:	31.09%
//	243:	23.25%
//	244:	23.13%
//
// TLDR: if the rune is less than 240 use strings.IndexRune.
func indexRuneCase(s string, r rune) int {
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
		// TODO: see if using the strings.Index fast path for
		// small values is faster here
		// if len(s) <= 16 {
		// 	return strings.Index(s, string(r))
		// }
		var n int
		var c0, c1, c2, c3 byte
		// Inlined version of utf8.EncodeRune
		{
			const (
				t1 = 0b00000000
				tx = 0b10000000
				t2 = 0b11000000
				t3 = 0b11100000
				t4 = 0b11110000

				maskx = 0b00111111

				rune1Max = 1<<7 - 1
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
		// TODO: check `n > len(s)` here ???
		if n == len(s) {
			if string(r) == s {
				return 0
			}
			return -1
		}
		// TODO:
		// 	* search for the last byte since it is the most uniformly distributed
		//	* test if we should have a cutoff for IndexByte and switch over to
		//	  iterating runes
		//
		// Search for r using the second byte of its UTF-8 encoded form
		// since it is more unique than the first byte. This 4-5x faster
		// when all the text is Unicode.
		switch n {
		case 2:
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
				i++
			}
		case 3:
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
				i++
			}
		case 4:
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
				i++
			}
		}
		return -1
	}
}

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
func indexRune2(s string, lower, upper rune) (int, int) {
	// WARN: latest changes made this slightly slower
	if lower|upper < utf8.RuneSelf {
		return indexByte(s, byte(lower))
	}
	n := indexRuneCase(s, lower)
	sz := utf8.RuneLen(lower)
	if n != 0 && lower != upper {
		if 0 <= n && n < len(s) {
			s = s[:n] // limit the search space
		}
		if o := indexRuneCase(s, upper); n == -1 || (o != -1 && o < n) {
			n = o
			sz = utf8.RuneLen(upper)
		}
	}
	return n, sz
}

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
		if folds, hasFolds := _FoldMap[r]; hasFolds {
			switch len(folds) {
			case 1:
				r := folds[0]
				for i := len(s); i > 0; {
					var sr rune
					if sr = rune(s[i-1]); sr < utf8.RuneSelf {
						i--
					} else {
						var size int
						sr, size = utf8.DecodeLastRuneInString(s[:i])
						i -= size
					}
					if sr == r {
						return i
					}
				}
			case 2:
				r0, r1 := folds[0], folds[1]
				for i := len(s); i > 0; {
					var sr rune
					if sr = rune(s[i-1]); sr < utf8.RuneSelf {
						i--
					} else {
						var size int
						sr, size = utf8.DecodeLastRuneInString(s[:i])
						i -= size
					}
					if sr == r0 || sr == r1 {
						return i
					}
				}
			case 3:
				r0, r1, r2 := folds[0], folds[1], folds[2]
				for i := len(s); i > 0; {
					var sr rune
					if sr = rune(s[i-1]); sr < utf8.RuneSelf {
						i--
					} else {
						var size int
						sr, size = utf8.DecodeLastRuneInString(s[:i])
						i -= size
					}
					if sr == r0 || sr == r1 || sr == r2 {
						return i
					}
				}
			case 4:
				r0, r1, r2, r3 := folds[0], folds[1], folds[2], folds[3]
				for i := len(s); i > 0; {
					var sr rune
					if sr = rune(s[i-1]); sr < utf8.RuneSelf {
						i--
					} else {
						var size int
						sr, size = utf8.DecodeLastRuneInString(s[:i])
						i -= size
					}
					if sr == r0 || sr == r1 || sr == r2 || sr == r3 {
						return i
					}
				}
			default:
				panic("invalid number of folds")
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

func hashStrRevUnicode(sep string) (uint32, uint32, int) {
	hash := uint32(0)
	n := 0
	for i := len(sep); i > 0; {
		var r rune
		var size int
		if sep[i-1] < utf8.RuneSelf {
			r, size = rune(sep[i-1]), 1
			if 'A' <= r && r <= 'Z' {
				r += 'a' - 'A'
			}
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

func indexRabinKarpRevUnicode(s, substr string) int {
	// Reverse Rabin-Karp search
	hashss, pow, n := hashStrRevUnicode(substr)
	var h uint32
	i := len(s)
	for i > 0 {
		var r rune
		var size int
		if s[i-1] < utf8.RuneSelf {
			r, size = rune(s[i-1]), 1
			if 'A' <= r && r <= 'Z' {
				r += 'a' - 'A'
			}
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
			r0, n0 = rune(s[i-1]), 1
			if 'A' <= r0 && r0 <= 'Z' {
				r0 += 'a' - 'A'
			}
		} else {
			r0, n0 = utf8.DecodeLastRuneInString(s[:i])
			r0 = caseFold(r0)
		}
		var r1 rune
		var n1 int
		if s[j-1] < utf8.RuneSelf {
			r1, n1 = rune(s[j-1]), 1
			if 'A' <= r1 && r1 <= 'Z' {
				r1 += 'a' - 'A'
			}
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

func indexRabinKarpUnicode(s, substr string) int {
	// Rabin-Karp search
	hashss, pow, n := hashStrUnicode(substr)
	var h uint32
	sz := 0 // byte size of 'n' runes
	for i, r := range s {
		orig := r
		if r < utf8.RuneSelf {
			if 'A' <= r && r <= 'Z' {
				r += 'a' - 'A'
			}
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
		if !isAlpha(c) {
			return strings.Count(s, substr)
		}
		l, u, _ := toUpperLower(rune(c))
		n := strings.Count(s, string(l)) + strings.Count(s, string(u))
		switch c {
		case 'K', 'k':
			n += strings.Count(s, string('K'))
		case 'S', 's':
			n += strings.Count(s, string('ſ'))
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
func makeASCIISet(chars string) (as asciiSet, ok bool) {
	for i := 0; i < len(chars); i++ {
		c := chars[i]
		if c >= utf8.RuneSelf {
			return as, false
		}
		// Can't use ASCII when non-ASCII chars fold to ASCII chars.
		switch c {
		case 'K', 'k', 'S', 's':
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
		if as, isASCII := makeASCIISet(chars); isASCII {
			for i := 0; i < len(s); i++ {
				if as.contains(s[i]) {
					return i
				}
			}
			return -1
		}
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
		if as, isASCII := makeASCIISet(chars); isASCII {
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

// TODO: does this matter?
// See: golang.org/x/sys/cpu.hostByteOrder
const littleEndian = runtime.GOARCH == "386" ||
	runtime.GOARCH == "amd64" ||
	runtime.GOARCH == "amd64p32" ||
	runtime.GOARCH == "alpha" ||
	runtime.GOARCH == "arm" ||
	runtime.GOARCH == "arm64" ||
	runtime.GOARCH == "loong64" ||
	runtime.GOARCH == "mipsle" ||
	runtime.GOARCH == "mips64le" ||
	runtime.GOARCH == "mips64p32le" ||
	runtime.GOARCH == "nios2" ||
	runtime.GOARCH == "ppc64le" ||
	runtime.GOARCH == "riscv" ||
	runtime.GOARCH == "riscv64" ||
	runtime.GOARCH == "sh"

// WARN: rename
// TODO: see if using aligned loads if faster on amd64
// TODO: write in assembly
func IndexNonASCII(s string) int {
	const wordSize = int(unsafe.Sizeof(uintptr(0)))

	// TODO: support big endian
	if !littleEndian {
		for i := 0; i < len(s); i++ {
			if s[i] >= utf8.RuneSelf {
				return i
			}
		}
		return -1
	}

	if len(s) < 8 {
		for i := 0; i < len(s); i++ {
			if s[i] >= utf8.RuneSelf {
				return i
			}
		}
		return -1
	}

	b := *(*[]byte)(unsafe.Pointer(&s))
	n := len(b) / wordSize
	sliceHeader := struct {
		p   unsafe.Pointer
		len int
		cap int
	}{unsafe.Pointer(&b[0]), n, n}

	var i int
	if wordSize == 8 {
		const mask64 uint64 = 0x8080808080808080
		us := *(*[]uint64)(unsafe.Pointer(&sliceHeader))
		for i := 0; i < len(us); i++ {
			if m := us[i] & mask64; m != 0 {
				return i*wordSize + bits.TrailingZeros64(m)/8
			}
		}
		i *= wordSize

	} else /* wordSize == 4 */ {
		const mask32 uint32 = 0x80808080
		us := *(*[]uint32)(unsafe.Pointer(&sliceHeader))
		for i := 0; i < len(us); i++ {
			if m := us[i] & mask32; m != 0 {
				return i*wordSize + bits.TrailingZeros32(m)/8
			}
		}
		i *= wordSize
	}

	// Check remaining bytes
	for ; i < len(s); i++ {
		if s[i] >= utf8.RuneSelf {
			return i
		}
	}
	return -1
}

// WARN WARN WARN WARN WARN WARN WARN WARN WARN WARN
//
// # Use this and optimize for ASCII chars
//
// WARN WARN WARN WARN WARN WARN WARN WARN WARN WARN
// func shortIndexUnicode2(s, substr string) int {
// 	u0, sz0 := utf8.DecodeRuneInString(substr)
// 	u1, sz1 := utf8.DecodeRuneInString(substr[sz0:])
// 	folds0, hasFolds0 := _FoldMapExcludingUpperLower[u0]
// 	folds1, hasFolds1 := _FoldMapExcludingUpperLower[u1]
// 	needle := substr[sz0+sz1:]
//
// 	// Ugly hack
// 	var l0, l1 rune
// 	if u0 != 'İ' {
// 		u0, l0, _ = toUpperLower(u0)
// 	} else {
// 		l0 = 'İ'
// 	}
// 	if u1 != 'İ' {
// 		u1, l1, _ = toUpperLower(u1)
// 	} else {
// 		l1 = 'İ'
// 	}
//
// 	fails := 0
// 	i := 0
//
// 	// WARN: resolve and benchmark this
// 	// TOOD: see if we can stop earlier: `t := len(s) - (len(substr) / 3) + 1`
// 	//
// 	// t := len(s) - (utf8.RuneCountInString(substr) + 1)
// 	t := len(s) - 1
// 	// t := len(s) - (len(substr) / 3) + 1
//
// 	if !hasFolds0 && !hasFolds1 && u0|l0|u1|l1 < utf8.RuneSelf {
// 		l0 := byte(l0)
// 		u0 := byte(u0)
// 		l1 := byte(l1)
// 		u1 := byte(u1)
// 		for i < t {
// 			if s[i] != l0 && s[i] != u0 {
// 				o := IndexByte(s[i+1:], l0)
// 				if o < 0 {
// 					return -1
// 				}
// 				i += o + 1
// 			}
// 			if s[i+1] == l1 || s[i+1] == u1 {
// 				if match, exhausted := hasPrefixUnicode(s[i+1:], needle); match {
// 					return i
// 				} else if exhausted {
// 					return -1
// 				}
// 			}
// 			i++
// 			fails++
//
// 			// FIXME: this needs to be tuned since the brute force
// 			// performance is very different than the stdlibs.
// 			if fails > cutover(i) {
// 				r := bruteForceIndexUnicode(s[i:], substr)
// 				if r >= 0 {
// 					return r + i
// 				}
// 				return -1
// 			}
// 		}
// 	} else {
// 		for i < t {
// 			var r0 rune
// 			var n0 int
// 			if s[i] < utf8.RuneSelf {
// 				r0, n0 = rune(s[i]), 1
// 			} else {
// 				r0, n0 = utf8.DecodeRuneInString(s[i:])
// 			}
//
// 			// TODO: See if we can use our own version of strings.Index(Rune)?
// 			// that is faster. Could pre-compute the Rabin-Karp table, only use
// 			// strings.IndexByte, etc...
//
// 			if r0 != u0 && r0 != l0 && (!hasFolds0 || (r0 != folds0[0] && r0 != folds0[1])) {
// 				var o, sz int
// 				if !hasFolds0 {
// 					o, sz = indexRune2(s[i+n0:], l0, u0)
// 				} else {
// 					o, sz = indexRune(s[i+n0:], l0)
// 				}
// 				if o < 0 {
// 					return -1
// 				}
// 				i += o + n0
// 				n0 = sz // The rune we matched on might not be the same size as c0
// 			}
//
// 			// FIXME: take len(substr) into accout
// 			// TODO: see if we can stop at `t`
// 			if i+n0 >= t {
// 				return -1
// 			}
//
// 			var r1 rune
// 			var n1 int
// 			if s[i+n0] < utf8.RuneSelf {
// 				r1, n1 = rune(s[i+n0]), 1
// 			} else {
// 				r1, n1 = utf8.DecodeRuneInString(s[i+n0:])
// 			}
//
// 			if r1 == u1 || r1 == l1 || (hasFolds1 && (r1 == folds1[0] || r1 == folds1[1])) {
// 				match, exhausted := hasPrefixUnicode(s[i+n0+n1:], needle)
// 				if match {
// 					return i
// 				}
// 				if exhausted {
// 					return -1
// 				}
// 			}
// 			fails++
// 			i += n0
//
// 			// FIXME: this needs to be tuned since the brute force
// 			// performance is very different than the stdlibs.
// 			if fails > cutover(i) {
// 				r := bruteForceIndexUnicode(s[i:], substr)
// 				if r >= 0 {
// 					return r + i
// 				}
// 				return -1
// 			}
// 		}
// 	}
// 	return -1
// }
