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

func toLower(c byte) byte {
	if isUpper(c) {
		c += 'a' - 'A'
	}
	return c
}

func toUpper(c byte) byte {
	if isLower(c) {
		c -= 'a' - 'A'
	}
	return c
}

// WARN: use IndexNonASCII()
func isASCII(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] >= utf8.RuneSelf {
			return false
		}
	}
	return true
}

func hasUnicode(s string) bool { return !isASCII(s) }

// func indexASCII(s, substr string) int {
// 	return 0
// }

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

		// // General case. SimpleFold(x) returns the next equivalent rune > x
		// // or wraps around to smaller values.
		// r := unicode.SimpleFold(sr)
		// for r != sr && r < tr {
		// 	r = unicode.SimpleFold(r)
		// }
		// // for r != sr {
		// // 	r = unicode.SimpleFold(r)
		// // }
		// if r == tr {
		// 	continue
		// }
		// // if unicode.ToLower(sr) == unicode.ToLower(tr) {
		// // 	continue
		// // }
		// // log.Printf("sr: %[1]c - %[1]U\n", sr)
		// // log.Printf("tr: %[1]c - %[1]U\n", tr)
		// // panic("WAT")
		// return false
	}

	return true, len(s) == 0 // Prefix exhausted
}

const maxBruteForce = 16 // substring length
const maxLen = 32        // subject length

func bruteForceIndexASCII(s, substr string) int {
	c0 := toLower(substr[0])
	// c1 := toLower(substr[1])
	n := len(s) - len(substr) + 1
	for i := 0; i < n; i++ {
		// if _lower[s[i]] == c0 && _lower[s[i+1]] == c1 {
		// 	if hasPrefixASCII(s[i+2:], substr[2:]) {
		// 		return i
		// 	}
		// }
		if _lower[s[i]] != c0 {
			continue
		}
		if hasPrefixASCII(s[i:], substr) {
			return i
		}
	}
	return -1
}

