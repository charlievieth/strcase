// Copyright 2023 Charlie Vieth. All rights reserved.
// Use of this source code is governed by the MIT license.

package strcase

import (
	"math/bits"
	"strings"
	"unicode/utf8"

	"github.com/charlievieth/strcase/internal/bytealg"
	"github.com/charlievieth/strcase/internal/tables"
)

const UnicodeVersion = tables.UnicodeVersion

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

// Compare returns an integer comparing two strings lexicographically
// ignoring case.
// The result will be 0 if a == b, -1 if a < b, and +1 if a > b.
func Compare(s, t string) int {
	n := len(s)
	if len(t) < n {
		n = len(t)
	}
	i := 0
	// Compare 8 bytes at a time: identical chunks are skipped outright and
	// all-ASCII chunks are compared case-insensitively using SWAR.
	for n-i >= 8 {
		x := le64(s[i:])
		y := le64(t[i:])
		if x == y {
			i += 8
			continue
		}
		if (x|y)&hi64 != 0 {
			break // non-ASCII: handled below
		}
		x = toLower8(x)
		y = toLower8(y)
		if x == y {
			i += 8
			continue
		}
		// The first (lowest addressed) mismatched byte decides the result.
		k := bits.TrailingZeros64(x^y) &^ 7
		if byte(x>>uint(k)) < byte(y>>uint(k)) {
			return -1
		}
		return 1
	}
	// Step back to a rune boundary: the chunked loop above can stop in the
	// middle of a multi-byte rune. Any continuation bytes preceding i were
	// only skipped as part of a byte-identical chunk, so the boundary found
	// here is shared by s and t.
	for i > 0 && i < n && s[i]&0xC0 == 0x80 {
		i--
	}
	for ; i < n; i++ {
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
	for len(s) != 0 {
		// If t is exhausted the strings are not equal.
		if len(t) == 0 {
			return 1
		}
		var sr, tr rune
		// Decode and case-fold the next rune of each string. Two byte
		// runes (the most common multi-byte runes and the bulk of the
		// runes that are case sensitive) are decoded inline since the
		// call to utf8.DecodeRuneInString is comparatively expensive.
		if c := s[0]; c < utf8.RuneSelf {
			sr, s = rune(_lower[c]), s[1:]
		} else if c < 0xE0 {
			if c >= 0xC2 && len(s) > 1 && s[1]&0xC0 == 0x80 {
				sr, s = tables.CaseFold(rune(c&0x1F)<<6|rune(s[1]&0x3F)), s[2:]
			} else {
				sr, s = utf8.RuneError, s[1:]
			}
		} else {
			r, size := utf8.DecodeRuneInString(s)
			sr, s = tables.CaseFold(r), s[size:]
		}
		if c := t[0]; c < utf8.RuneSelf {
			tr, t = rune(_lower[c]), t[1:]
		} else if c < 0xE0 {
			if c >= 0xC2 && len(t) > 1 && t[1]&0xC0 == 0x80 {
				tr, t = tables.CaseFold(rune(c&0x1F)<<6|rune(t[1]&0x3F)), t[2:]
			} else {
				tr, t = utf8.RuneError, t[1:]
			}
		} else {
			r, size := utf8.DecodeRuneInString(t)
			tr, t = tables.CaseFold(r), t[size:]
		}
		if sr == tr {
			continue
		}
		if sr < tr {
			return -1
		}
		return 1
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

const lo64 = 0x0101010101010101 // each byte is 0x01
const hi64 = 0x8080808080808080 // each byte is 0x80

// le64 returns the first 8 bytes of s as a little-endian uint64.
// The compiler lowers this to a single 8 byte load.
func le64(s string) uint64 {
	_ = s[7] // bounds check hint to compiler
	return uint64(s[0]) | uint64(s[1])<<8 | uint64(s[2])<<16 | uint64(s[3])<<24 |
		uint64(s[4])<<32 | uint64(s[5])<<40 | uint64(s[6])<<48 | uint64(s[7])<<56
}

// toLower8 converts the 8 ASCII bytes packed into x to lower case.
// x must not contain any bytes >= utf8.RuneSelf.
func toLower8(x uint64) uint64 {
	// A byte is in ['A', 'Z'] iff adding 0x80-'A' sets its high bit and
	// adding 0x80-'Z'-1 does not (no per-byte overflow since x is ASCII).
	m := ((x + lo64*(0x80-'A')) &^ (x + lo64*(0x80-'Z'-1))) & hi64
	return x | m>>2 // 0x80>>2 == 0x20
}

// containsKelvin returns true if string s contains rune 'K' (Kelvin).
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

	// Compare 8 bytes at a time: identical chunks are skipped outright and
	// all-ASCII chunks are compared case-insensitively using SWAR.
	i := 0
	m := len(s)
	if len(prefix) < m {
		m = len(prefix)
	}
	for m-i >= 8 {
		x := le64(s[i:])
		y := le64(prefix[i:])
		if x == y {
			i += 8
			continue
		}
		if (x|y)&hi64 != 0 {
			break // non-ASCII: handled below
		}
		x = toLower8(x)
		y = toLower8(y)
		if x == y {
			i += 8
			continue
		}
		return false, i+bits.TrailingZeros64(x^y)>>3 == len(s)-1
	}
	// Step back to a rune boundary: the chunked loop above can stop in the
	// middle of a multi-byte rune. Any continuation bytes preceding i were
	// only skipped as part of a byte-identical chunk, so the boundary found
	// here is shared by s and prefix.
	for i > 0 && i < m && s[i]&0xC0 == 0x80 {
		i--
	}
	// ASCII fast path
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
	for len(prefix) != 0 {
		// If s is exhausted the strings are not equal.
		if len(s) == 0 {
			return false, true
		}
		var sr, tr rune
		// Decode and case-fold the next rune of each string. Two byte
		// runes (the most common multi-byte runes and the bulk of the
		// runes that are case sensitive) are decoded inline since the
		// call to utf8.DecodeRuneInString is comparatively expensive.
		if c := prefix[0]; c < utf8.RuneSelf {
			tr, prefix = rune(_lower[c]), prefix[1:]
		} else if c < 0xE0 {
			if c >= 0xC2 && len(prefix) > 1 && prefix[1]&0xC0 == 0x80 {
				tr, prefix = tables.CaseFold(rune(c&0x1F)<<6|rune(prefix[1]&0x3F)), prefix[2:]
			} else {
				tr, prefix = utf8.RuneError, prefix[1:]
			}
		} else {
			r, size := utf8.DecodeRuneInString(prefix)
			tr, prefix = tables.CaseFold(r), prefix[size:]
		}
		if c := s[0]; c < utf8.RuneSelf {
			sr, s = rune(_lower[c]), s[1:]
		} else if c < 0xE0 {
			if c >= 0xC2 && len(s) > 1 && s[1]&0xC0 == 0x80 {
				sr, s = tables.CaseFold(rune(c&0x1F)<<6|rune(s[1]&0x3F)), s[2:]
			} else {
				sr, s = utf8.RuneError, s[1:]
			}
		} else {
			r, size := utf8.DecodeRuneInString(s)
			sr, s = tables.CaseFold(r), s[size:]
		}
		if tr == sr {
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

	// Compare 8 bytes at a time: identical chunks are skipped outright and
	// all-ASCII chunks are compared case-insensitively using SWAR.
	i := 0
	m := len(s)
	if len(prefix) < m {
		m = len(prefix)
	}
	for m-i >= 8 {
		x := le64(s[i:])
		y := le64(prefix[i:])
		if x == y {
			i += 8
			continue
		}
		if (x|y)&hi64 != 0 {
			break // non-ASCII: handled below
		}
		if toLower8(x) == toLower8(y) {
			i += 8
			continue
		}
		return s
	}
	// Step back to a rune boundary (see hasPrefixUnicode).
	for i > 0 && i < m && s[i]&0xC0 == 0x80 {
		i--
	}
	// ASCII fast path
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
	for len(prefix) != 0 {
		// If s is exhausted the strings are not equal.
		if len(s) == 0 {
			return ss
		}

		var sr, tr rune
		if c := prefix[0]; c < utf8.RuneSelf {
			tr, prefix = rune(_lower[c]), prefix[1:]
		} else if c < 0xE0 {
			if c >= 0xC2 && len(prefix) > 1 && prefix[1]&0xC0 == 0x80 {
				tr, prefix = tables.CaseFold(rune(c&0x1F)<<6|rune(prefix[1]&0x3F)), prefix[2:]
			} else {
				tr, prefix = utf8.RuneError, prefix[1:]
			}
		} else {
			r, size := utf8.DecodeRuneInString(prefix)
			tr, prefix = tables.CaseFold(r), prefix[size:]
		}
		if c := s[0]; c < utf8.RuneSelf {
			sr, s = rune(_lower[c]), s[1:]
		} else if c < 0xE0 {
			if c >= 0xC2 && len(s) > 1 && s[1]&0xC0 == 0x80 {
				sr, s = tables.CaseFold(rune(c&0x1F)<<6|rune(s[1]&0x3F)), s[2:]
			} else {
				sr, s = utf8.RuneError, s[1:]
			}
		} else {
			r, size := utf8.DecodeRuneInString(s)
			sr, s = tables.CaseFold(r), s[size:]
		}
		if tr == sr {
			continue
		}
		return ss
	}
	return s
}

// HasSuffix tests whether the string s ends with suffix.
func HasSuffix(s, suffix string) bool {
	ok, _ := hasSuffixUnicode(s, suffix)
	return ok
}

// hasSuffixUnicode returns if string s ends with suffix and the starting index
// of the suffix in s (so that we can trim it).
func hasSuffixUnicode(s, suffix string) (bool, int) {
	// TODO: if s and suffix have similar lengths it might be faster
	// to trim the left side of s then use HasPrefix since that saves
	// us from have to use utf8.DecodeLastRuneInString which is slow.

	// The max difference in encoded lengths between cases is 2 bytes for
	// [kK] (1 byte) and Kelvin 'K' (3 bytes).
	nt := len(suffix)
	ns := len(s)
	if nt == 0 {
		return true, ns
	}
	if ns*3 < nt || (ns*2 < nt && !containsKelvin(suffix)) {
		return false, 0
	}

	t := suffix
	li := ns
	lj := nt
	// Compare the trailing 8 bytes at a time: identical chunks are skipped
	// outright and all-ASCII chunks are compared case-insensitively using
	// SWAR.
	for li >= 8 && lj >= 8 {
		x := le64(s[li-8:])
		y := le64(t[lj-8:])
		if x == y {
			li -= 8
			lj -= 8
			continue
		}
		if (x|y)&hi64 != 0 {
			break // non-ASCII: handled below
		}
		if toLower8(x) == toLower8(y) {
			li -= 8
			lj -= 8
			continue
		}
		return false, 0
	}
	// Advance to a rune boundary: the chunked loop above can stop in the
	// middle of a multi-byte rune whose trailing bytes were skipped as part
	// of a byte-identical chunk (making the boundary shared by s and t).
	for li < ns && s[li]&0xC0 == 0x80 {
		li++
		lj++
	}
	{
		i := li - 1
		j := lj - 1
		for ; i >= 0 && j >= 0; i, j = i-1, j-1 {
			sr := s[i]
			tr := t[j]
			if (sr|tr)&utf8.RuneSelf != 0 {
				s = s[:i+1]
				t = t[:j+1]
				goto hasUnicode
			}
			if tr == sr || _lower[sr] == _lower[tr] {
				continue
			}
			return false, 0
		}
		return j == -1, i + 1
	}

hasUnicode:
	for len(s) != 0 && len(t) != 0 {
		var sr, tr rune
		// Decode and case-fold the last rune of each string. Two byte
		// runes are decoded inline since utf8.DecodeLastRuneInString is
		// comparatively expensive.
		if n := len(s) - 1; s[n] < utf8.RuneSelf {
			sr, s = rune(_lower[s[n]]), s[:n]
		} else if n >= 1 && s[n]&0xC0 == 0x80 && s[n-1] >= 0xC2 && s[n-1] < 0xE0 {
			sr, s = tables.CaseFold(rune(s[n-1]&0x1F)<<6|rune(s[n]&0x3F)), s[:n-1]
		} else {
			r, size := utf8.DecodeLastRuneInString(s)
			sr, s = tables.CaseFold(r), s[:len(s)-size]
		}
		if n := len(t) - 1; t[n] < utf8.RuneSelf {
			tr, t = rune(_lower[t[n]]), t[:n]
		} else if n >= 1 && t[n]&0xC0 == 0x80 && t[n-1] >= 0xC2 && t[n-1] < 0xE0 {
			tr, t = tables.CaseFold(rune(t[n-1]&0x1F)<<6|rune(t[n]&0x3F)), t[:n-1]
		} else {
			r, size := utf8.DecodeLastRuneInString(t)
			tr, t = tables.CaseFold(r), t[:len(t)-size]
		}
		if sr == tr {
			continue
		}
		return false, 0
	}

	return len(t) == 0, len(s)
}

// TrimSuffix returns s without the provided trailing suffix string.
// If s doesn't end with suffix, s is returned unchanged.
func TrimSuffix(s, suffix string) string {
	if match, i := hasSuffixUnicode(s, suffix); match {
		return s[:i]
	}
	return s
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
	folds0 := tables.FoldMapExcludingUpperLower(u0)
	folds1 := tables.FoldMapExcludingUpperLower(u1)
	hasFolds0 := folds0[0] != 0
	hasFolds1 := folds1[0] != 0
	needle := substr[sz0+sz1:]

	// 'İ' (U+0130) and 'ı' (U+0131) map to themselves in the _UpperLower
	// table (see internal/gen/gentables), so ToUpperLower handles them
	// correctly without any special casing here.
	var l0, l1 rune
	u0, l0, _ = tables.ToUpperLower(u0)
	u1, l1, _ = tables.ToUpperLower(u1)

	// Limit search space.
	t := len(s) - len(substr)/3 + 2
	if t > len(s) {
		t = len(s)
	}
	switch {
	case !hasFolds0 && u0 == l0 && !hasFolds1 && u1 == l1:
		i := 0
		// Fast check for the first two runes.
		if u0 != utf8.RuneError && u1 != utf8.RuneError {
			i = strings.Index(s, substr[:sz0+sz1])
			if i < 0 {
				return -1
			}
		}
		for i < t {
			var n0 int
			var r0 rune
			if c := s[i]; c < utf8.RuneSelf {
				r0, n0 = rune(c), 1
			} else if c < 0xE0 {
				if c >= 0xC2 && i+1 < len(s) && s[i+1]&0xC0 == 0x80 {
					r0, n0 = rune(c&0x1F)<<6|rune(s[i+1]&0x3F), 2
				} else {
					r0, n0 = utf8.RuneError, 1
				}
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
			if c := s[i+n0]; c < utf8.RuneSelf {
				r1, n1 = rune(c), 1
			} else if c < 0xE0 {
				if c >= 0xC2 && i+n0+1 < len(s) && s[i+n0+1]&0xC0 == 0x80 {
					r1, n1 = rune(c&0x1F)<<6|rune(s[i+n0+1]&0x3F), 2
				} else {
					r1, n1 = utf8.RuneError, 1
				}
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
			if c := s[i]; c < utf8.RuneSelf {
				r0, n0 = rune(c), 1
			} else if c < 0xE0 {
				if c >= 0xC2 && i+1 < len(s) && s[i+1]&0xC0 == 0x80 {
					r0, n0 = rune(c&0x1F)<<6|rune(s[i+1]&0x3F), 2
				} else {
					r0, n0 = utf8.RuneError, 1
				}
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
			if c := s[i+n0]; c < utf8.RuneSelf {
				r1, n1 = rune(c), 1
			} else if c < 0xE0 {
				if c >= 0xC2 && i+n0+1 < len(s) && s[i+n0+1]&0xC0 == 0x80 {
					r1, n1 = rune(c&0x1F)<<6|rune(s[i+n0+1]&0x3F), 2
				} else {
					r1, n1 = utf8.RuneError, 1
				}
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
			if c := s[i]; c < utf8.RuneSelf {
				r0, n0 = rune(c), 1
			} else if c < 0xE0 {
				if c >= 0xC2 && i+1 < len(s) && s[i+1]&0xC0 == 0x80 {
					r0, n0 = rune(c&0x1F)<<6|rune(s[i+1]&0x3F), 2
				} else {
					r0, n0 = utf8.RuneError, 1
				}
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
			if c := s[i+n0]; c < utf8.RuneSelf {
				r1, n1 = rune(c), 1
			} else if c < 0xE0 {
				if c >= 0xC2 && i+n0+1 < len(s) && s[i+n0+1]&0xC0 == 0x80 {
					r1, n1 = rune(c&0x1F)<<6|rune(s[i+n0+1]&0x3F), 2
				} else {
					r1, n1 = utf8.RuneError, 1
				}
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

// nonLetterASCII checks if the first 32 bytes of s consist only of
// non-letter ASCII characters. This is used to quickly check if we
// can use strings.Index.
func nonLetterASCII(s string) bool {
	i := 0
	if len(s) >= 8 {
		for ; len(s)-i >= 8; i += 8 {
			x := le64(s[i:])
			if x&hi64 != 0 {
				return false
			}
			// A byte of y is in ['a', 'z'] iff the corresponding byte of
			// x is an ASCII letter (only letters fold into ['a', 'z']
			// under |0x20).
			y := x | lo64*0x20
			if ((y+lo64*(0x80-'a'))&^(y+lo64*(0x80-'z'-1)))&hi64 != 0 {
				return false
			}
		}
	}
	for ; i < len(s); i++ {
		// NB: this is faster than using a lookup table
		c := s[i] | ' ' // simplify check for alpha
		if c&utf8.RuneSelf != 0 || 'a' <= c && c <= 'z' {
			return false
		}
	}
	return true
}

// TODO: check substr for any folds - this should almost always be faster
// than our current search - especially since we end up having to scan it
// multiple times. Maybe: check first rune (and maybe second), then check
// if substr has any folds.
//
// That said this library is meant for dealing with text and this might
// only with languages that don't have any/few folds.
//
// One thing we could do is use the longest prefix that does not have any
// folds and use the fast string search for that.

// Index returns the index of the first instance of substr in s, or -1 if
// substr is not present in s. Both s and substr are interpreted as UTF-8
// strings and simple Unicode case-folding is used to check for equality.
// All invalid UTF-8 encoded runes are considered equal - this matches the
// behavior of [strings.EqualFold].
//
// If substr is a single byte and an invalid UTF-8 sequence (0x80-0xFF) the
// index of the first invalid rune is returned. This matches the behavior of
// [IndexRune], but does not match the behavior of [IndexByte], which will
// return the index of bytes in the range of 0x80-0xFF.
// This is required because we guarantee that the result of Index would equal
// if compared with [strings.EqualFold].
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
	case n == 1 && r != utf8.RuneError:
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
		// WARN:
		//  * this does not take non-folding runes into account
		//  * figure out when this negatively impacts performance
		//  * consider skipping if s is small relative to substr
		//  * check if s contains the first char of substr first ???
		//  * check after the maxBruteForce check ???
		//
		// Fast path if the sub-string is all non-alpha ASCII chars
		//
		// NB: This only works because len(substr) <= 32
		//
		// TODO:
		//  * Are we just gaming benchmarks here and need a quicker
		//    cutover to Rabin-Karp?
		//  * We should skip this check if s is small
		//
		if bytealg.NativeIndex && n <= 32 && nonLetterASCII(substr) {
			return bytealg.IndexString(s, substr)
		}
		// TODO: tune this
		if len(s) <= maxBruteForce {
			return bruteForceIndexUnicode(s, substr)
		}
		// fallthrough
	}

	// All-ASCII substr fast path: use a byte-wise SIMD accelerated search
	// when Unicode aware matching is not required. Unicode aware matching
	// is only required if substr contains one of [KkSs], which are the only
	// ASCII characters that multi-byte runes (Kelvin 'K' and Latin small
	// letter long s 'ſ') can fold to, and s contains one of those runes.
	// NB: the substr[0] check cheaply excludes needles that start with a
	// multi-byte rune (an all-ASCII substr must start with an ASCII char).
	if bytealg.NativeIndexPair && substr[0] < utf8.RuneSelf {
		isASCII := true
		hasK, hasS := false, false
		if n <= 64 {
			for i := 0; i < n; i++ {
				c := substr[i]
				if c >= utf8.RuneSelf {
					isASCII = false
					break
				}
				switch c | ' ' {
				case 'k':
					hasK = true
				case 's':
					hasS = true
				}
			}
		} else if bytealg.IndexNonASCII(substr) < 0 {
			hasK = bytealg.IndexByteString(substr, 'k') >= 0
			hasS = bytealg.IndexByteString(substr, 's') >= 0
		} else {
			isASCII = false
		}
		if isASCII {
			i := indexASCIIFold(s, substr)
			if !hasK && !hasS {
				return i
			}
			// A missed match involving Kelvin or long s must start before
			// i and its window in s is less than 3*n bytes long (each of
			// the n ASCII bytes of substr matches at most 3 bytes of s),
			// so only s[:i+3*n] needs to be checked for those runes.
			ss := s
			if i >= 0 && i+3*n < len(s) {
				ss = s[:i+3*n]
			}
			if bytealg.IndexNonASCII(ss) < 0 {
				return i // ss is all ASCII: it cannot contain those runes
			}
			if (!hasK || indexRuneCase(ss, '\u212A') < 0) &&
				(!hasS || indexRuneCase(ss, '\u017F') < 0) {
				return i
			}
			// s contains Kelvin or long s: fall through to the generic
			// Unicode aware search.
		}
	}

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

	// Use Rabin-Karp if either of the first two runes are invalid
	// this is slower but simplifies the logic below.
	if u0 == utf8.RuneError || u1 == utf8.RuneError {
		return indexRabinKarpUnicode(s, substr)
	}

	// hasFolds{0,1} should be rare so consider optimizing
	// the no folds case
	folds0 := tables.FoldMapExcludingUpperLower(u0)
	folds1 := tables.FoldMapExcludingUpperLower(u1)
	needle := substr[sz0+sz1:]

	// TODO: we can possibly get rid of the ToUpperLower function
	// and table since it's not always on the critical path and it
	// adds a a lot to the size of this package

	// 'İ' (U+0130) and 'ı' (U+0131) map to themselves in the _UpperLower
	// table (see internal/gen/gentables), so ToUpperLower handles them
	// correctly without any special casing here.
	var l0, l1 rune
	u0, l0, _ = tables.ToUpperLower(u0)
	u1, l1, _ = tables.ToUpperLower(u1)

	fails := 0
	// TODO: see if we can stop sooner.
	t := len(s) - len(substr)/3 + 1
	if t > len(s) {
		t = len(s)
	}
	for i := 0; i < t; {
		var r0 rune
		var n0 int
		if c := s[i]; c < utf8.RuneSelf {
			r0, n0 = rune(c), 1
		} else if c < 0xE0 {
			if c >= 0xC2 && i+1 < len(s) && s[i+1]&0xC0 == 0x80 {
				r0, n0 = rune(c&0x1F)<<6|rune(s[i+1]&0x3F), 2
			} else {
				r0, n0 = utf8.RuneError, 1
			}
		} else {
			r0, n0 = utf8.DecodeRuneInString(s[i:])
		}

		if r0 != u0 && r0 != l0 && (folds0[0] == 0 || (r0 != folds0[0] && r0 != folds0[1])) {
			var o, sz int
			if folds0[0] == 0 {
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
		if c := s[i+n0]; c < utf8.RuneSelf {
			r1, n1 = rune(c), 1
		} else if c < 0xE0 {
			if c >= 0xC2 && i+n0+1 < len(s) && s[i+n0+1]&0xC0 == 0x80 {
				r1, n1 = rune(c&0x1F)<<6|rune(s[i+n0+1]&0x3F), 2
			} else {
				r1, n1 = utf8.RuneError, 1
			}
		} else {
			r1, n1 = utf8.DecodeRuneInString(s[i+n0:])
		}

		if r1 == u1 || r1 == l1 || (folds1[0] != 0 && (r1 == folds1[0] || r1 == folds1[1])) {
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

// LastIndex returns the index of the last instance of substr in s, or -1 if
// substr is not present in s.
func LastIndex(s, substr string) int {
	n := len(substr)
	var r rune
	var size int
	if n > 0 {
		if substr[n-1] < utf8.RuneSelf {
			r, size = rune(substr[n-1]), 1
		} else {
			r, size = utf8.DecodeRuneInString(substr)
		}
	}
	switch {
	case n == 0:
		return len(s)
	// case n == 1 && r != utf8.RuneError:
	case n == 1:
		return LastIndexByte(s, substr[0])
	case n == size:
		// TODO: indexRabinKarpRevUnicode might be faster here
		return lastIndexRune(s, r)
	case n >= len(s):
		if n > len(s)*3 {
			return -1
		}
		if n > len(s)*2 && !containsKelvin(substr) {
			return -1
		}
		// fallthrough
	}

	// All-ASCII substr fast path: mirrors the equivalent fast path in Index
	// (see the comments there for details).
	// NB: the substr[0] check cheaply excludes needles that start with a
	// multi-byte rune (an all-ASCII substr must start with an ASCII char).
	if bytealg.NativeIndexPair && substr[0] < utf8.RuneSelf {
		isASCII := true
		hasK, hasS := false, false
		if n <= 64 {
			for i := 0; i < n; i++ {
				c := substr[i]
				if c >= utf8.RuneSelf {
					isASCII = false
					break
				}
				switch c | ' ' {
				case 'k':
					hasK = true
				case 's':
					hasS = true
				}
			}
		} else if bytealg.IndexNonASCII(substr) < 0 {
			hasK = bytealg.IndexByteString(substr, 'k') >= 0
			hasS = bytealg.IndexByteString(substr, 's') >= 0
		} else {
			isASCII = false
		}
		if isASCII {
			i := lastIndexASCIIFold(s, substr)
			if !hasK && !hasS {
				return i
			}
			// A missed match involving Kelvin or long s must start after
			// i so only s[i+1:] needs to be checked for those runes.
			ss := s
			if i >= 0 {
				ss = s[i+1:]
			}
			if bytealg.IndexNonASCII(ss) < 0 {
				return i // ss is all ASCII: it cannot contain those runes
			}
			if (!hasK || indexRuneCase(ss, '\u212A') < 0) &&
				(!hasS || indexRuneCase(ss, '\u017F') < 0) {
				return i
			}
			// s contains Kelvin or long s: fall through to the generic
			// Unicode aware search.
		}
	}
	return indexRabinKarpRevUnicode(s, substr)
}

// IndexByte returns the index of the first instance of c (ignoring case) in s,
// or -1 if c is not present in s. Matching is case-insensitive and Unicode
// simple folding is used which means that ASCII bytes 'K' and 'k' match Kelvin
// 'K', and ASCII bytes 'S' and 's' match 'ſ' (Latin small letter long S).
// Therefore, string s may be scanned twice when c is in [KkSs]
// (because the optimized assembly is ASCII only).
//
// On amd64 and arm64 this is only ~20-25% slower than strings.IndexByte for
// small strings (<4M) and ~6% slower for larger strings.
// The slowdown for small strings is due to additional checks and function call
// overheard.
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
	if o := indexRuneCase(s, r); n == -1 || (o != -1 && o < n) {
		return o, sz
	}
	return n, 1
}

// lastIndexByteASCII returns the index of the last instance of c in s, or -1
// if c is not present in s. ASCII letters are matched case-insensitively.
// The caller must handle c being one of [KkSs] (which can also be matched by
// the non-ASCII runes Kelvin and Latin small letter long s) if s may contain
// non-ASCII characters.
func lastIndexByteASCII(s string, c byte) int {
	// Fold ASCII letters to lower case. Only the two cases of an ASCII
	// letter can fold to c, so this cannot introduce false positives.
	var fold uint64
	if isAlpha(c) {
		c |= ' '
		fold = lo64 * 0x20
	}
	cc := lo64 * uint64(c)
	i := len(s)
	// Search 8 bytes at a time using SWAR: locate zero bytes in the XOR
	// with c using an exact zero-byte mask.
	for i >= 8 {
		y := (le64(s[i-8:]) | fold) ^ cc
		if m := ^((y | hi64) - lo64) &^ y & hi64; m != 0 {
			return i - 1 - bits.LeadingZeros64(m)/8
		}
		i -= 8
	}
	for i--; i >= 0; i-- {
		if s[i]|byte(fold) == c {
			return i
		}
	}
	return -1
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
		return lastIndexByteASCII(s, c)
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
		if folds := tables.FoldMap(r); folds != nil {
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
		} else if u, l, ok := tables.ToUpperLower(r); ok {
			return indexRune2(s, l, u)
		}
		return indexRuneCase(s, r), utf8.RuneLen(r)
	}
}

// TODO: consider creating a version of this function for when we've already
// decoded the rune and know that it is valid and not-ASCII.
//
// indexRuneCase is a *case-sensitive* version of strings.IndexRune that is
// generally faster for Unicode characters since it searches for the rune's
// second byte, which is generally more unique, instead of the first byte,
// like strings.IndexRune.
func indexRuneCase(s string, r rune) int {
	// TODO:
	//   * remove if my PR is ever merged
	//   * consider searching for the first byte if it is not one of:
	//     240, 243, or 244 (which are the first byte of ~78% of multi-byte
	//     Unicode characters).
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
		// Inlined version of utf8.EncodeRune. This is one of our hottest
		// functions and EncodeRune or string(r) add significant overhead
		// when the string being searched is small.
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
		// Search for r using the last byte of its UTF-8 encoded form
		// since it is more unique than the first byte. This 4-5x faster
		// when all the text is Unicode.
		var fails, i int
		switch n {
		case 2:
			i = 1
			for i < len(s) {
				if s[i] != c1 {
					o := strings.IndexByte(s[i+1:], c1)
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
				// Switch to bytealg.IndexString, if implemented, when the needle
				// is small and IndexByte produces too many false positives.
				if (bytealg.NativeIndex && fails > bytealg.Cutover(i) && i < len(s)) ||
					(!bytealg.NativeIndex && fails >= 4+i>>4 && i < len(s)) {
					if bytealg.NativeIndex {
						// NB: Impossible to match in the middle of the two byte
						// rune so we don't need to search from: s[i-:].
						if j := bytealg.IndexString(s[i:], string(r)); j != -1 {
							return i + j
						}
					} else {
						// Elided on platforms with a native IndexString.
						// This is faster than calling strings.Index or
						// using Rabin-Karp.
						for ; i < len(s); i++ {
							if s[i] == c1 && s[i-1] == c0 {
								return i - 1
							}
						}
					}
					return -1
				}
			}
		case 3:
			i = 2
			for i < len(s) {
				if s[i] != c2 {
					o := strings.IndexByte(s[i+1:], c2)
					if o < 0 {
						return -1
					}
					i += o + 1
				}
				if s[i-2] == c0 && s[i-1] == c1 {
					return i - 2
				}
				fails++
				i++
				if (bytealg.NativeIndex && fails > bytealg.Cutover(i) && i < len(s)) ||
					(!bytealg.NativeIndex && fails >= 4+i>>4 && i < len(s)) {
					if bytealg.NativeIndex {
						// We might be in the middle of a rune so search: s[i-2:]
						if j := bytealg.IndexString(s[i-2:], string(r)); j != -1 {
							return i + j - 2
						}
					} else {
						for ; i < len(s); i++ {
							if s[i] == c2 && s[i-1] == c1 && s[i-2] == c0 {
								return i - 2
							}
						}
					}
					return -1
				}
			}
		case 4:
			i = 3
			for i < len(s) {
				if s[i] != c3 {
					o := strings.IndexByte(s[i+1:], c3)
					if o < 0 {
						return -1
					}
					i += o + 1
				}
				if s[i-3] == c0 && s[i-2] == c1 && s[i-1] == c2 {
					return i - 3
				}
				fails++
				i++
				if (bytealg.NativeIndex && fails > bytealg.Cutover(i) && i < len(s)) ||
					(!bytealg.NativeIndex && fails >= 4+i>>4 && i < len(s)) {
					if bytealg.NativeIndex {
						// We might be in the middle of a rune so search: s[i-3:]
						if j := bytealg.IndexString(s[i-3:], string(r)); j != -1 {
							return i + j - 3
						}
					} else {
						for ; i < len(s); i++ {
							if s[i] == c3 && s[i-1] == c2 && s[i-2] == c1 && s[i-3] == c0 {
								return i - 3
							}
						}
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
		if folds := tables.FoldMap(r); folds != nil {
			for i := len(s); i > 0; {
				var sr rune
				if c := s[i-1]; c < utf8.RuneSelf {
					sr = rune(c)
					i--
				} else if i >= 2 && c&0xC0 == 0x80 && s[i-2] >= 0xC2 && s[i-2] < 0xE0 {
					sr = rune(s[i-2]&0x1F)<<6 | rune(c&0x3F)
					i -= 2
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
			u, l, _ := tables.ToUpperLower(r)
			if u == l {
				rs := string(r)
				last := len(rs) - 1
				// Search for the last byte of r (SWAR accelerated) and
				// check the preceding bytes on each candidate match.
				for j := len(s); j > last; {
					i := lastIndexByteASCII(s[:j], rs[last])
					if i < last {
						break
					}
					if s[i-last:i+1] == rs {
						return i - last
					}
					j = i
				}
			} else {
				for i := len(s); i > 0; {
					var sr rune
					if c := s[i-1]; c < utf8.RuneSelf {
						sr = rune(c)
						i--
					} else if i >= 2 && c&0xC0 == 0x80 && s[i-2] >= 0xC2 && s[i-2] < 0xE0 {
						sr = rune(s[i-2]&0x1F)<<6 | rune(c&0x3F)
						i -= 2
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
		}
		return -1
	}
}

// primeRK is the prime base used in Rabin-Karp algorithm.
const primeRK = 16777619

// equalFoldASCII reports whether s and t, which must be the same length,
// are equal with ASCII letters compared case-insensitively and all other
// bytes (including non-ASCII bytes) compared exactly.
func equalFoldASCII(s, t string) bool {
	i := 0
	for ; len(s)-i >= 8; i += 8 {
		x, y := le64(s[i:]), le64(t[i:])
		if x == y {
			continue
		}
		if (x|y)&hi64 != 0 {
			// Non-ASCII bytes: compare bytewise (toLower8 requires ASCII).
			for j := i; j < i+8; j++ {
				if _lower[s[j]] != _lower[t[j]] {
					return false
				}
			}
			continue
		}
		if toLower8(x) != toLower8(y) {
			return false
		}
	}
	for ; i < len(s); i++ {
		if _lower[s[i]] != _lower[t[i]] {
			return false
		}
	}
	return true
}

// hashStrASCII returns the hash of the lower case form of sep and the
// appropriate multiplicative factor for use in Rabin-Karp algorithm.
func hashStrASCII(sep string) (uint32, uint32) {
	// Process 4 bytes at a time to shorten the multiply dependency chain.
	p := uint32(primeRK)
	p2 := p * p
	p3 := p2 * p
	p4 := p2 * p2
	hash := uint32(0)
	i := 0
	for ; i+4 <= len(sep); i += 4 {
		hash = hash*p4 +
			uint32(_lower[sep[i]])*p3 +
			uint32(_lower[sep[i+1]])*p2 +
			uint32(_lower[sep[i+2]])*p +
			uint32(_lower[sep[i+3]])
	}
	for ; i < len(sep); i++ {
		hash = hash*p + uint32(_lower[sep[i]])
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

// hashStrRevASCII returns the hash of the lower case form of the reverse of
// sep and the appropriate multiplicative factor for use in Rabin-Karp
// algorithm.
func hashStrRevASCII(sep string) (uint32, uint32) {
	// Process 4 bytes at a time to shorten the multiply dependency chain.
	p := uint32(primeRK)
	p2 := p * p
	p3 := p2 * p
	p4 := p2 * p2
	hash := uint32(0)
	i := len(sep) - 1
	for ; i >= 3; i -= 4 {
		hash = hash*p4 +
			uint32(_lower[sep[i]])*p3 +
			uint32(_lower[sep[i-1]])*p2 +
			uint32(_lower[sep[i-2]])*p +
			uint32(_lower[sep[i-3]])
	}
	for ; i >= 0; i-- {
		hash = hash*p + uint32(_lower[sep[i]])
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

// indexASCIIFold returns the index of the first occurrence of substr in s
// where substr consists only of ASCII characters and is matched ASCII
// case-insensitively (byte-wise). The caller must ensure that a byte-wise
// ASCII search is sufficient: that is s must not contain the multi-byte
// runes Kelvin 'K' or Latin small letter long s 'ſ' if substr contains any
// of [KkSs] (no other multi-byte runes fold to ASCII characters).
func indexASCIIFold(s, substr string) int {
	return indexASCIIFoldPair(s, substr, bytealg.PackPair(substr[0], substr[1]))
}

// indexASCIIFoldPair is indexASCIIFold with the packed pair of the first two
// bytes of substr provided by the caller (so that repeated searches for the
// same substr do not need to recompute it).
func indexASCIIFoldPair(s, substr string, pair uint32) int {
	n := len(substr)
	t := len(s) - n // last valid match start (inclusive)
	if t < 0 {
		return -1
	}
	fails := 0
	i := 0
	for i <= t {
		// Search for the first two bytes of substr (case-insensitively).
		// NB: limit the search space to valid match starts (a pair match
		// requires one byte following it so include one extra byte).
		j := bytealg.IndexPairFold(s[i:t+2], pair)
		if j < 0 {
			return -1
		}
		i += j
		if equalFoldASCII(s[i:i+n], substr) {
			return i
		}
		fails++
		i++
		// Switch to Rabin-Karp if the pair scan produces too many false
		// candidates (mirrors the cutover used by bytes.Index).
		if fails >= 4+i>>4 && i <= t {
			if j := indexRabinKarpASCII(s[i:], substr); j >= 0 {
				return i + j
			}
			return -1
		}
	}
	return -1
}

// lastIndexASCIIFold is the reverse version of indexASCIIFold: it returns
// the index of the last occurrence of substr in s (see indexASCIIFold for
// the conditions the caller must ensure).
func lastIndexASCIIFold(s, substr string) int {
	n := len(substr)
	t := len(s) - n // last valid match start (inclusive)
	if t < 0 {
		return -1
	}
	pair := bytealg.PackPair(substr[0], substr[1])
	fails := 0
	j := t // maximum candidate start
	for j >= 0 {
		k := bytealg.LastIndexPairFold(s[:j+2], pair)
		if k < 0 {
			return -1
		}
		if equalFoldASCII(s[k:k+n], substr) {
			return k
		}
		fails++
		j = k - 1
		// Switch to Rabin-Karp if the pair scan produces too many false
		// candidates.
		if fails >= 4+(len(s)-j)>>4 && j >= 0 {
			return indexRabinKarpRevASCII(s[:j+n], substr)
		}
	}
	return -1
}

// indexRabinKarpASCII uses the Rabin-Karp search algorithm to return the index
// of the first occurrence of substr in s, or -1 if not present. Both s and
// substr must consist only of ASCII characters (in which case Unicode simple
// folding is equivalent to ASCII case-insensitive comparison since no
// multi-byte rune folds to an ASCII character... except for Kelvin and long s,
// which cannot occur in an all ASCII string).
func indexRabinKarpASCII(s, substr string) int {
	n := len(substr)
	if len(s) < n {
		return -1
	}
	hashss, pow := hashStrASCII(substr)
	var h uint32
	for i := 0; i < n; i++ {
		h = h*primeRK + uint32(_lower[s[i]])
	}
	if h == hashss && equalFoldASCII(s[:n], substr) {
		return 0
	}
	// Roll the hash 2 bytes at a time to shorten the multiply dependency
	// chain (the intermediate hash h1 is computed off the critical path).
	p := uint32(primeRK)
	p2 := p * p
	i := n
	for ; i+2 <= len(s); i += 2 {
		x0 := uint32(_lower[s[i]])
		x1 := uint32(_lower[s[i+1]])
		y0 := uint32(_lower[s[i-n]])
		y1 := uint32(_lower[s[i-n+1]])
		h1 := h*p + x0 - pow*y0
		h = h*p2 + x0*p + x1 - pow*(y0*p+y1)
		if h1 == hashss && equalFoldASCII(s[i+1-n:i+1], substr) {
			return i + 1 - n
		}
		if h == hashss && equalFoldASCII(s[i+2-n:i+2], substr) {
			return i + 2 - n
		}
	}
	for ; i < len(s); i++ {
		h = h*p + uint32(_lower[s[i]]) - pow*uint32(_lower[s[i-n]])
		if h == hashss && equalFoldASCII(s[i+1-n:i+1], substr) {
			return i + 1 - n
		}
	}
	return -1
}

// indexRabinKarpRevASCII uses the Rabin-Karp search algorithm to return the
// index of the last occurrence of substr in s, or -1 if not present. Both s
// and substr must consist only of ASCII characters.
func indexRabinKarpRevASCII(s, substr string) int {
	n := len(substr)
	if len(s) < n {
		return -1
	}
	hashss, pow := hashStrRevASCII(substr)
	var h uint32
	last := len(s) - n
	for i := len(s) - 1; i >= last; i-- {
		h = h*primeRK + uint32(_lower[s[i]])
	}
	if h == hashss && equalFoldASCII(s[last:], substr) {
		return last
	}
	// Roll the hash 2 bytes at a time to shorten the multiply dependency
	// chain (the intermediate hash h1 is computed off the critical path).
	p := uint32(primeRK)
	p2 := p * p
	i := last - 1
	for ; i >= 1; i -= 2 {
		x0 := uint32(_lower[s[i]])
		x1 := uint32(_lower[s[i-1]])
		y0 := uint32(_lower[s[i+n]])
		y1 := uint32(_lower[s[i-1+n]])
		h1 := h*p + x0 - pow*y0
		h = h*p2 + x0*p + x1 - pow*(y0*p+y1)
		if h1 == hashss && equalFoldASCII(s[i:i+n], substr) {
			return i
		}
		if h == hashss && equalFoldASCII(s[i-1:i-1+n], substr) {
			return i - 1
		}
	}
	if i == 0 {
		h = h*p + uint32(_lower[s[0]]) - pow*uint32(_lower[s[n]])
		if h == hashss && equalFoldASCII(s[:n], substr) {
			return 0
		}
	}
	return -1
}

// hashStrUnicode returns the hash and the appropriate multiplicative
// factor for use in Rabin-Karp algorithm, and the number of runes
// in sep.
func hashStrUnicode(sep string) (uint32, uint32, int) {
	hash := uint32(0)
	n := 0
	for i := 0; i < len(sep); {
		var r rune
		if c := sep[i]; c < utf8.RuneSelf {
			r = rune(_lower[c])
			i++
		} else if c < 0xE0 {
			if c >= 0xC2 && i+1 < len(sep) && sep[i+1]&0xC0 == 0x80 {
				r = tables.CaseFold(rune(c&0x1F)<<6 | rune(sep[i+1]&0x3F))
				i += 2
			} else {
				r = utf8.RuneError
				i++
			}
		} else {
			rr, size := utf8.DecodeRuneInString(sep[i:])
			r = tables.CaseFold(rr)
			i += size
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
		if c := sep[i-1]; c < utf8.RuneSelf {
			r, size = rune(_lower[c]), 1
		} else if i >= 2 && c&0xC0 == 0x80 && sep[i-2] >= 0xC2 && sep[i-2] < 0xE0 {
			r, size = tables.CaseFold(rune(sep[i-2]&0x1F)<<6|rune(c&0x3F)), 2
		} else {
			r, size = utf8.DecodeLastRuneInString(sep[:i])
			r = tables.CaseFold(r)
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
	// Use the much faster ASCII version if possible. Scanning both strings
	// for non-ASCII characters is nearly free (SIMD) relative to the cost
	// of the search itself.
	if bytealg.IndexNonASCII(substr) < 0 && bytealg.IndexNonASCII(s) < 0 {
		return indexRabinKarpRevASCII(s, substr)
	}
	// Reverse Rabin-Karp search
	hashss, pow, n := hashStrRevUnicode(substr)
	var h uint32
	i := len(s)
	for i > 0 {
		var r rune
		var size int
		if c := s[i-1]; c < utf8.RuneSelf {
			r, size = rune(_lower[c]), 1
		} else if i >= 2 && c&0xC0 == 0x80 && s[i-2] >= 0xC2 && s[i-2] < 0xE0 {
			r, size = tables.CaseFold(rune(s[i-2]&0x1F)<<6|rune(c&0x3F)), 2
		} else {
			r, size = utf8.DecodeLastRuneInString(s[:i])
			r = tables.CaseFold(r)
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
		if c := s[i-1]; c < utf8.RuneSelf {
			r0, n0 = rune(_lower[c]), 1
		} else if i >= 2 && c&0xC0 == 0x80 && s[i-2] >= 0xC2 && s[i-2] < 0xE0 {
			r0, n0 = tables.CaseFold(rune(s[i-2]&0x1F)<<6|rune(c&0x3F)), 2
		} else {
			r0, n0 = utf8.DecodeLastRuneInString(s[:i])
			r0 = tables.CaseFold(r0)
		}
		var r1 rune
		var n1 int
		if c := s[j-1]; c < utf8.RuneSelf {
			r1, n1 = rune(_lower[c]), 1
		} else if j >= 2 && c&0xC0 == 0x80 && s[j-2] >= 0xC2 && s[j-2] < 0xE0 {
			r1, n1 = tables.CaseFold(rune(s[j-2]&0x1F)<<6|rune(c&0x3F)), 2
		} else {
			r1, n1 = utf8.DecodeLastRuneInString(s[:j])
			r1 = tables.CaseFold(r1)
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
	// Use the much faster ASCII version if possible. Scanning both strings
	// for non-ASCII characters is nearly free (SIMD) relative to the cost
	// of the search itself.
	if bytealg.IndexNonASCII(substr) < 0 && bytealg.IndexNonASCII(s) < 0 {
		return indexRabinKarpASCII(s, substr)
	}
	// Rabin-Karp search
	hashss, pow, n := hashStrUnicode(substr)
	var h uint32
	sz := 0 // byte size of 'n' runes
	for sz < len(s) {
		var r rune
		if c := s[sz]; c < utf8.RuneSelf {
			r = rune(_lower[c])
			sz++
		} else if c < 0xE0 {
			if c >= 0xC2 && sz+1 < len(s) && s[sz+1]&0xC0 == 0x80 {
				r = tables.CaseFold(rune(c&0x1F)<<6 | rune(s[sz+1]&0x3F))
				sz += 2
			} else {
				r = utf8.RuneError
				sz++
			}
		} else {
			rr, size := utf8.DecodeRuneInString(s[sz:])
			r = tables.CaseFold(rr)
			sz += size
		}
		h = h*primeRK + uint32(r)
		n--
		if n == 0 {
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
		if c := s[j]; c < utf8.RuneSelf {
			s0, n0 = rune(_lower[c]), 1
		} else if c < 0xE0 {
			if c >= 0xC2 && j+1 < len(s) && s[j+1]&0xC0 == 0x80 {
				s0, n0 = tables.CaseFold(rune(c&0x1F)<<6|rune(s[j+1]&0x3F)), 2
			} else {
				s0, n0 = utf8.RuneError, 1
			}
		} else {
			s0, n0 = utf8.DecodeRuneInString(s[j:])
			s0 = tables.CaseFold(s0)
		}
		if c := s[i]; c < utf8.RuneSelf {
			s1, n1 = rune(_lower[c]), 1
		} else if c < 0xE0 {
			if c >= 0xC2 && i+1 < len(s) && s[i+1]&0xC0 == 0x80 {
				s1, n1 = tables.CaseFold(rune(c&0x1F)<<6|rune(s[i+1]&0x3F)), 2
			} else {
				s1, n1 = utf8.RuneError, 1
			}
		} else {
			s1, n1 = utf8.DecodeRuneInString(s[i:])
			s1 = tables.CaseFold(s1)
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

func countRune(s string, r rune) (n int) {
	sz := utf8.RuneLen(r)
	for {
		i := indexRuneCase(s, r)
		if i == -1 {
			return n
		}
		n++
		s = s[i+sz:]
	}
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
			n += countRune(s, 'K')
		case 'S', 's':
			n += countRune(s, 'ſ')
		}
		return n
	}
	// Fast path for all-ASCII substrings: matches are exactly len(substr)
	// bytes long so the Unicode aware checks (and rune counting) performed
	// by Index can be hoisted out of the loop. This requires that s does
	// not contain Kelvin 'K' or Latin small letter long s 'ſ' (the only
	// multi-byte runes that fold to ASCII) if substr contains [KkSs].
	if bytealg.NativeIndexPair && len(substr) >= 2 && substr[0] < utf8.RuneSelf {
		isASCII := true
		hasK, hasS := false, false
		for i := 0; i < len(substr); i++ {
			c := substr[i]
			if c >= utf8.RuneSelf {
				isASCII = false
				break
			}
			switch c | ' ' {
			case 'k':
				hasK = true
			case 's':
				hasS = true
			}
		}
		if isASCII && ((!hasK && !hasS) || bytealg.IndexNonASCII(s) < 0 ||
			((!hasK || indexRuneCase(s, '\u212A') < 0) &&
				(!hasS || indexRuneCase(s, '\u017F') < 0))) {
			pair := bytealg.PackPair(substr[0], substr[1])
			n := 0
			for {
				i := indexASCIIFoldPair(s, substr, pair)
				if i == -1 {
					return n
				}
				n++
				s = s[i+len(substr):]
			}
		}
	}
	n := 0
	runeCount := -1 // Lazily calculated after the first match
	for {
		i := Index(s, substr)
		if i == -1 {
			return n
		}
		n++
		if runeCount < 0 {
			runeCount = utf8.RuneCountInString(substr)
		}
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
	if len(chars) == 0 {
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
			// For a small number of chars it is faster to scan s once per
			// char with the SIMD accelerated (and ASCII case-insensitive)
			// IndexByteString than to check each byte of s against the set.
			//
			// NB: makeASCIISet returning true means that either chars does
			// not contain [KkSs] or s is all ASCII, so an ASCII only search
			// is safe here.
			if len(chars) <= 4 && len(s) >= 32 {
				n := -1
				for i := 0; i < len(chars); i++ {
					o := bytealg.IndexByteString(s, chars[i])
					if o != -1 {
						n = o
						if n == 0 {
							break
						}
						s = s[:n] // limit the search space
					}
				}
				return n
			}
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
	if len(chars) == 0 {
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
			// For a small number of chars it is faster to scan s once per
			// char with the SWAR accelerated (and ASCII case-insensitive)
			// lastIndexByteASCII than to check each byte of s against the
			// set.
			//
			// NB: makeASCIISet returning true means that either chars does
			// not contain [KkSs] or s is all ASCII, so an ASCII only search
			// is safe here.
			if len(chars) <= 4 && len(s) >= 32 {
				n := -1
				for i := 0; i < len(chars); i++ {
					o := lastIndexByteASCII(s[n+1:], chars[i])
					if o != -1 {
						n += 1 + o // convert to an index into s
					}
				}
				return n
			}
			for i := len(s) - 1; i >= 0; i-- {
				if as.contains(s[i]) {
					return i
				}
			}
			return -1
		}
	}
	if len(chars) <= 32 && len(s) > len(chars)*2 {
		// Avoid the overhead of repeatedly decoding s and searching chars
		// if s is significantly longer than chars and the number of chars
		// is small (mirrors IndexAny).
		//
		// NB: each search must cover all of s (no trimming) since a match
		// for one char does not tell us the size of the matched rune, so
		// we cannot safely re-slice s at a rune boundary past it.
		n := -1
		for _, r := range chars {
			var i int
			if r < utf8.RuneSelf {
				i = LastIndexByte(s, byte(r))
			} else {
				i = lastIndexRune(s, r)
			}
			if i > n {
				n = i
			}
		}
		return n
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
		var r rune
		if c := s[i-1]; c < utf8.RuneSelf {
			r = rune(c)
			i--
		} else if i >= 2 && c&0xC0 == 0x80 && s[i-2] >= 0xC2 && s[i-2] < 0xE0 {
			r = rune(s[i-2]&0x1F)<<6 | rune(c&0x3F)
			i -= 2
		} else {
			var size int
			r, size = utf8.DecodeLastRuneInString(s[:i])
			i -= size
		}
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
	if len(prefix) == 0 {
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
	if len(suffix) == 0 {
		return s, true
	}
	if match, i := hasSuffixUnicode(s, suffix); match {
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
