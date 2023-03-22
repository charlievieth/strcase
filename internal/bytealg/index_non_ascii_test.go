package bytealg

import (
	"strings"
	"testing"
	"unicode/utf8"
)

// These tests fail with strcasestr.
var unicodeIndexTests = []IndexTest{
	// Map Kelvin 'K' (U+212A) to lowercase latin 'k'.
	{"abcK@", "k@", 3},

	// Map the long 'S' 'ſ' to 'S' and 's'.
	{"abcſ@", "s@", 3},
	{"abcS@", "ſ@", 3},

	// Test with a unicode prefix in the substr to make sure the unicode
	// implementation is correct.
	{"abc☻K@", "☻k@", 3},
	{"abc☻S@", "☻ſ@", 3},

	// Sep longer (in bytes)
	{"a\u212a", "a\u212a", 0},
	{"a\u212a", "a\u212a\u212a", -1},

	// WARN: fix these
	{"İ", "İ", 0},
	{"İİ", "İİ", 0},
	{"İİİİ", "İİ", 0},
	{"İİİİİİ", "İİ", 0},
	{"0123456789İİ", "İİ", 10},
	{"01234567890123456789İİ", "İİ", 20},
	{"İİ" + strings.Repeat("a", 64), "İİ" + strings.Repeat("a", 64), 0},

	// Tests discovered with fuzzing
	{"4=K ", "=\u212a", 1},
	{"I", "\u0131", -1},

	// Evil strings discovered fuzzing.
	{
		s:   "32ⱭⱭⱭⱭⱭ45678890ⱭⱭⱭⱭⱭ234567890ⱭⱭⱭⱭⱭ234567890ⱭⱭⱭⱭⱭ23456789ⱭⱭⱭⱭⱭⱭⱭⱭⱭⱭ",
		sep: "0ⱭⱭⱭⱭⱭ234567890ⱭⱭⱭⱭⱭ234567890ⱭⱭⱭⱭⱭ234567890",
		out: -1,
	},
	{
		s:   "<<\ua7ac\x02\ub680\U0010f410\U0002ac40\n\x15\u2126\ufa12\x14",
		sep: "<\ua7ac\x02\ub680\U0010f410\U0002ac40\n\x15\u03c9\ufa12",
		out: 1,
	},
	{
		s:   "\U00024b8a\u2c65I\u7c12\u313a/A\u027d\u017f=\x05",
		sep: "\U00024b8a\u2c65I\u7c12\u313a/a\u2c64\u017f=",
		out: 0,
	},
	{
		s:   "\U0002a433\u3577\U000230d6\U001024b4\u73f1\u56f0\U0002d7db\U0010e3ac\U000204ca\u2575~\u8825\U0002ba82\U0002c0e4\u743aK]",
		sep: "\u743a\u212a",
		out: 48,
	},
	{
		s:   "z0\U0010640b\U0001f326k-k\U00100621\U000240ff\U000e013fl",
		sep: "\u212a-\u212a",
		out: 10,
	},
	{
		s:   "\U0007279d\ufffd\ufffd\ufffd\ufffd\ufffd\ufffd\ufffd",
		sep: "\ufffd\ufffd\ufffd\ufffd\ufffd",
		out: 4,
	},
}

// TODO: do we need this?
func init() {
	p0 := strings.Repeat("\u212a", 64)
	p1 := strings.Repeat("K", 64)
	n := utf8.RuneLen('\u212a')
	for i := 2; i <= 64; i += 2 {
		s0 := p0[:i*n]
		s1 := p1[:i]
		unicodeIndexTests = append(unicodeIndexTests, IndexTest{s0, s1, 0}, IndexTest{s1, s0, 0})
	}
}

func testIndexNonASCII(t *testing.T, name string, fn func(s string) int) {
	const maxFailures = 80

	index := func(s string) int {
		for i, r := range s {
			if r >= utf8.RuneSelf {
				return i
			}
		}
		return -1
	}

	t.Run("IndexTests", func(t *testing.T) {
		fails := 0
		tests := append([]IndexTest(nil), indexTests...)
		tests = append(tests, unicodeIndexTests...)
		for _, test := range tests {
			want := index(test.s)
			got := fn(test.s)
			if got != want {
				fails++
				if fails <= maxFailures {
					t.Errorf("%s(%q) = %d; want: %d", name, test.s, got, want)
				}
			}
		}

		if fails > 0 {
			t.Errorf("Failed: %d/%d", fails, len(tests))
		}
	})

	t.Run("LongString", func(t *testing.T) {
		fails := 0

		long := strings.Repeat("a", 4096) + "βaβa"
		idx := index(long)
		for i := 0; i < len(long); i++ {
			s := long[i:]
			want := idx - i
			if want < 0 {
				want = index(s)
			}
			got := fn(s)
			if got != want {
				fails++
				if fails <= maxFailures {
					t.Errorf("%s(long[%d:]) = %d; want: %d", name, i, got, want)
				}
			}
		}

		if fails > 0 {
			t.Errorf("Failed: %d/%d", fails, len(long))
		}
	})

}

func TestIndexNonASCII(t *testing.T) {
	testIndexNonASCII(t, "IndexNonASCII", IndexNonASCII)
}

func TestIndexByteNonASCII(t *testing.T) {
	testIndexNonASCII(t, "IndexByteNonASCII", func(s string) int {
		return IndexByteNonASCII([]byte(s))
	})
}

func benchIndexNonASCII(b *testing.B, sizes []int, f func(b *testing.B, n int)) {
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

func BenchmarkIndexByteNonASCII(b *testing.B) {
	benchIndexNonASCII(b, indexSizes, bmIndexNonASCII(IndexByteNonASCII))
}

func bmIndexNonASCII(index func([]byte) int) func(b *testing.B, n int) {
	return func(b *testing.B, n int) {
		buf := bmbuf[0:n]
		for i := 0; i < b.N; i++ {
			_ = index(buf) // Search for uppercase variant
		}
	}
}
