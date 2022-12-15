// Package strings is a case-insensitive implementation of the strings package.
// Except where noted, simple Unicode case-folding is used to determine equality.
package strcase

// TODO: make sure package doc is accurate.

// BUG(cvieth): There is no mechanism for full case folding, that is, for
// characters that involve multiple runes in the input or output
// (see: https://pkg.go.dev/unicode#pkg-note-BUG).

//go:generate go run gen.go

import (
	"io"
	"log"
	"math/bits"
	"os"
	"runtime"
	"strings"
	"unicode"
	"unicode/utf8"
	"unsafe"
)

// WARN WARN WARN
func init() {
	log.SetFlags(log.Lshortfile)
	log.SetPrefix("### ")
	if true {
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

// TODO: use code folding
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
			tr = unicode.To(unicode.LowerCase, tr)
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

// HasPrefix tests whether the string s begins with prefix ignoring case.
func HasPrefix(s, prefix string) bool {
	ok, _ := hasPrefixUnicode(s, prefix)
	return ok
}

// hasPrefixUnicode returns if string s begins with prefix (ignoring case) and
// if s was exhausted before a match was found.
func hasPrefixUnicode(s, prefix string) (bool, bool) {
	// The max difference in encoded lengths between cases is 2 bytes for
	// [kK] (Latin - 1 byte) and 'K' (Kelvin - 3 bytes).
	n := len(s)
	if n*3 < len(prefix) || (n*2 < len(prefix) && !strings.Contains(prefix, string('\u212A'))) {
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

	return true, len(s) == 0 // s exhausted
}

// TODO: replace with
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
	if ns*3 < nt || (ns*2 < nt && !strings.Contains(suffix, string('\u212A'))) {
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
		} else {
			r, size := utf8.DecodeLastRuneInString(s)
			sr, s = r, s[:len(s)-size]
		}
		if n := len(t) - 1; t[n] < utf8.RuneSelf {
			tr, t = rune(t[n]), t[:n]
		} else {
			r, size := utf8.DecodeLastRuneInString(t)
			tr, t = r, t[:len(t)-size]
		}

		if sr == tr {
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
			return false, len(s) == 0, 0
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
		return false, len(s) == 0, 0
	}

	// log.Printf("i: %d ns: %d s: %d", i, ns, len(s))
	return len(t) == 0, len(s) == 0, len(s)
	// return len(t) == 0, len(s) == 0, ns - len(s)
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
		i := strings.Index(s, string(u0))
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
	// TOOD: see if we can stop earlier: `t := len(s) - (len(substr) / 3) + 1`
	for i < len(s) {
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
	// TOOD: see if we can use `t` to stop earlier.
	t := len(s) - (len(substr) / 3) + 1
	for i < len(s) {
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

		// TODO: see if we can stop at `t`
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

// Index returns the index of the first instance of substr in s, or -1 if
// substr is not present in s.
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
		if n > len(s)*2 && !strings.Contains(substr, string('\u212A')) {
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
		if len(s) <= maxBruteForce {
			return bruteForceIndexUnicode(s, substr)
		}
		return shortIndexUnicode(s, substr)
	}
	return indexUnicode(s, substr)
}

// LastIndex returns the index of the last instance of substr in s, or -1 if substr is not present in s.
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
		if n > len(s)*2 && !strings.Contains(substr, string('\u212A')) {
			return -1
		}
		return bruteForceLastIndexUnicode(s, substr)
	}
	i := indexRabinKarpRevUnicode(s, substr)
	if i != -2 {
		return i
	}
	// TODO: see if we can use Rabin-Karp to skip some runes
	return bruteForceLastIndexUnicode(s, substr)
}

// FIXME: document
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
	if n > 0 && len(s) >= 16 {
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
	case 'K', 'k':
		r = 'K'
		sz = 3
	case 'S', 's':
		r = 'ſ'
		sz = 2
	default:
		return n, 1
	}

	if n > 0 && len(s) >= 16 {
		s = s[:n]
	}
	// strings.IndexRune uses strings.Index and we know the rune is valid.
	if o := strings.Index(s, string(r)); n == -1 || (o != -1 && o < n) {
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

	// Special case for Unicode characters that map to ASCII.
	var r rune
	switch c {
	case 'K', 'k':
		r = 'K'
	case 'S', 's':
		r = 'ſ'
	default:
		if !isAlpha(c) {
			return strings.LastIndexByte(s, c)
		}
		// ASCII and c is a lower/uppper case character
		c0 := c
		c1 := c ^ ' ' // swap case
		for i := len(s) - 1; i >= 0; i-- {
			if s[i] == c0 || s[i] == c1 {
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

func lastIndexRune(s string, r rune) int {
	switch {
	case 0 <= r && r < utf8.RuneSelf:
		return LastIndexByte(s, byte(r))
	case r == utf8.RuneError:
		// WARN: test this
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

// func indexRune2(s string, lower, upper rune) (int, int) {
// 	// WARN: this is broken because we need to keep searching
// 	// through s with IndexBytes
// 	n := -1
// 	sz := 0
// 	uc := string(upper)[0]
// 	lc := string(lower)[0]
// 	o := strings.IndexByte(s, uc)
// 	if o >= 0 {
// 		if strings.HasPrefix(s[o:], string(upper)) {
// 			n = o
// 			sz = utf8.RuneLen(upper)
// 		} else if uc == lc && strings.HasPrefix(s[o:], string(lower)) {
// 			n = o
// 			sz = utf8.RuneLen(lower)
// 		}
// 	}
// 	if n != 0 && lc != uc {
// 		if n > 0 {
// 			s = s[:n]
// 		}
// 		o := strings.IndexByte(s, lc)
// 		if o >= 0 && (n == -1 || o < n) && strings.HasPrefix(s[o:], string(lower)) {
// 			n = o
// 			sz = utf8.RuneLen(lower)
// 		}
// 	}
// 	return n, sz
// }

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
			if mustFold(r) {
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
			if mustFold(r) {
				return 0, 0, -2
			}
			r = unicode.To(unicode.LowerCase, r)
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
	if n == -2 {
		return -2
	}
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
			if mustFold(r) {
				return -2
			}
			r = unicode.To(unicode.LowerCase, r)
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
			if mustFold(r0) {
				return -2
			}
			r0 = unicode.To(unicode.LowerCase, r0)
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
			if mustFold(r1) {
				return -2
			}
			r1 = unicode.To(unicode.LowerCase, r1)
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
	if n == -2 {
		return -2
	}
	var h uint32
	sz := 0 // byte size of 'o' runes
	for i, r := range s {
		if mustFold(r) {
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
			if mustFold(s0) {
				return -2
			}
			s0 = unicode.To(unicode.LowerCase, s0)
		}
		if s[i] < utf8.RuneSelf {
			s1, n1 = rune(_lower[s[i]]), 1
		} else {
			s1, n1 = utf8.DecodeRuneInString(s[i:])
			if mustFold(s1) {
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

	if wordSize == 8 {
		const mask64 uint64 = 0x8080808080808080
		us := *(*[]uint64)(unsafe.Pointer(&sliceHeader))
		i := 0
		for i := 0; i < len(us); i++ {
			if m := us[i] & mask64; m != 0 {
				return i*wordSize + bits.TrailingZeros64(m)/8
			}
		}
		i *= wordSize

	} else /* wordSize == 4 */ {
		const mask32 uint32 = 0x80808080
		us := *(*[]uint32)(unsafe.Pointer(&sliceHeader))
		i := 0
		for i := 0; i < len(us); i++ {
			if m := us[i] & mask32; m != 0 {
				return i*wordSize + bits.TrailingZeros32(m)/8
			}
		}
		i *= wordSize
	}

	for i := 0; i < len(s); i++ {
		if s[i] >= utf8.RuneSelf {
			return i
		}
	}
	return -1
}
