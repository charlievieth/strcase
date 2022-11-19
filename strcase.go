package strcase

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
		sr := _lower[s[i]]
		tr := _lower[t[i]]
		if sr == tr {
			continue
		}
		if sr|tr >= utf8.RuneSelf {
			goto hasUnicode
		}
		return clamp(int(_lower[sr]) - int(_lower[tr]))

		// sr := s[i]
		// tr := t[i]
		// if sr|tr >= utf8.RuneSelf {
		// 	goto hasUnicode
		// }
		// if sr == tr || _lower[sr] == _lower[tr] {
		// 	continue
		// }
		// return clamp(int(_lower[sr]) - int(_lower[tr]))
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
		} else {
			r, size := utf8.DecodeRuneInString(t)
			tr, t = r, t[size:]
		}

		// If they match, keep going; if not, return false.

		// Easy case.
		if tr == sr {
			continue
		}

		// Make sr < tr to simplify what follows.
		sign := 1
		if tr < sr {
			sign = -1
			tr, sr = sr, tr
		}
		// Fast check for ASCII.
		if tr < utf8.RuneSelf {
			// ASCII only, sr/tr must be upper/lower case
			if 'A' <= sr && sr <= 'Z' && tr == sr+'a'-'A' {
				continue
			}
			return clamp(int(sr)-int(tr)) * sign
		}

		r := unicode.SimpleFold(sr)
		for r != sr && r < tr {
			r = unicode.SimpleFold(r)
		}
		if r == tr {
			continue
		}
		return clamp(int(sr)-int(tr)) * sign
	}
	if len(t) == 0 {
		return 0
	}
	return -1
}

func isLower(c byte) bool { return 'a' <= c && c <= 'z' }
func isUpper(c byte) bool { return 'A' <= c && c <= 'Z' }
func isAlpha(c byte) bool { return isUpper(c) || isLower(c) }

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

var _asciiSafe = [256]bool{
	true, true, true, true, true, true, true, true, true, true, true, true,
	true, true, true, true, true, true, true, true, true, true, true, true,
	true, true, true, true, true, true, true, true, true, true, true, true,
	true, true, true, true, true, true, true, true, true, true, true, true,
	true, true, true, true, true, true, true, true, true, true, true, true,
	true, true, true, true, true, true, true, true, true, true, true, true,
	true, false, true, false, true, true, true, true, true, true, true, true,
	true, true, true, true, true, true, true, true, true, true, true, true,
	true, true, true, true, true, true, true, true, true, false, true, false,
	true, true, true, true, true, true, true, true, true, true, true, true,
	true, true, true, true, true, true, true, true, false, false, false, false,
	false, false, false, false, false, false, false, false, false, false,
	false, false, false, false, false, false, false, false, false, false,
	false, false, false, false, false, false, false, false, false, false,
	false, false, false, false, false, false, false, false, false, false,
	false, false, false, false, false, false, false, false, false, false,
	false, false, false, false, false, false, false, false, false, false,
	false, false, false, false, false, false, false, false, false, false,
	false, false, false, false, false, false, false, false, false, false,
	false, false, false, false, false, false, false, false, false, false,
	false, false, false, false, false, false, false, false, false, false,
	false, false, false, false, false, false, false, false, false, false,
	false, false, false, false, false, false, false, false, false, false,
	false, false, false, false,
}

// var _asciiSafe = [128]bool{
// 	true, true, true, true, true, true, true, true, true, true, true, true,
// 	true, true, true, true, true, true, true, true, true, true, true, true,
// 	true, true, true, true, true, true, true, true, true, true, true, true,
// 	true, true, true, true, true, true, true, true, true, true, true, true,
// 	true, true, true, true, true, true, true, true, true, true, true, true,
// 	true, true, true, true, true, true, true, true, true, true, true, true,
// 	true, false, true, false, true, true, true, true, true, true, true, true,
// 	true, true, true, true, true, true, true, true, true, true, true, true,
// 	true, true, true, true, true, true, true, true, true, false, true, false,
// 	true, true, true, true, true, true, true, true, true, true, true, true,
// 	true, true, true, true, true, true, true, true,
// }

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

// HasPrefix tests whether the string s begins with prefix ignoring case.
func HasPrefix(s, prefix string) bool {
	ok, _ := hasPrefixUnicode(s, prefix)
	return ok
}

