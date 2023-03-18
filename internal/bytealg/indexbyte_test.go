package bytealg

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"strings"
	"testing"
	"unicode"
	"unicode/utf8"
)

var quiet = flag.Bool("quiet", false, "quiet test output")

type IndexTest struct {
	s   string
	sep string
	out int
}

var indexTests = []IndexTest{
	{"", "", 0},
	{"", "a", -1},
	{"", "foo", -1},
	{"fo", "foo", -1},
	{"foo", "foo", 0},
	{"oofofoofooo", "f", 2},
	{"oofofoofooo", "foo", 4},
	{"barfoobarfoo", "foo", 3},
	{"foo", "", 0},
	{"foo", "o", 1},
	{"jrzm6jjhorimglljrea4w3rlgosts0w2gia17hno2td4qd1jz", "jz", 47},
	{"ekkuk5oft4eq0ocpacknhwouic1uua46unx12l37nioq9wbpnocqks6", "ks6", 52},
	{"999f2xmimunbuyew5vrkla9cpwhmxan8o98ec", "98ec", 33},
	{"9lpt9r98i04k8bz6c6dsrthb96bhi", "96bhi", 24},
	{"55u558eqfaod2r2gu42xxsu631xf0zobs5840vl", "5840vl", 33},
	// cases with one byte strings - test special case in Index()
	{"", "a", -1},
	{"x", "a", -1},
	{"x", "x", 0},
	{"abc", "a", 0},
	{"abc", "b", 1},
	{"abc", "c", 2},
	{"abc", "x", -1},
	// test special cases in Index() for short strings
	{"", "ab", -1},
	{"bc", "ab", -1},
	{"ab", "ab", 0},
	{"xab", "ab", 1},
	{"xab"[:2], "ab", -1},
	{"", "abc", -1},
	{"xbc", "abc", -1},
	{"abc", "abc", 0},
	{"xabc", "abc", 1},
	{"xabc"[:3], "abc", -1},
	{"xabxc", "abc", -1},
	{"", "abcd", -1},
	{"xbcd", "abcd", -1},
	{"abcd", "abcd", 0},
	{"xabcd", "abcd", 1},
	{"xyabcd"[:5], "abcd", -1},
	{"xbcqq", "abcqq", -1},
	{"abcqq", "abcqq", 0},
	{"xabcqq", "abcqq", 1},
	{"xyabcqq"[:6], "abcqq", -1},
	{"xabxcqq", "abcqq", -1},
	{"xabcqxq", "abcqq", -1},
	{"", "01234567", -1},
	{"32145678", "01234567", -1},
	{"01234567", "01234567", 0},
	{"x01234567", "01234567", 1},
	{"x0123456x01234567", "01234567", 9},
	{"xx01234567"[:9], "01234567", -1},
	{"", "0123456789", -1},
	{"3214567844", "0123456789", -1},
	{"0123456789", "0123456789", 0},
	{"x0123456789", "0123456789", 1},
	{"x012345678x0123456789", "0123456789", 11},
	{"xyz0123456789"[:12], "0123456789", -1},
	{"x01234567x89", "0123456789", -1},
	{"", "0123456789012345", -1},
	{"3214567889012345", "0123456789012345", -1},
	{"0123456789012345", "0123456789012345", 0},
	{"x0123456789012345", "0123456789012345", 1},
	{"x012345678901234x0123456789012345", "0123456789012345", 17},
	{"", "01234567890123456789", -1},
	{"32145678890123456789", "01234567890123456789", -1},
	{"01234567890123456789", "01234567890123456789", 0},
	{"x01234567890123456789", "01234567890123456789", 1},
	{"x0123456789012345678x01234567890123456789", "01234567890123456789", 21},
	{"xyz01234567890123456789"[:22], "01234567890123456789", -1},
	{"", "0123456789012345678901234567890", -1},
	{"321456788901234567890123456789012345678911", "0123456789012345678901234567890", -1},
	{"0123456789012345678901234567890", "0123456789012345678901234567890", 0},
	{"x0123456789012345678901234567890", "0123456789012345678901234567890", 1},
	{"x012345678901234567890123456789x0123456789012345678901234567890", "0123456789012345678901234567890", 32},
	{"xyz0123456789012345678901234567890"[:33], "0123456789012345678901234567890", -1},
	{"", "01234567890123456789012345678901", -1},
	{"32145678890123456789012345678901234567890211", "01234567890123456789012345678901", -1},
	{"01234567890123456789012345678901", "01234567890123456789012345678901", 0},
	{"x01234567890123456789012345678901", "01234567890123456789012345678901", 1},
	{"x0123456789012345678901234567890x01234567890123456789012345678901", "01234567890123456789012345678901", 33},
	{"xyz01234567890123456789012345678901"[:34], "01234567890123456789012345678901", -1},
	{"xxxxxx012345678901234567890123456789012345678901234567890123456789012", "012345678901234567890123456789012345678901234567890123456789012", 6},
	{"", "0123456789012345678901234567890123456789", -1},
	{"xx012345678901234567890123456789012345678901234567890123456789012", "0123456789012345678901234567890123456789", 2},
	{"xx012345678901234567890123456789012345678901234567890123456789012"[:41], "0123456789012345678901234567890123456789", -1},
	{"xx012345678901234567890123456789012345678901234567890123456789012", "0123456789012345678901234567890123456xxx", -1},
	{"xx0123456789012345678901234567890123456789012345678901234567890120123456789012345678901234567890123456xxx", "0123456789012345678901234567890123456xxx", 65},
	// test fallback to Rabin-Karp.
	{"oxoxoxoxoxoxoxoxoxoxoxoy", "oy", 22},
	{"oxoxoxoxoxoxoxoxoxoxoxox", "oy", -1},
}