func bruteForceIndexUnicode(s, substr string) int {
	if len(substr) == 0 {
		return 0 // WARN: this should never happen
	}

	r0, sz := utf8.DecodeRuneInString(substr)
	if sz == len(substr) {
		return IndexRune(s, r0) // WARN: this should never happen
	}

	r0 = unicode.ToLower(r0)
	for i, r1 := range s {
		if r1 <= unicode.MaxASCII {
			if 'A' <= r1 && r1 <= 'Z' {
				r1 += 'a' - 'A'
			}
		} else if r1 != r0 {
			r1 = unicode.To(unicode.LowerCase, r1)
		}
		if r1 == r0 {
			match, noMore := hasPrefixUnicode(s[i:], substr)
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

func bruteForceIndexUnicode_OLD(s, substr string) int {
	if len(substr) == 0 {
		return 0 // WARN: this should never happen
	}
	c0, _ := utf8.DecodeRuneInString(substr) // TODO: rename
	folds := make([]rune, 1, 4)
	folds[0] = c0
	for r := unicode.SimpleFold(c0); r != c0; r = unicode.SimpleFold(r) {
		folds = append(folds, r)
	}
	// r := unicode.SimpleFold(c0)
	// for r != c0 {
	// 	folds = append(folds, r)
	// 	r = unicode.SimpleFold(r)
	// }

	// TODO:
	// 	1. Use strings.Index() with folds ???
	// 	2. Switch based on len(folds) ???

	for i, sr := range s {
		for _, tr := range folds {
			if sr != tr {
				continue
			}
			match, noMore := hasPrefixUnicode(s[i:], substr)
			if match {
				return i
			}
			if noMore {
				break
			}
		}
	}

	// for i, sr := range s[:len(s)-len(substr)+size] {
	// 	for _, tr := range folds {
	// 		if sr == tr && hasPrefixUnicode(s[i:], substr) {
	// 			return i
	// 		}
	// 	}
	// }
	return -1
}

// WARN: DELETE ME
// func shittyBruteForceIndexUnicode(s, substr string) int {
// 	panic("DELETE ME")
// 	r, _ := utf8.DecodeRuneInString(substr) // TODO: rename
// 	r = unicode.ToLower(r)
//
// 	for i, sr := range s {
// 		if sr == r || unicode.ToLower(sr) == r {
// 			if hasPrefixUnicode(s[i:], substr) {
// 				return i
// 			}
// 		}
// 	}
// 	return -1
// }

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
	n := len(substr)
	c0, size := utf8.DecodeRuneInString(substr)
	c1, _ := utf8.DecodeRuneInString(substr[size:])
	// TODO: check if c0 and c1 are not-letters and thus don't need to use
	// caseless comparisons.
	c0 = unicode.ToLower(c0)
	c1 = unicode.ToLower(c1)

	i := 0
	t := len(s) - n + size // WARN: this is wrong
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
			log.Printf("%d: HIT HIT", i)
			if i+n0 >= t {
				// WARN WARN WARN WARN WARN WARN WARN
				//
				// Looks like we are consuming too many chars from s
				//
				// WARN WARN WARN WARN WARN WARN WARN

				log.Printf("i: %d t: %d s: %q", i+n0, t, s[i+n0:])

				// x := shittyBruteForceIndexUnicode(s[i+n0:], substr)
				// log.Printf("shittyBruteForceIndexUnicode(%q, %q) = %d", s[i+n0:], substr, x)
				// return shittyBruteForceIndexUnicode(s[i+n0:], substr)

				// if hasPrefixUnicode(s[i+n0:], substr) {
				// 	return i + n0
				// }
				// This can happen when the encoded size
				// of upper/lower case runes differs.
				return -1
			}
			log.Println("i:", i)
			// o, sz := indexRune(s[i+n0:t], c0)
			// log.Printf("IndexRune(%q, %q) = %d", s[i+n0:t], c0, o)
			o, sz := indexRune(s[i+n0:], c0)
			log.Printf("IndexRune(%q, %q) = %d", s[i+n0:], c0, o)
			if o < 0 {
				return -1
			}
			log.Printf("S: %q : %q", s[i+n0:i+n0+o], s[i+o+n0:])

			// WARN WARN WARN WARN WARN WARN
			//
			// I think something here is wrong when encoded lengths differ.
			//
			// WARN WARN WARN WARN WARN WARN

			i += o + n0
			n0 = sz // The rune we matched on might not be the same size as c0

		} else {
			log.Printf("i: %d r0: %c c0: %c\n", i, r0, c0)
		}
		var r1 rune
		var n1 int
		if s[i+n0] < utf8.RuneSelf {
			r1, n1 = rune(s[i+n0]), 1
		} else {
			r1, n1 = utf8.DecodeRuneInString(s[i+n0:])
		}
		log.Printf("%d: %q\n", i, s[i:])
		log.Printf("%d: %q\n", i, s[i+n0:])

		log.Printf("i: %d r1: %c c1: %c - %t\n", i,
			unicode.ToLower(r1), unicode.ToLower(c1),
			unicode.ToLower(r1) == unicode.ToLower(c1))

		// if r1 == c1 || unicode.ToLower(r1) == c1 {
		// 	log.Println("HIT HIT")
		// 	log.Printf("hasPrefixUnicode(%q, %q) = %t\n", s[i:i+n], substr, hasPrefixUnicode(s[i:], substr))
		// }
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
		// WARN WARN WARN WARN WARN WARN WARN WARN WARN WARN WARN WARN
		// This was the issue
		i += n0
		_ = n1
		// i += n0 + n1
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
	n := len(substr)
	c0, size := utf8.DecodeRuneInString(substr)
	c1, _ := utf8.DecodeRuneInString(substr[size:])
	c0 = unicode.ToLower(c0)
	c1 = unicode.ToLower(c1)

	i := 0
	t := len(s) - n + size // WARN: broken (maybe)
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
			if i+n0 >= t {
				// This can happen when the encoded size
				// of upper/lower case runes differs.
				return -1
			}
			o, sz := indexRune(s[i+n0:t], c0)
			// log.Printf("IndexRune(%q, %q) = %d", s[i+n0:t], c0, o)
			if o < 0 {
				return -1
			}
			// log.Printf("S: %q : %q", s[i+n0:i+n0+o], s[i+o+n0:])
			i += o + n0
			n0 = sz // The rune we matched on might not be the same size
		}
		var r1 rune
		var n1 int
		if s[i+n0] < utf8.RuneSelf {
			r1, n1 = rune(s[i+n0]), 1
		} else {
			r1, n1 = utf8.DecodeRuneInString(s[i+n0:])
		}
		// log.Printf("r1: %q c1: %q n1: %d size: %d", r1, c1, n1, utf8.RuneLen(c1))
		if r1 == c1 || unicode.ToLower(r1) == c1 {
			match, exhausted := hasPrefixUnicode(s[i:], substr)
			if match {
				return i
			}
			if exhausted {
				return -1
			}
		}
		// WARN WARN WARN: this is wrong
		// i += n0 + n1
		i += n0
		_ = n1
		fails++

		if fails >= 4+i>>4 && i < t {
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
	case n == len(s):
		// WARN: this is broken for multibyte strings
		if HasPrefix(s, substr) {
			return 0
		}
		return -1
	case n > len(s):
		if len(s) == 0 {
			return -1
		}
		// switch len(s) {
		// case 0:
		// 	return -1
		// case 1:
		// 	return hasPrefixUnicode(s, substr)
		// default:
		// }

		tn := utf8.RuneCountInString(substr)
		if tn == len(substr) {
			return -1 // ASCII
		}
		if tn > len(s) {
			return -1 // WARN: this breaks for 'ſ' (Latin s) and 'K' (Kelvin)
		}
		// At most the difference in encoded size between cases is 1 byte
		// per rune.
		//
		// TODO: combine this with the above check
		if len(s)+tn > n {
			return -1
		}

		// WARN: FIXME
		for range s {
			tn--
			if tn == 0 {
				break
			}
		}
		if tn > 0 {
			return -1
		}
		if tn == 0 {
			// panic("HERE")
			if match, _ := hasPrefixUnicode(s, substr); match {
				return 0
			}
			return -1
		}
		// panic("HERE")

		// WARN:
		// fallthrough
		// WARN: this is broken for variable width runes
		// return -1
	case n <= maxLen: // WARN: 32 is for arm64 (see: bytealg.MaxLen)
		if isASCII(substr) {
			if len(s) <= maxBruteForce {
				return bruteForceIndexASCII(s, substr)
			}
			return shortIndexASCII(s, substr)
		}

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
		return shortIndexUnicode(s, substr)
	}
	if isASCII(substr) {
		// return indexRabinKarp(s, substr)
		return indexASCII(s, substr)
	}
	return indexUnicode(s, substr)
}

func IndexByte(s string, c byte) int {
	n := strings.IndexByte(s, c)
	if n == 0 || !isAlpha(c) {
		return n
	}
	if n != -1 && n <= len(s)/2 && len(s) >= 256 {
		s = s[:n] // limit search space
	}
	c ^= ' ' // swap case
	o := strings.IndexByte(s, c)
	if n == -1 {
		return o
	}
	if o == -1 || n < o {
		return n
	}
	return o
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
		// WARN: Only works if s is all ASCII
		return IndexByte(s, byte(r)), 1
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

// TODO: the unicode symbols 'ſ' (Latin s) and 'K' (Kelvin) map to ASCII symbols.
var asciiFolds = map[rune][]rune{
	'ſ': {'ſ', 'S', 's'}, // Latin letter long s, an obsolete variant of s
	'K': {'K', 'K', 'k'}, // Kelvin sign
}