// hasPrefixUnicode returns if string s begins with prefix (ignoring case) and
// if s was exhausted before a match was found.
func hasPrefixUnicode(s, prefix string) (bool, bool) {
	// The max difference in encoded lengths between cases is 2 bytes (K).
	if len(s)*2 < len(prefix) {
		return false, true
	}

	// TODO: see if this is faster in some cases
	//
	// if strings.HasPrefix(s, prefix) {
	// 	return true, len(s) == len(prefix)
	// }

	// TODO: use optimized code from Compare()

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

		tr = unicode.ToLower(tr)
		if sr == tr || unicode.ToLower(sr) == tr {
			continue
		}
		return false, len(s) == 0
	}

	return true, len(s) == 0 // Prefix exhausted
}

const maxBruteForce = 16 // substring length
const maxLen = 32        // subject length

func bruteForceIndexASCII(s, substr string) int {
	// n1 := len(substr) - 1
	// if n1 > 32 {
	// 	n1 = 32
	// }
	// c0 := _lower[substr[0]]
	// c1 := _lower[substr[n1]] // check a non-relative but close index

	// n1 := len(substr) - 1
	// if n1 > 32 {
	// 	n1 = 32
	// }
	c0 := _lower[substr[0]]
	c1 := _lower[substr[1]]

	t := len(s) - len(substr) + 1
	for i := 0; i < t; i++ {
		// if _lower[s[i]] == c0 && _lower[s[i+1]] == c1 {
		// 	if hasPrefixASCII(s[i+2:], substr[2:]) {
		// 		return i
		// 	}
		// }

		// if _lower[s[i]] != c0 {
		// 	continue
		// }
		// if hasPrefixASCII(s[i:], substr) {
		// 	return i
		// }

		if _lower[s[i]] == c0 && _lower[s[i+1]] == c1 && hasPrefixASCII(s[i+2:], substr[2:]) {
			return i
		}
		// if _lower[s[i]] == c0 && _lower[s[i+n1]] == c1 && hasPrefixASCII(s[i:], substr) {
		// 	return i
		// }
	}
	return -1
}