func TestIndexByte(t *testing.T) {
	for _, tt := range indexTests {
		if len(tt.sep) != 1 {
			continue
		}
		pos := IndexByte([]byte(tt.s), tt.sep[0])
		if pos != tt.out {
			t.Errorf(`IndexByte(%q, %q) = %v; want %v`, tt.s, tt.sep[0], pos, tt.out)
		}
	}
}

func TestIndexByteString(t *testing.T) {
	for _, tt := range indexTests {
		if len(tt.sep) != 1 {
			continue
		}
		pos := IndexByteString(tt.s, tt.sep[0])
		if pos != tt.out {
			t.Errorf(`IndexByteString(%q, %q) = %v; want %v`, tt.s, tt.sep[0], pos, tt.out)
		}
	}
}

func testIndexByteASCII(t *testing.T, name string, fn func(s []byte, c byte) int) {
	s := make([]byte, 0, utf8.RuneSelf)
	for i := 0; i < 256; i++ {
		c := byte(i)
		if !isAlphaPortable(c) {
			s = append(s, c)
		}
	}
	for i, c := range s {
		o := fn(s, c)
		if o != i {
			t.Errorf(`%s(%q, %q) = %v; want %v`, name, s, c, o, i)
		}
	}
	// do it in reverse
	{
		s := s[:0]
		for i := 255; i >= 0; i-- {
			c := byte(i)
			if !isAlphaPortable(c) {
				s = append(s, c)
			}
		}
		for i, c := range s {
			o := fn(s, c)
			if o != i {
				t.Errorf(`%s(%q, %q) = %v; want %v`, name, s, c, o, i)
			}
		}
	}
}

func TestIndexByteASCII(t *testing.T) {
	testIndexByteASCII(t, "IndexByte", IndexByte)
}

func TestIndexByteStringASCII(t *testing.T) {
	testIndexByteASCII(t, "IndexByteString", func(s []byte, c byte) int {
		return IndexByteString(string(s), c)
	})
}

func max(a, b int) int {
	if a >= b {
		return a
	}
	return b
}

func testIndexByte(t *testing.T, base, name, replacements string, fn func([]byte, byte) int) {
	const maxErrors = 40
	if t.Failed() {
		t.FailNow()
		return
	}
	if strings.ContainsAny(base, "xX") {
		t.Fatalf("base string %q may not contain %q", base, "xX")
	}
	var results []string

	for size := 1; size <= os.Getpagesize(); size <<= 1 {
		t.Run(fmt.Sprintf("%d", size), func(t *testing.T) {
			const delta = 512 // Test indexes 512 bytes on either side of size

			orig := strings.Repeat(base, (size+delta/len(base))+1)
			orig = orig[:size]
			s1 := make([]byte, len(orig))
			copy(s1, orig)

			errCount := 0
			for i := max(0, size-delta*2); i < len(orig); i++ {
				for j, c := range []byte(replacements) {
					if unicode.IsLetter(rune(c)) {
						if i < len(s1)-1 {
							s1[i] = c ^ ' ' // swap case
							s1[i+1] = c
						} else {
							s1[i] = c
						}
					} else {
						s1[i] = c
					}
					if o := fn(s1, c); o != i {
						if errCount < maxErrors {
							if !*quiet {
								t.Errorf("%d.%d got: %d; want: %d", i, j, o, i)
							} else {
								t.Fail()
							}
						}
						errCount++
					}
					copy(s1[i:], orig[i:])
				}
			}
			if errCount > 0 {
				results = append(results, fmt.Sprintf("%d: failed %d/%d", size, errCount, size*2))
			}
		})
	}
	if t.Failed() {
		t.Logf("%s Summary:\n%s", name, strings.Join(results, "\n"))
	}
}

const alphaLower = "abcdefghijklmnopqrstuvwyz" // no X
const alphaUpper = "ABCDEFGHIJKLMNOPQRSTUVWYZ" // no X

func TestIndexByteLimits(t *testing.T) {
	testIndexByte(t, alphaLower, "IndexByte", "xX", IndexByte)
	testIndexByte(t, alphaUpper, "IndexByte", "xX", IndexByte)
	testIndexByte(t, alphaUpper, "IndexByte", "1", IndexByte)
}

func TestIndexByteStringLimits(t *testing.T) {
	fn := func(s []byte, c byte) int {
		return IndexByteString(string(s), c)
	}
	testIndexByte(t, alphaLower, "IndexByteString", "xX", fn)
	testIndexByte(t, alphaUpper, "IndexByteString", "xX", fn)
	testIndexByte(t, alphaUpper, "IndexByteString", "1", fn)
}

var bmbuf []byte

func valName(x int) string {
	if s := x >> 20; s<<20 == x {
		return fmt.Sprintf("%dM", s)
	}
	if s := x >> 10; s<<10 == x {
		return fmt.Sprintf("%dK", s)
	}
	return fmt.Sprint(x)
}

func benchBytes(b *testing.B, sizes []int, f func(b *testing.B, n int)) {
	for _, n := range sizes {
		b.Run(valName(n), func(b *testing.B) {
			if len(bmbuf) < n {
				bmbuf = make([]byte, n)
			}
			b.SetBytes(int64(n))
			f(b, n)
		})
	}
}

var indexSizes = []int{10, 32, 4 << 10, 4 << 20, 64 << 20}

func BenchmarkIndexByte(b *testing.B) {
	benchBytes(b, indexSizes, bmIndexByte(IndexByte, true))
}

func BenchmarkIndexBytePortable(b *testing.B) {
	b.Skip("DELETE ME")
	benchBytes(b, indexSizes, bmIndexByte(indexBytePortable, true))
}

func BenchmarkIndexByteStdLib(b *testing.B) {
	benchBytes(b, indexSizes, bmIndexByte(bytes.IndexByte, false))
}

func BenchmarkIndexByteCutover(b *testing.B) {
	// s := strings.Repeat("a", 64)
	for _, size := range [...]int{2, 4, 8, 12, 16, 32, 64} {
		b.Run(fmt.Sprintf("%d", size), func(b *testing.B) {
			s := strings.Repeat("a", size-1) + "b"
			c := byte('b')
			for i := 0; i < b.N; i++ {
				_ = IndexByteString(s, c)
			}
		})
	}
}

func bmIndexByte(index func([]byte, byte) int, caseless bool) func(b *testing.B, n int) {
	return func(b *testing.B, n int) {
		buf := bmbuf[0:n]
		buf[n-1] = 'x'
		ch := byte('x')
		if caseless {
			ch = 'X'
		}
		for i := 0; i < b.N; i++ {
			j := index(buf, ch) // Search for uppercase variant
			if j != n-1 {
				b.Fatal("bad index", j)
			}
		}
		buf[n-1] = '\x00'
	}
}

func isAlphaPortable(c byte) bool {
	return 'A' <= c && c <= 'Z' || 'a' <= c && c <= 'z'
}

func indexBytePortable(s []byte, c byte) int {
	n := bytes.IndexByte(s, c)
	if n == 0 || !isAlphaPortable(c) {
		return n
	}

	// TODO: calculate the optimal cutoff
	if n > 0 && len(s) >= 16 {
		s = s[:n] // limit search space
	}

	c ^= ' ' // swap case
	if o := bytes.IndexByte(s, c); n == -1 || (o != -1 && o < n) {
		n = o
	}
	return n
}