// TODO: pass the first decoded rune
func bruteForceIndexUnicode_OLD(s, substr string) int {
	if len(substr) == 0 {
		return 0 // WARN: this should never happen
	}

	// WARN WARN WARN
	// if utf8.RuneCountInString(substr) < 2 {
	// 	panic("BAD")
	// }

	c0, sz := utf8.DecodeRuneInString(substr)
	if sz == len(substr) {
		return IndexRune(s, c0) // WARN: this should never happen
	}
	// c1, _ := utf8.DecodeRuneInString(substr[sz:])

	c0 = unicode.ToLower(c0)
	// c1 = unicode.ToLower(c1)

	for i, r1 := range s {
		if 'A' <= r1 && r1 <= 'Z' {
			r1 += 'a' - 'A'
		} else if r1 > unicode.MaxASCII {
			r1 = unicode.To(unicode.LowerCase, r1)
		}
		if r1 == c0 {
			// TODO: skip first rune since we know it matches

			// match, noMore := hasPrefixUnicode(s[i+utf8.RuneLen(x):], substr[sz:])
			match, noMore := hasPrefixUnicode(s[i:], substr[:])
			if match {
				return i
			}
			if noMore {
				break
			}
		}
	}
	return -1
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

func hasFolds(sr rune) bool {
	r := unicode.SimpleFold(sr)
	n := 1
	for r != sr && n < 3 {
		n++
		r = unicode.SimpleFold(r)
	}
	return n > 2
}

func mustLower(r rune) bool {
	// Runes where `r != ToUpper(ToLower(r))`
	switch r {
	case 'I', 'Θ', 'ß', 'Ω', 'K', 'Å', 'i', 'θ', 'ω', 'k', 'å':
		return true
	default:
		rr := unicode.SimpleFold(r)
		n := 1
		for rr != r && n < 3 {
			n++
			rr = unicode.SimpleFold(rr)
		}
		return n > 2
	}
}

type caseFolds struct {
	folds [4]rune
}

func (c *caseFolds) init(r rune) {
	if c.folds[0] != 0 {
		c.folds = [4]rune{} // zero
	}
	c.folds[0] = r
	rr := unicode.SimpleFold(r)
	i := 0
	for rr != r {
		i++
		c.folds[i] = rr
		rr = unicode.SimpleFold(rr)
	}
	c.sort(i + 1)
	rh := c.folds[i]
	for ; i < len(c.folds); i++ {
		c.folds[i] = rh
	}
}

func (c *caseFolds) sort(n int) {
	a := &c.folds
	for i := 1; i < n; i++ {
		for j := i; j > 0 && a[j] < a[j-1]; j-- {
			a[j], a[j-1] = a[j-1], a[j]
		}
	}
}

/*
func insertionSort(data Interface, a, b int) {
	for i := a + 1; i < b; i++ {
		for j := i; j > a && data.Less(j, j-1); j-- {
			data.Swap(j, j-1)
		}
	}
}
*/

type foldSet struct {
	folds [4]rune
}

func (c *foldSet) Init(r rune) {
	if c.folds[0] != 0 {
		c.folds = [4]rune{} // zero
	}
	c.folds[0] = r
	rr := unicode.SimpleFold(r)
	i := 0
	for rr != r {
		i++
		c.folds[i] = rr
		rr = unicode.SimpleFold(rr)
	}
	c.sort(i + 1)
}

func (c *foldSet) sort(n int) {
	a := &c.folds
	for i := 1; i < n; i++ {
		for j := i; j > 0 && a[j] < a[j-1]; j-- {
			a[j], a[j-1] = a[j-1], a[j]
		}
	}
}

func (c *foldSet) Match(r rune) bool {
	for i := 0; i < len(c.folds); i++ {
		f := c.folds[i]
		if f == r {
			return true
		}
		// if f > r || f == 0 {
		if f == 0 {
			break
		}
	}
	return false
}

// WARN WARN WARN: do something like this (but make it work)
func bruteForceIndexUnicode(s, substr string) int {
	if len(substr) == 0 {
		return 0 // WARN: this should never happen
	}

	// WARN WARN WARN
	// if utf8.RuneCountInString(substr) < 2 {
	// 	panic("BAD")
	// }

	u0, sz0 := utf8.DecodeRuneInString(substr)
	if sz0 == len(substr) {
		return IndexRune(s, u0) // WARN: this should never happen
	}
	u1, sz1 := utf8.DecodeRuneInString(substr[sz0:])
	needle := substr[sz0+sz1:]

	var l0, l1 rune
	u0, l0, _ = toUpperLower(u0)
	u1, l1, _ = toUpperLower(u1)

	fold0 := mustLower(u0)
	fold1 := mustLower(u1)

	i := 0
	t := len(s) - (len(substr) / 2) + 1
	for i < t {
		var n0 int
		var r0 rune
		if s[i] < utf8.RuneSelf {
			r0, n0 = rune(s[i]), 1
		} else {
			r0, n0 = utf8.DecodeRuneInString(s[i:])
		}
		if r0 != u0 && r0 != l0 {
			// WARN: this is wrong if u0 is lowercase
			if !fold0 || unicode.ToLower(r0) != l0 {
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
			// WARN: this is wrong if u0 is lowercase
			if !fold1 {
				i += n0
				continue
			}
			if r1 = unicode.ToLower(r1); r1 != l1 {
				i += n0
				if r1 == l0 {
					i += n1
				}
				continue
			}
			// if !fold1 || unicode.ToLower(r1) != l1 {
			// 	i += n0
			// 	continue
			// }

			// if rl1 := unicode.ToLower(r1); rl1 != l1 {
			// 	i += n0
			// 	// TODO: benchmark this
			// 	if rl1 != l0 {
			// 		panic("HERE")
			// 		i += n1 // skip second byte
			// 	}
			// 	continue
			// }
		}
		match, noMore := hasPrefixUnicode(s[i+n0+n1:], needle)
		if match {
			return i
		}
		if noMore {
			break
		}
		i += n0
		if !fold0 && r1 != u0 && r1 != l0 {
			i += n1
		}
		// if r1 == l0 {
		// 	i += n1
		// }

		// if r1 != u1 && r1 != l1 && u1 != l1 {
		// 	r1 = unicode.ToLower(r1)
		// }
		// if r1 == l1 {
		// 	match, noMore := hasPrefixUnicode(s[i+n0+n1:], needle)
		// 	if match {
		// 		return i
		// 	}
		// 	if noMore {
		// 		break
		// 	}
		// }
		// i += n0
		// // if r1 != l0 {
		// // 	i += n1
		// // }
	}
	return -1
}

func bruteForceIndexUnicode_NEW(s, substr string) int {
	u0, sz0 := utf8.DecodeRuneInString(substr)
	if sz0 == len(substr) {
		return IndexRune(s, u0) // WARN: this should never happen
	}
	u1, sz1 := utf8.DecodeRuneInString(substr[sz0:])
	needle := substr[sz0+sz1:]

	var l0, l1 rune
	u0, l0, _ = toUpperLower(u0)
	u1, l1, _ = toUpperLower(u1)

	fold0 := mustLower(u0)
	fold1 := mustLower(u1)

	t := len(s) - (len(substr) / 2) + 1
	for i, r0 := range s[:t] {
		if r0 != u0 && r0 != l0 {
			// WARN: this is wrong if u0 is lowercase
			if !fold0 || unicode.ToLower(r0) != l0 {
				continue
			}
		}

		n0 := utf8.RuneLen(r0)
		if i+n0 == len(s) {
			break
		}

		var r1 rune
		var n1 int
		if s[i+n0] < utf8.RuneSelf {
			r1, n1 = rune(s[i+n0]), 1
		} else {
			r1, n1 = utf8.DecodeRuneInString(s[i+n0:])
		}
		if r1 != u1 && r1 != l1 {
			if !fold1 || unicode.ToLower(r1) != l1 {
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

func cutover(n int) int {
	// FIXME: our cutoff is probably different since our algo is not optimized

	if runtime.GOARCH == "amd64" {
		// 1 error per 8 characters, plus a few slop to start.
		return (n + 16) / 8

	}
	// arm64
	// 1 error per 16 characters, plus a few slop to start.
	return 4 + n>>4
	// return (n + 16) / 8
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
			o := IndexByte(s[i+1:t], c0)
			log.Printf("%d: IndexByte(%q, %c) = %d", i, s[i+1:t], c0, o)
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
			// WARN: this is broken !!!
			// CEV: Rabin-Karp is faster here and more consistent
			// return indexRabinKarp(s[i:], substr)

			// Switch to bytealg.IndexString when IndexByte produces too many false positives.
			r := bruteForceIndexASCII(s[i:], substr)
			if r >= 0 {
				return r + i
			}
			return -1
		}
	}
	return -1
}

func shortIndexUnicode(s, substr string) int {
	c0, size := utf8.DecodeRuneInString(substr)
	c1, _ := utf8.DecodeRuneInString(substr[size:])
	// TODO: check if c0 and c1 are not-letters and thus don't need to use
	// caseless comparisons.
	c0 = unicode.ToLower(c0)
	c1 = unicode.ToLower(c1)

	// TODO: stop iteration earlier

	i := 0
	fails := 0
	for i < len(s) {
		var r0 rune
		var n0 int
		if s[i] < utf8.RuneSelf {
			r0, n0 = rune(s[i]), 1
		} else {
			r0, n0 = utf8.DecodeRuneInString(s[i:])
		}
		if r0 != c0 && unicode.ToLower(r0) != c0 {
			o, sz := indexRune(s[i+n0:], c0)
			if o < 0 {
				return -1
			}
			i += o + n0
			n0 = sz // The rune we matched on might not be the same size as c0
		}

		// WARN
		if i+n0 >= len(s) {
			return -1
		}

		var r1 rune
		if s[i+n0] < utf8.RuneSelf {
			r1 = rune(s[i+n0])
		} else {
			r1, _ = utf8.DecodeRuneInString(s[i+n0:])
		}
		if r1 == c1 || unicode.ToLower(r1) == c1 {
			match, exhausted := hasPrefixUnicode(s[i:], substr)
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

func indexUnicode(s, substr string) int {
	c0, size := utf8.DecodeRuneInString(substr)
	c1, _ := utf8.DecodeRuneInString(substr[size:])
	c0 = unicode.ToLower(c0)
	c1 = unicode.ToLower(c1)

	i := 0
	t := len(s) - (len(substr) / 2) + 1
	fails := 0
	for i < t {
		var r0 rune
		var n0 int
		if s[i] < utf8.RuneSelf {
			r0, n0 = rune(s[i]), 1
		} else {
			r0, n0 = utf8.DecodeRuneInString(s[i:])
		}
		if r0 != c0 && unicode.ToLower(r0) != c0 {
			o, sz := indexRune(s[i+n0:t], c0)
			if o < 0 {
				return -1
			}
			i += o + n0
			n0 = sz // The rune we matched on might not be the same size
		}
		// if i+n0 >= len(s) {
		// 	log.Panicf("%d: indexUnicode(%q, %q)", i, s, substr)
		// 	return -1
		// }
		var r1 rune
		// WARN: can panic
		if s[i+n0] < utf8.RuneSelf {
			r1 = rune(s[i+n0])
		} else {
			r1, _ = utf8.DecodeRuneInString(s[i+n0:])
		}
		if r1 == c1 || unicode.ToLower(r1) == c1 {
			match, exhausted := hasPrefixUnicode(s[i:], substr)
			if match {
				return i
			}
			if exhausted {
				return -1
			}
		}
		i += n0
		fails++

		// WARN WARN WARN
		//
		// FIXME
		//
		// WARN WARN WARN
		if fails >= 4+i>>4 /*&& i < t*/ {
			// // See comment in ../bytes/bytes.go.
			// j := indexRabinKarpUnicode(s[i:], substr)
			// if j < 0 {
			// 	return -1
			// }
			// return i + j
		}
	}
	return -1
}

func Index(s, substr string) int {
	// WARN WARN WARN
	//
	// Only works if s is all ASCII
	//
	// WARN WARN WARN

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
		if n > len(s)*2 {
			return -1
		}
		if len(s) == 0 {
			return 0
		}
		// TODO: optimize
		return bruteForceIndexUnicode(s, substr)
	case n <= maxLen: // WARN: 32 is for arm64 (see: bytealg.MaxLen)
		if isASCII(substr) {
			if len(s) <= maxBruteForce {
				return bruteForceIndexASCII(s, substr)
			}
			return shortIndexASCII(s, substr)
		}

		// WARN: remove
		if substr[0] >= utf8.RuneSelf {
			r, size := utf8.DecodeRuneInString(substr)
			if size == len(substr) {
				return IndexRune(s, r)
			}
		}

		// WARN: profile bruteForceIndexUnicode()
		if len(s) <= maxBruteForce {
			return bruteForceIndexUnicode(s, substr)
		}
		// WARN WARN WARN WARN WARN WARN WARN WARN WARN WARN
		// WARN WARN WARN WARN WARN WARN WARN WARN WARN WARN
		return bruteForceIndexUnicode(s, substr)
		// return shortIndexUnicode(s, substr)
	}
	if isASCII(substr) {
		// return indexRabinKarp(s, substr)
		return indexASCII(s, substr)
	}
	// WARN WARN WARN WARN WARN WARN WARN WARN WARN WARN
	// WARN WARN WARN WARN WARN WARN WARN WARN WARN WARN
	return bruteForceIndexUnicode(s, substr)
	// return indexUnicode(s, substr)
}

// WARN: this breaks if there is a non-ASCII form of byte c
func IndexByte(s string, c byte) int {
	i, _ := indexByte(s, c)
	return i
}

func indexByte(s string, c byte) (int, int) {
	// TODO TODO TODO TODO TODO TODO TODO
	//
	// Iterate folds

	n := strings.IndexByte(s, c)
	if n == 0 || !isAlpha(c) {
		return n, 1
	}
	if n != -1 && n <= len(s)/2 && len(s) >= 256 {
		s = s[:n] // limit search space
	}

	c ^= ' ' // swap case
	if o := strings.IndexByte(s, c); n == -1 || (o != -1 && o < n) {
		n = o
	}

	// WARN: maybe only allow 'K' (since 'İ' doesn't fold to 'i')
	//
	// Special case for Unicode characters that map to ASCII.
	var r rune
	var sz int
	switch c {
	case 'I', 'i':
		r = 'İ'
		sz = 2
	case 'K', 'k':
		r = 'K'
		sz = 3
	default:
		return n, 1
	}

	if 0 <= n && n < len(s) {
		s = s[:n]
	}
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
			return n, size
		}
		if n > 0 {
			s = s[:n]
		}
		rr := unicode.SimpleFold(r)
		for rr != r {
			o := strings.Index(s, string(rr))
			if o != -1 && (n == -1 || o < n) {
				n = o
				s = s[:n]
				size = utf8.RuneLen(rr)
			}
			rr = unicode.SimpleFold(rr)
		}
		return n, size
	}
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

// TODO: do we need separate funcs for this???
func hashStrUnicode(sep string) (uint32, uint32) {
	hash := uint32(0)
	for _, r := range sep {
		hash = hash*primeRK + uint32(unicode.ToLower(r))
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
	hashss, pow := hashStrUnicode(substr)
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

func indexRabinKarpUnicode(s, substr string) int {
	// Rabin-Karp search
	hashss, pow := hashStr(substr)
	n := len(substr)
	var h uint32
	for _, r := range substr {
		h = h*primeRK + uint32(unicode.ToLower(r))
	}
	if h == hashss && strings.EqualFold(s[:n], substr) {
		return 0
	}
	for i := n; i < len(s); {
		h *= primeRK
		s0, size := utf8.DecodeRuneInString(s[i:])
		s1, _ := utf8.DecodeRuneInString(s[i-n:])
		h += uint32(unicode.ToLower(s0))
		h -= pow * uint32(unicode.ToLower(s1))
		i += size
		if h == hashss && strings.EqualFold(s[i-n:i], substr) {
			return i - n
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

// // TODO: the unicode symbols 'ſ' (Latin s) and 'K' (Kelvin) map to ASCII symbols.
// var asciiFolds = map[rune][]rune{
// 	'ſ': {'ſ', 'S', 's'}, // Latin letter long s, an obsolete variant of s
// 	'K': {'K', 'K', 'k'}, // Kelvin sign
// }
