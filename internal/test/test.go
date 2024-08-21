package test

import (
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"unicode"
	"unicode/utf8"

	"github.com/charlievieth/strcase/internal/tables"
)

type IndexFunc func(s, substr string) int

func ByteIndexFunc(fn func(s, sep []byte) int) IndexFunc {
	return func(s, sep string) int {
		return fn([]byte(s), []byte(sep))
	}
}

func WrapRabinKarp(rabinKarp IndexFunc) IndexFunc {
	return func(s, substr string) int {
		if len(substr) == 0 {
			// Can't use Rabin-Karp for this test case
			return 0
		}
		return rabinKarp(s, substr)
	}
}

type ContainsFunc func(s, substr string) bool

func ByteContainsFunc(fn func(s, sep []byte) bool) ContainsFunc {
	return func(s, sep string) bool {
		return fn([]byte(s), []byte(sep))
	}
}

type IndexRuneFunc func(s string, r rune) int

func ByteIndexRuneFunc(fn func(s []byte, r rune) int) IndexRuneFunc {
	return func(s string, r rune) int {
		return fn([]byte(s), r)
	}
}

type IndexByteFunc func(s string, c byte) int

func ByteIndexByte(fn func(s []byte, c byte) int) IndexByteFunc {
	return func(s string, c byte) int {
		return fn([]byte(s), c)
	}
}

type PrefixFunc func(s, prefix string) (bool, bool)

func BytePrefixFunc(fn func(s, prefix []byte) (bool, bool)) PrefixFunc {
	return func(s, prefix string) (bool, bool) {
		return fn([]byte(s), []byte(prefix))
	}
}

type TrimFunc func(s1, s2 string) string

func ByteTrimFunc(fn func(s1, s2 []byte) []byte) TrimFunc {
	return func(s1, s2 string) string {
		return string(fn([]byte(s1), []byte(s2)))
	}
}

func UnicodeVersion(t *testing.T, version string) {
	if version != unicode.Version {
		t.Fatalf("unicode.Version (%s) != UnicodeVersion (%s):\n"+
			"The version of Unicode included in the version of Go (%s) running this test\n"+
			"does not match the Unicode version the strcase tables were generated with.\n"+
			"\n"+
			"This is likely due to the Unicode version being updated in a newer Go release.\n"+
			"To regenerate the Unicode tables run: `go generate` and check in the changes to\n"+
			"\"tables.go\" and \".tables.json\".\n"+
			"\n"+
			"NOTE: re-generating the Unicode tables can take a few minutes.",
			unicode.Version, version, runtime.Version())
	}
}

type compareTest struct {
	s, t string
	out  int
}

var compareTests = []compareTest{
	{"", "", 0},
	{"a", "a", 0},
	{"a", "ab", -1},
	{"ab", "a", 1},
	{"ABC", "abd", -1},
	{"abc", "ABD", -1},
	{"abd", "ABC", 1},
	{"123abc", "123ABC", 0},
	{"αβδ", "ΑΒΔ", 0},
	{"ΑΒΔ", "αβδ", 0},
	{"αβδa", "ΑΒΔ", 1},
	{"αβδ", "ΑΒΔa", -1},
	{"αβa", "ΑΒΔ", -1},
	{"ΑΒΔ", "αβa", 1},
	{"αβδ", "ΑΒa", 1},
	{"αabc", "αABD", -1},
	{"αabd", "αABC", 1},
	{strings.Repeat("\u212a", 8), strings.Repeat("k", 8), 0},

	// Invalid UTF-8 should be considered equal (mapped to RuneError)
	{"a" + string(utf8.RuneError), "a" + string(unicode.MaxRune+1), 0},
	{"a" + string(utf8.RuneError), "a\xFF", 0},
	{"\xed\xa0\x80", "\xed\xa0\x81", 0},
	{"\xF4\x7F\xBF\xBF", "\xF2\x7F\xBF\xBF", 0},
}

func Compare(t *testing.T, fn IndexFunc) {
	// Test the tests (NOTE: we may want to remove this at some point since
	// strings.ToLower is not always correct).
	for i, test := range compareTests {
		got := strings.Compare(strings.ToLower(test.s), strings.ToLower(test.t))
		if got != test.out {
			t.Errorf("%d: strings.Compare(%q, %q) = %d; want: %d",
				i, strings.ToLower(test.s), strings.ToLower(test.t), got, test.out)
		}
	}
	if t.Failed() {
		t.Fatal("invalid Compare tests")
		return
	}

	for i, test := range compareTests {
		got := fn(test.s, test.t)
		if got != test.out {
			t.Errorf("%d: Compare(%q, %q) = %d; want: %d", i, test.s, test.t, got, test.out)
		}
	}
}

func EqualFold(t *testing.T, fn func(s1, s2 string) bool) {
	// Ensure that strings.EqualFold does not match 'İ' (U+0130)
	// and ASCII 'i' or 'I'. This is mostly a sanity check.
	tests := append(compareTests,
		compareTest{"İ", "i", 1},
		compareTest{"İ", "I", 1},
	)
	for _, test := range tests {
		want := test.out == 0
		got := strings.EqualFold(test.s, test.t)
		if got != want {
			t.Errorf("strings.EqualFold(%q, %q) = %t; want: %t", test.s, test.t, got, want)
		}
	}
	if t.Failed() {
		t.Fatal("invalid EqualFold tests")
		return
	}

	for _, test := range tests {
		want := test.out == 0
		got := fn(test.s, test.t)
		if got != want {
			t.Errorf("EqualFold(%q, %q) = %t; want: %t", test.s, test.t, got, want)
		}
	}
}

type indexTest struct {
	s   string
	sep string
	out int
}

var indexTests = []indexTest{
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
	{"abcABCabc", "A", 0},
	{"abcVBCabc", "V", 3},
	// cases with one byte strings - test special case in Index()
	{"", "a", -1},
	{"x", "a", -1},
	{"x", "x", 0},
	{"abc", "a", 0},
	{"abc", "b", 1},
	{"abc", "c", 2},
	{"ABC", "BC", 1},
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

	// Invalid UTF8
	{"abc" + string(rune(utf8.RuneError)) + "123", string(rune(utf8.RuneError)), 3},
	{"abc", string(rune(utf8.RuneError)), -1},
	{"abc", string(rune(utf8.MaxRune)), -1},

	// test fallback to Rabin-Karp.
	{"oxoxoxoxoxoxoxoxoxoxoxoy", "oy", 22},
	{"oxoxoxoxoxoxoxoxoxoxoxox", "oy", -1},

	// Actually test fallback to Rabin-Karp (the above tests don't trigger it).
	{strings.Repeat("ox", 64) + "yox", "oα" + strings.Repeat("ox", 32/len("ox")), -1},
	{strings.Repeat("ox", 64) + "oα" + strings.Repeat("ox", 32/2), "oα" + strings.Repeat("ox", 32/2), 128},

	// Sep longer (in bytes) than s
	{"aa", "aaa", -1},
	{"aa", "aaaa", -1},
	{"aa", "aaaaa", -1},

	// Unicode strings
	{"oxoxoxoxoxoxoxoxoxoxoxoyoα", "oα", 24},
	{"oxoxoxoxoxoxoxoxoxoxoxα", "α", 22},

	// test fallback to Rabin-Karp (unicode).
	{"xx0123456789012345678901234567890123456789012345678901234567890120123456789012345678901234567890123456xxx☻", "0123456789012345678901234567890123456xxx☻", 65},

	{"abc☻", "abc☻", 0},
	{"abc☻", "ABC☻", 0},
	{"123abc☻", "ABC☻", 3},
}

// These tests fail with strcasestr.
var unicodeIndexTests = []indexTest{
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
	{"\u212a", "a", -1},
	{"\u212a", "k", 0},
	{"a\u212a", "a\u212a", 0},
	{"a\u212a", "a\u212a\u212a", -1},

	// Test that İ does not fold to [Ii]
	{"İ", "İ", 0},
	{"İ", "i", -1},
	{"İ", "I", -1},
	{"İİ", "İİ", 0},
	{"İİİİ", "İİ", 0},
	{"İİİİİİ", "İİ", 0},
	{"0123456789İİ", "İİ", 10},
	{"01234567890123456789İİ", "İİ", 20},
	{"İİ" + strings.Repeat("a", 64), "İİ" + strings.Repeat("a", 64), 0},

	// "İ" does not fold to "i"
	{"İ", "i", -1},
	{"aİ", "ai", -1},
	{"aİ", "ai", -1},

	// Special Unicode points that are not equal to either their
	// uppercase or lowercase form.
	{"aǈǇǉb", "ǉǉ", 1},
	{"aǲǱǳb", "ǳǱǲ", 1},
	{"ǲǱǳǲǱǳ", "ǳǱǲa", -1},

	// Test the cutover to to bytealg.IndexString when it is triggered in
	// the middle of rune that contains consecutive runs of equal bytes.
	{"aaaaaKKKK\U000bc104a", "\U000bc104a", 17}, // cutover: (n + 16) / 8
	{"aaaaaKKKK鄄a", "鄄a", 17},
	{"aaKKKKKa\U000bc104a", "\U000bc104a", 18}, // cutover: 4 + n>>4
	{"aaKKKKKa鄄a", "鄄a", 18},

	// Test cases found by fuzzing
	{"\x00iK", "iK", 1},
	{"İKKKK\x00iK", "iK", 15},
	{"İKKKKiK", "iK", 14},
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
	{
		s:   ">Ā\U000c4c1bKKKK\x00YUв\U000bc104q9",
		sep: "\U000bc104q9",
		out: 24,
	},
	{
		s:   "\U000bc104q9",
		sep: "\U000bc104q9",
		out: 0,
	},
}

func init() {
	// Append some test cases that include Kelvin K and ASCII K. Since Kelvin
	// K is 3x the width of ASCII [Kk] we want to test the logic for handling
	// that.
	p0 := strings.Repeat("\u212a", 64) // Kelvin K
	p1 := strings.Repeat("K", 64)
	n := utf8.RuneLen('\u212a')
	for i := 2; i <= 64; i *= 2 {
		s0 := p0[:i*n]
		s1 := p1[:i]
		unicodeIndexTests = append(unicodeIndexTests, indexTest{s0, s1, 0}, indexTest{s1, s0, 0})
	}
}

var lastIndexTests = []indexTest{
	{"", "", 0},
	{"", "a", -1},
	{"", "foo", -1},
	{"fo", "foo", -1},
	{"foo", "foo", 0},
	{"foo", "f", 0},
	{"oofofoofooo", "f", 7},
	{"oofofoofooo", "foo", 7},
	{"barfoobarfoo", "foo", 9},
	{"foo", "", 3},
	{"foo", "o", 2},
	{"abcABCabc", "A", 6},
	{"abcABCabc", "a", 6},

	// Invalid UTF8
	{"abc" + string(rune(utf8.RuneError)) + "123", string(rune(utf8.RuneError)), 3},
	{"abc", string(rune(utf8.RuneError)), -1},
	{"abc", string(rune(utf8.MaxRune)), -1},
	{
		string(rune(unicode.MaxRune)),
		string(rune(unicode.MaxRune)),
		strings.LastIndex(string(rune(unicode.MaxRune)), string(rune(unicode.MaxRune))),
	},
	{
		"a" + string(rune(unicode.MaxRune)),
		string(rune(unicode.MaxRune)),
		strings.LastIndex("a"+string(rune(unicode.MaxRune)), string(rune(unicode.MaxRune))),
	},
	{
		string(rune(unicode.MaxRune + 1)),
		string(rune(unicode.MaxRune + 1)),
		strings.LastIndex(string(rune(unicode.MaxRune+1)), string(rune(unicode.MaxRune+1))),
	},
	{
		"a" + string(rune(unicode.MaxRune+1)),
		string(rune(unicode.MaxRune + 1)),
		strings.LastIndex("a"+string(rune(unicode.MaxRune+1)), string(rune(unicode.MaxRune+1))),
	},

	// Unicode

	{"fooΑΒΔbar", "αβδ", 3},

	// Map Kelvin 'K' (U+212A) to lowercase latin 'k'.
	{"abcK@", "k@", 3},

	// Map the long 'S' 'ſ' to 'S' and 's'.
	{"abcſ@", "s@", 3},
	{"abcS@", "ſ@", 3},

	// Test with a unicode prefix in the substr to make sure the unicode
	// implementation is correct.
	{"abc☻K@", "☻k@", 3},
	{"abc☻S@", "☻ſ@", 3},

	// Sep longer (in bytes) than s
	{"aa", "aaa", -1},
	{"aa", "aaaa", -1},
	{"aa", "aaaaa", -1},
	{"a\u212a", "a\u212a", 0},
	{"a\u212a", "a\u212a\u212a", -1},
	{"a\u212a", "a\u212a\u212a\u212a", -1},
	{"a\u212aa", "ka", 1},

	// Tests discovered with fuzzing
	{"4=K ", "=\u212a", 1},
	{"I", "\u0131", -1},
	{"aßẛ", "ß", 1},
	{"aßẛ", "a", 0},
	{"OFf", "İF", -1},
	{"``Ɽ", "\U000823eb`", -1},

	{"\u0250\u0250\u0250\u0250\u0250 a", "\u2C6F\u2C6F\u2C6F\u2C6F\u2C6F A", 0}, // grows one byte per char
	{"a\u0250\u0250\u0250\u0250\u0250", "A\u2C6F\u2C6F\u2C6F\u2C6F\u2C6F", 0},   //
	{"\u2C6D\u2C6D\u2C6D\u2C6D\u2C6D a", "\u0251\u0251\u0251\u0251\u0251 A", 0}, // shrinks one byte per char
	{"a\u2C6D\u2C6D\u2C6D\u2C6D\u2C6D", "A\u0251\u0251\u0251\u0251\u0251", 0},   // shrinks one byte per char
	{"abc\u2C6D\u2C6D\u2C6D\u2C6D\u2C6D", "\u0251\u0251\u0251\u0251\u0251", 3},
	{"ΑΒΔ\u2C6D\u2C6D\u2C6D\u2C6D\u2C6D", "\u0251\u0251\u0251\u0251\u0251", len("ΑΒΔ")},
}

// Execute f on each test case.  funcName should be the name of f; it's used
// in failure reports.
func runIndexTests(t *testing.T, f IndexFunc, funcName string, testCases []indexTest, noError bool) {
	t.Helper()
	fails := 0
	for _, test := range testCases {
		actual := f(test.s, test.sep)
		if actual != test.out {
			fails++
			errorf := t.Errorf
			if noError {
				errorf = t.Logf
			}
			var foldable bool
			for _, r := range test.sep {
				foldable = tables.FoldMap(r) != nil
				if foldable {
					break
				}
			}
			errorf("%s\n"+
				"S:    %q\n"+
				"Sep:  %q\n"+
				"Got:  %d\n"+
				"Want: %d\n"+
				"Fold: %t\n"+
				"\n"+
				"S:    %s\n"+
				"Sep:  %s\n"+
				"Lower:\n"+
				"S:    %s\n"+
				"Sep:  %s\n"+
				"\n",
				funcName,
				test.s, test.sep, actual, test.out,
				foldable,
				strconv.QuoteToASCII(test.s),
				strconv.QuoteToASCII(test.sep),
				strconv.QuoteToASCII(strings.ToLower(test.s)),
				strconv.QuoteToASCII(strings.ToLower(test.sep)),
			)
		}
	}
	if t.Failed() && testing.Verbose() {
		t.Logf("%s: failed %d out of %d tests", funcName, fails, len(testCases))
	}
}

func Index(t *testing.T, fn IndexFunc) {
	if t.Failed() {
		t.Fatal("Reference Index function failed: tests are invalid")
	}
	runIndexTests(t, fn, "Index", unicodeIndexTests, false)
}

func IndexUnicode(t *testing.T, fn IndexFunc) {
	type Replacement struct {
		old, new string
	}
	replacements := [][]Replacement{
		{{"a", "α"}, {"A", "Α"}, {"1", "Δ"}},
		{{"a", "α"}, {"A", "Α"}, {"1", "日a本b語ç日ð本Ê語þ日¥本¼語i日©"}},
		{{"1", "\u2C6D\u2C6D\u2C6D\u2C6D\u2C6D"}}, // shrinks one byte per char
		{{"1", "\u0250\u0250\u0250\u0250\u0250"}}, // grows one byte per char
	}
	for _, reps := range replacements {
		t.Run("", func(t *testing.T) {
			r := func(s string) string {
				for _, rr := range reps {
					o := strings.ReplaceAll(s, rr.old, rr.new)
					if !utf8.ValidString(o) {
						t.Fatalf("Invalid transformation %q => %q", s, o)
					}
					s = o
				}
				return s
			}

			tests := append([]indexTest(nil), indexTests...)
			for i, test := range tests {
				if test.out > 0 {
					test.out = len(r(test.s[:test.out]))
				}
				test.s = r(test.s)
				test.sep = r(test.sep)
				tests[i] = test
			}

			runIndexTests(t, fn, "Index", tests, false)

			// // TODO: can probably remove this
			// t.Run("RabinKarp", func(t *testing.T) {
			// 	filter := func(t IndexTest) bool {
			// 		return len(t.sep) > 0 && len(t.s) > len(t.sep)
			// 	}
			// 	rtests := filterIndexTests(filter, tests)
			// 	runIndexTests(t, indexRabinKarpUnicode, "indexRabinKarpUnicode", rtests, false)
			// })
		})
	}
}

// Test our use of bytealg.IndexString
func IndexNumeric(t *testing.T, fn IndexFunc) {
	ns := strings.Repeat("1234", 128/4)
	hs := strings.Repeat(" ", 256)
	tests := make([]indexTest, 0, 1024)
	// Test the boundaries around the bytealg.MaxBruteForce cutover
	for _, i := range []int{1, 4, 8, 15, 16, 17, 31, 32, 33, 63, 64, 65, 128} {
		for j := 0; j <= len(hs); j += 3 {
			sep := ns[:i]
			tests = append(tests, indexTest{
				s:   hs[:j] + sep,
				sep: sep,
				out: j,
			})
			if len(sep) > 1 {
				tests = append(tests, indexTest{
					s:   hs[:j] + sep[:len(sep)-1] + " ",
					sep: sep,
					out: -1,
				})
			}
		}
	}
	runIndexTests(t, fn, "Index", tests, false)
}

// Extensively test the handling of Kelvin K since it is three times the size
// of ASCII [Kk] it requires special handling.
func IndexKelvin(t *testing.T, fn IndexFunc) {
	const K = "\u212A" // Kelvin

	test := func(t *testing.T, s, substr string, want int) {
		t.Helper()
		if got := fn(s, substr); got != want {
			t.Errorf("Index(%q, %q) = %d; want: %d", s, substr, got, want)
		}
	}

	t.Run("Match0", func(t *testing.T) {
		for i := 1; i < 128; i++ {
			s := strings.Repeat("k", i)
			substr := strings.Repeat(K, i)
			test(t, s, substr, 0)
			test(t, K+s[:len(s)-1], substr, 0)
			test(t, s[:len(s)-1]+K, substr, 0)
		}
	})

	r := strings.Repeat
	t.Run("Match1", func(t *testing.T) {
		for i := 1; i < 128; i++ {
			test(t, "a"+r("k", i), r(K, i), 1)
		}
	})
	t.Run("NoMatchPrefix", func(t *testing.T) {
		for i := 1; i < 128; i++ {
			test(t, "a"+r("k", i-1), r(K, i), -1)
		}
	})
	t.Run("NoMatchSuffix", func(t *testing.T) {
		for i := 1; i < 128; i++ {
			test(t, r("k", i-1)+"a", r(K, i), -1)
		}
	})
}

func Contains(t *testing.T, fn ContainsFunc) {
	for _, test := range indexTests {
		got := fn(test.s, test.sep)
		want := test.out >= 0
		if got != want {
			t.Errorf("Contains(%q, %q) = %t; want: %t", test.s, test.sep, got, want)
		}
	}
}

const (
	a32  = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa" // "a" repeated 32 times
	dots = "1....2....3....4"
)

var ContainsAnyTests = []struct {
	str, substr string
	expected    bool
}{
	{"", "", false},
	{"", "a", false},
	{"", "abc", false},
	{"a", "", false},
	{"a", "a", true},
	{"aaa", "a", true},
	{"abc", "xyz", false},
	{"abc", "xcz", true},
	{"bas", "SsKs", true},
	{"bak", "SsKs", true},
	{a32 + "\u212a", "k", true},
	{a32 + "\u212a", "K", true},
	{"a☺b☻c☹d", "uvw☻xyz", true},
	{"aRegExp*", ".(|)*+?^$[]", true},
	{dots + dots + dots, " ", false},

	// Case-insensitive
	{"a", "A", true},
	{"aaa", "A", true},
	{"αβa", "ΑΒΔ", true},

	// Use asciiSet only if str is all ASCII
	{a32, "sS", false},
	{a32, "kK", false},
	// Cannot use asciiSet fallback to Unicode aware algorithm
	{a32 + "\u212a", "sS", false},
	{a32 + "\u212a", "kK", true},
	{a32, "kK" + "\u212a", false},
}

func ContainsAny(t *testing.T, fn ContainsFunc) {
	for _, ct := range ContainsAnyTests {
		if fn(ct.str, ct.substr) != ct.expected {
			t.Errorf("ContainsAny(%s, %s) = %v, want %v",
				ct.str, ct.substr, !ct.expected, ct.expected)
		}
	}
}

func LastIndex(t *testing.T, fn IndexFunc) {
	reference := func(s, sep string) int {
		return LastIndexRunesReference([]rune(s), []rune(sep))
	}
	runIndexTests(t, reference, "LastIndexReference", lastIndexTests, false)

	runIndexTests(t, fn, "LastIndex", lastIndexTests, false)
}

type indexRuneTest struct {
	in   string
	rune rune
	want int
}

var indexRuneTests = []indexRuneTest{
	{"", 'a', -1},
	{"", '☺', -1},
	{"foo", '☹', -1},
	{"foo", 'o', 1},
	{"foo☺bar", '☺', 3},
	{"foo☺☻☹bar", '☹', 9},
	{"a A x", 'A', 0},
	{"some_text=some_value", '=', 9},
	{"☺a", 'a', 3},
	{"a☻☺b", '☺', 4},
	{"abc𐀀", '𐀀', 3},

	// RuneError should match any invalid UTF-8 byte sequence.
	{"�", '�', 0},
	{"\xff", '�', 0},
	{"☻x�", '�', len("☻x")},
	{"☻x\xe2\x98", '�', len("☻x")},
	{"☻x\xe2\x98�", '�', len("☻x")},
	{"☻x\xe2\x98x", '�', len("☻x")},

	// Invalid rune values should never match.
	{"a☺b☻c☹d\xe2\x98�\xff�\xed\xa0\x80", -1, -1},
	{"a☺b☻c☹d\xe2\x98�\xff�\xed\xa0\x80", 0xD800, -1}, // Surrogate pair
	{"a☺b☻c☹d\xe2\x98�\xff�\xed\xa0\x80", utf8.MaxRune + 1, -1},

	// Case-folding
	{"Αβδ", 'α', 0}, // "ΑΒΔ"
	{"αβδ", 'Α', 0}, // ""
	{"αβδ", 'Δ', 4}, // "ΑΒΔ"

	// Case-folding with ASCII
	{"K", 'K', 0},  // U+212A
	{"S", 'ſ', 0},  // U+017F
	{"K", 'k', 0},  // U+006B
	{"ſ", 's', 0},  // U+0073
	{"İ", 'İ', 0},  // U+0130
	{"i", 'İ', -1}, // U+0130
	{"ſS*ք", 'S', 0},

	// Test cutover when strings.IndexByte does not advance far
	// enough. All the runes here have the same last byte when
	// encoded as UTF-8.
	{strings.Repeat("ā", 128) + "Á", 'Á', len("ā") * 128}, // 2 bytes per-rune
	{strings.Repeat("ā", 128), 'Á', -1},
	{strings.Repeat("ᲅ", 128) + "ꙅ", 'ꙅ', len("ᲅ") * 128}, // 3 bytes per-rune
	{strings.Repeat("ᲅ", 128), 'ꙅ', -1},
	{strings.Repeat("𥺻", 128) + "𥻻", '𥻻', len("𥺻") * 128}, // 4 bytes per-rune
	{strings.Repeat("𥺻", 128), '𥻻', -1},

	// Test the cutover to to bytealg.IndexString when it is triggered in
	// the middle of rune that contains consecutive runs of equal bytes.
	{"aaaaaKKKK\U000bc104", '\U000bc104', 17}, // cutover: (n + 16) / 8
	{"aaaaaKKKK鄄", '鄄', 17},
	{"aaKKKKKa\U000bc104", '\U000bc104', 18}, // cutover: 4 + n>>4
	{"aaKKKKKa鄄", '鄄', 18},

	// Invalid rune
	{"abc", utf8.RuneError, -1},
}

func IndexRune(t *testing.T, fn IndexRuneFunc) {
	for _, tt := range indexRuneTests {
		if got := fn(tt.in, tt.rune); got != tt.want {
			t.Errorf("IndexRune(%q, %q) = %v; want %v", tt.in, tt.rune, got, tt.want)
		}
	}
}

func IndexRuneCase(t *testing.T, fn IndexRuneFunc) {
	tests := []indexRuneTest{
		{"", 'a', -1},
		{"", '☺', -1},
		{"foo", '☹', -1},
		{"foo", 'o', 1},
		{"foo☺bar", '☺', 3},
		{"foo☺☻☹bar", '☹', 9},
		{"a A x", 'A', 2},
		{"some_text=some_value", '=', 9},
		{"☺a", 'a', 3},
		{"a☻☺b", '☺', 4},

		// RuneError should match any invalid UTF-8 byte sequence.
		{"a", utf8.RuneError, -1},
		{"�", '�', 0},
		{"\xff", '�', 0},
		{"☻x�", '�', len("☻x")},
		{"☻x\xe2\x98", '�', len("☻x")},
		{"☻x\xe2\x98�", '�', len("☻x")},
		{"☻x\xe2\x98x", '�', len("☻x")},

		// Invalid rune values should never match.
		{"a☺b☻c☹d\xe2\x98�\xff�\xed\xa0\x80", -1, -1},
		{"a☺b☻c☹d\xe2\x98�\xff�\xed\xa0\x80", 0xD800, -1}, // Surrogate pair
		{"a☺b☻c☹d\xe2\x98�\xff�\xed\xa0\x80", utf8.MaxRune + 1, -1},

		// Make sure IndexRune does not panic when the byte being searched
		// for occurs at the end of the string.
		{"abcÀ"[:len("abcÀ")-1], 'À', -1},
		{"abc本"[:len("abc本")-1], '本', -1},
		{"abc本"[:len("abc本")-2], '本', -1},
		{"abc𐀀"[:len("abc𐀀")-1], '𐀀', -1},
		{"abc𐀀"[:len("abc𐀀")-2], '𐀀', -1},
		{"abc𐀀"[:len("abc𐀀")-3], '𐀀', -1},
	}
	for _, tt := range tests {
		if got := fn(tt.in, tt.rune); got != tt.want {
			t.Errorf("indexRuneCase(%q, %d) = %v; want %v", tt.in, tt.rune, got, tt.want)
		}
	}
}

func ContainsRune(t *testing.T, fn func(s string, r rune) bool) {
	for _, test := range indexRuneTests {
		got := fn(test.in, test.rune)
		want := test.want >= 0
		if got != want {
			t.Errorf("ContainsRune(%q, 0x%04X) = %t; want: %t", test.in, test.rune, got, want)
		}
	}
}

func LastIndexRune(t *testing.T, fn IndexRuneFunc) {
	tests := []struct {
		in   string
		rune rune
		want int
	}{
		{"", 'a', -1},
		{"", '☺', -1},
		{"foo", '☹', -1},
		{"foo", 'o', 2},
		{"foo☺bar", '☺', 3},
		{"foo☺☻☹bar", '☹', 9},
		{"a A x", 'A', 2},
		{"some_text=some_value", '=', 9},
		{"☺a", 'a', 3},
		{"a☻☺b", '☺', 4},

		// RuneError should match any invalid UTF-8 byte sequence.
		{"�", '�', 0},
		{"\xff", '�', 0},
		{"☻x�", '�', len("☻x")},

		// Invalid rune values should never match.
		{"foo" + string(rune(utf8.RuneError)), utf8.RuneError, 3},
		{"foo" + string(rune(unicode.MaxRune+1)), unicode.MaxRune + 1, -1},
		{"foo" + string(utf8.RuneError), utf8.RuneError, 3},
		{"a☺b☻c☹d\xe2\x98�\xff�\xed\xa0\x80", -1, -1},
		{"a☺b☻c☹d\xe2\x98�\xff�\xed\xa0\x80", 0xD800, -1}, // Surrogate pair
		{"a☺b☻c☹d\xe2\x98�\xff�\xed\xa0\x80", utf8.MaxRune + 1, -1},

		// Case-folding
		{"Αβδ", 'α', 0}, // "ΑΒΔ"
		{"αβδ", 'Α', 0}, // "ΑΒΔ"
		{"αβδ", 'Δ', 4}, // "ΑΒΔ"
		{"αβδ", 'Δ', 4}, // "ΑΒΔ"
		{"abcßẞ", 'ß', len("abcß")},
		{"aΩωΩ", 'ω', len("aΩω")},
		{"Θθϑϴabc", 0x03D1, len("Θθϑ")},

		// Case-folding with ASCII
		{"K", 'K', 0},  // U+212A
		{"S", 'ſ', 0},  // U+017F
		{"K", 'k', 0},  // U+006B
		{"ſ", 's', 0},  // U+0073
		{"İ", 'İ', 0},  // U+0130
		{"i", 'İ', -1}, // U+0130
	}
	for _, tt := range tests {
		if got := fn(tt.in, tt.rune); got != tt.want {
			t.Errorf("lastIndexRune(%q, %q) = %v; want %v", tt.in, tt.rune, got, tt.want)
		}
	}
}

func IndexByte(t *testing.T, fn IndexByteFunc) {
	tests := []struct {
		in   string
		char byte
		want int
	}{
		// Case-folding with ASCII
		{"", 0, -1},
		{"K", 'k', 0},
		{"K", 'K', 0},
		{"ſ", 's', 0},
		{"ſ", 'S', 0},
		{"sſ", 'S', 0},
		{"aKkK", 'k', 1},
		{"aſSs", 's', 1},
	}
	for _, tt := range tests {
		if got := fn(tt.in, tt.char); got != tt.want {
			t.Errorf("IndexByte(%q, %q) = %v; want %v", tt.in, tt.char, got, tt.want)
		}
	}
}

func LastIndexByte(t *testing.T, fn IndexByteFunc) {
	tests := []struct {
		in   string
		char byte
		want int
	}{
		{"", 'a', -1},
		{"1", '2', -1},
		{"abc", 'A', 0},
		{"abc", 'B', 1},
		{"abc", 'c', 2},
		{"abc", 'x', -1},

		// Case-folding with ASCII
		{"K", 'k', 0},
		{"K", 'K', 0},
		{"ſ", 's', 0},
		{"ſ", 'S', 0},
		{"x", 'S', -1},
		{"akK", 'k', len("ak")},
		{"aſSx", 's', len("aſ")},
	}
	for _, tt := range tests {
		if got := fn(tt.in, tt.char); got != tt.want {
			t.Errorf("LastIndexByte(%q, %q) = %v; want %v", tt.in, tt.char, got, tt.want)
		}
	}
}

func IndexNonASCII(t *testing.T, fn func(s string) int) {
	index := func(s string) int {
		for i, r := range s {
			if r >= utf8.RuneSelf {
				return i
			}
		}
		return -1
	}

	t.Run("IndexTests", func(t *testing.T) {
		tests := append([]indexTest(nil), indexTests...)
		tests = append(tests, unicodeIndexTests...)
		for _, test := range tests {
			want := index(test.s)
			got := fn(test.s)
			if got != want {
				t.Errorf("IndexNonASCII(%q) = %d; want: %d", test.s, got, want)
			}
		}
	})

	t.Run("LongString", func(t *testing.T) {
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
				t.Errorf("IndexNonASCII(long[%d:]) = %d; want: %d", i, got, want)
			}
		}
	})
}

func ContainsNonASCII(t *testing.T, fn func(s string) bool) {
	contains := func(s string) bool {
		for i := 0; i < len(s); i++ {
			if s[i] >= utf8.RuneSelf {
				return true
			}
		}
		return false
	}

	tests := append([]indexTest(nil), indexTests...)
	tests = append(tests, unicodeIndexTests...)
	for _, test := range tests {
		want := contains(test.s)
		got := fn(test.s)
		if got != want {
			t.Errorf("ContainsNonASCII(%q) = %t; want: %t", test.s, got, want)
		}
	}
}

type prefixTest struct {
	s, prefix      string
	out, exhausted bool
}

var prefixTests = []prefixTest{
	{"", "", true, true},
	{"1", "", true, false},
	{"1", "2", false, true},
	{"foo", "f", true, false},
	{"αβδ", "ΑΒΔ", true, true},
	{"αβδΑΒΔ", "ΑΒΔ", true, false},
	{"abc", "xyz", false, false},
	{"abc", "XYZ", false, false},
	{"abc", "abc", true, true},
	{"abc", "abd", false, true},
	{"abcdefghijk", "abcdefghijX", false, true},
	{"abcdefghijk", "abcdefghij\u212A", true, true},
	{"abcdefghijk", "abcdefghij\u212Axyz", false, true},
	{"abcdefghijk☺", "abcdefghij\u212A", true, false},
	{"abcdefghijkz", "abcdefghij\u212Ay", false, true},
	{"abcdefghijKz", "abcdefghij\u212Ay", false, true},
	{"☺aβ", "☺aΔ", false, true},
	{"☺aβc", "☺aΔ", false, false},
	{"\u0250\u0250\u0250\u0250\u0250 a", "\u2C6F\u2C6F\u2C6F\u2C6F\u2C6F A", true, true}, // grows one byte per char
	{"a\u0250\u0250\u0250\u0250\u0250", "A\u2C6F\u2C6F\u2C6F\u2C6F\u2C6F", true, true},   //
	{"\u2C6D\u2C6D\u2C6D\u2C6D\u2C6D a", "\u0251\u0251\u0251\u0251\u0251 A", true, true}, // shrinks one byte per char
	{"a\u2C6D\u2C6D\u2C6D\u2C6D\u2C6D", "A\u0251\u0251\u0251\u0251\u0251", true, true},   // shrinks one byte per char

	// Handle large differences in encoded size ([kK]: 1 vs. 'K' (U+212A): 3 bytes).
	{strings.Repeat("\u212a", 8), strings.Repeat("k", 8), true, true},
	{strings.Repeat("k", 8), strings.Repeat("\u212a", 8), true, true},
	{"k-k", "\u212a-\u212a", true, true},

	{"a", "bbb", false, true},
	{"\u212a", strings.Repeat("a", len("\u212a")*2), false, true},
	{"\u212a", strings.Repeat("a", len("\u212a")*3), false, true},
	{"\u212a", strings.Repeat("a", len("\u212a")*4), false, true},
}

func HasPrefix(t *testing.T, fn PrefixFunc) {
	// Make sure the tests cases are valid
	for i, test := range prefixTests {
		s := []rune(test.s)
		prefix := []rune(test.prefix)
		out, exhausted := HasPrefixRunes(s, prefix)
		if out != test.out || exhausted != test.exhausted {
			t.Errorf("invalid test: %d: %+v", i, test)
		}
		if n := len(prefix); len(s) >= n {
			if out := strings.EqualFold(string(s[:n]), string(prefix)); out != test.out {
				t.Errorf("strings.EqualFold(%q, %q) = %t; want: %t",
					test.s, test.prefix, out, test.out)
			}
		}
	}
	if t.Failed() {
		t.Fatal("Invalid tests cases")
	}

	for _, test := range prefixTests {
		out, exhausted := fn(test.s, test.prefix)
		if out != test.out || exhausted != test.exhausted {
			t.Errorf("hasPrefixUnicode(%q, %q) = %t, %t; want: %t, %t", test.s, test.prefix,
				out, exhausted, test.out, test.exhausted)
			t.Error("s:     ", len(test.s), utf8.RuneCountInString(test.s))
			t.Error("prefix:", len(test.prefix), utf8.RuneCountInString(test.prefix))
		}
	}
}

func TrimPrefix(t *testing.T, fn TrimFunc) {
	for i, test := range prefixTests {
		want := test.s
		if test.out {
			s := []rune(test.s)
			prefix := []rune(test.prefix)
			if len(prefix) <= len(s) {
				want = string(s[len(prefix):])
			}
		}
		got := fn(test.s, test.prefix)
		if got != want {
			t.Errorf("%d: TrimPrefix(%q, %q) = %q; want: %q",
				i, test.s, test.prefix, got, want)
		}
	}
}

type suffixTest struct {
	s, suffix string
	out       bool
}

var suffixTests = []suffixTest{
	{"", "", true},
	{"a", "", true},
	{"", "a", false},
	{"1", "2", false},
	{"αβδ", "ΑΒΔ", true},
	{"αβδΑΒΔ", "ΑΒΔ", true},
	{"abc", "xyz", false},
	{"abc", "XYZ", false},
	{"abc", "abc", true},
	{"abc", "abd", false},
	{"aaβ", "☺aβ", false},
	{"☺aβc", "☺aΔ", false},

	{"\u0250\u0250\u0250\u0250\u0250 a", "\u2C6F\u2C6F\u2C6F\u2C6F\u2C6F A", true}, // grows one byte per char
	{"a\u0250\u0250\u0250\u0250\u0250", "A\u2C6F\u2C6F\u2C6F\u2C6F\u2C6F", true},   //
	{"\u2C6D\u2C6D\u2C6D\u2C6D\u2C6D a", "\u0251\u0251\u0251\u0251\u0251 A", true}, // shrinks one byte per char
	{"a\u2C6D\u2C6D\u2C6D\u2C6D\u2C6D", "A\u0251\u0251\u0251\u0251\u0251", true},   // shrinks one byte per char

	// Handle large differences in encoded size ([kK]: 1 vs. 'K' (U+212A): 3 bytes).
	{strings.Repeat("\u212a", 8), strings.Repeat("k", 8), true},
	{strings.Repeat("k", 8), strings.Repeat("\u212a", 8), true},
	{"k-k", "\u212a-\u212a", true},

	{"g^Y3i", "I", true},
	{"G|S&>;C", "&>;C", true},
}

func HasSuffix(t *testing.T, fn func(s, suffix string) bool) {
	// Make sure the tests cases are valid
	for _, test := range suffixTests {
		out := regexp.MustCompile(`(?i)` + regexp.QuoteMeta(test.suffix) + "$").MatchString(test.s)
		if out != test.out {
			t.Errorf("Invalid test s: %q, suffix: %q got: %t want: %t", test.s, test.suffix, out, test.out)
		}
	}
	if t.Failed() {
		t.Fatal("Invalid tests cases")
	}

	for _, test := range suffixTests {
		out := fn(test.s, test.suffix)
		if out != test.out {
			t.Errorf("HasSuffix(%q, %q) = %t; want: %t", test.s, test.suffix, out, test.out)
		}
	}
}

func TrimSuffix(t *testing.T, fn TrimFunc) {
	for i, test := range suffixTests {
		hasSuffix := test.out && test.suffix != ""
		want := test.s
		if hasSuffix {
			s := []rune(test.s)
			suffix := []rune(test.suffix)
			if len(s) >= len(suffix) {
				want = string(s[:len(s)-len(suffix)])
			}
		}
		got := fn(test.s, test.suffix)
		if got != want {
			t.Errorf("%d: TrimSuffix(%q, %q) = %q; want: %q",
				i, test.s, test.suffix, got, want)
		}
	}
}

var countTests = []struct {
	s, sep string
	num    int
}{
	{"", "", 1},
	{"", "notempty", 0},
	{"notempty", "", 9},
	{"smaller", "not smaller", 0},
	{"12345678987654321", "6", 2},
	{"611161116", "6", 3},
	{"notequal", "NotEqual", 1},
	{"equal", "equal", 1},
	{"abc1231231123q", "123", 3},
	{"11111", "11", 2},
	{"aAaAa", "a", 5},
	{"a\u212akKa", "K", 3},
	{"a\u212akKa", "S", 0},
	{"a\u212a", "a\u212a", 1},
	{"a\u212aa\u212a", "a\u212a", 2},
	{"sſS", "s", 3},
	{strings.Repeat("\u212a", 8), "kk", 4},
	{strings.Repeat("k", 8), "\u212a\u212a", 4},
	{strings.Repeat("\u212a", 32), "kk", 16},
	{strings.Repeat("k", 32), "\u212a\u212a", 16},
}

func Count(t *testing.T, fn IndexFunc) {
	for _, tt := range countTests {
		if num := fn(tt.s, tt.sep); num != tt.num {
			t.Errorf("Count(%q, %q) = %d, want %d", tt.s, tt.sep, num, tt.num)
		}
	}
}

var indexAnyTests = []indexTest{
	{"", "", -1},
	{"", "a", -1},
	{"", "abc", -1},
	{"a", "", -1},
	{"a", "a", 0},
	{"\x80", "\xffb", 0},
	{"aaa", "a", 0},
	{"abc", "xyz", -1},
	{"abc", "xcz", 2},
	{"abc", "XCZ", 2},
	{"abcdefghijklmnop", "xyz", -1},
	{"ab☺c", "x☺yz", 2},
	{"a☺b☻c☹d", "cx", len("a☺b☻")},
	{"a☺b☻c☹d", "uvw☻xyz", len("a☺b")},
	{"aRegExp*", ".(|)*+?^$[]", 7},
	{dots + dots + dots, " ", -1},
	{dots + dots + dots + "a", "A", len(dots + dots + dots)},
	{dots + dots + dots + "\u212a", "k", len(dots + dots + dots)},
	{dots + dots + dots + "a", "Z", -1},
	{"012abcba210", "\xffb", 4},
	{"012\x80bcb\x80210", "\xffb", 3},
	{"0123456\xcf\x80abc", "\xcfb\x80", 10},
	{"a☺b☻c☹d", "☺"[:1], -1},

	// ASCII chars that are equal to multi-byte runes
	{"\u212A" + strings.Repeat("x", 16), "k", 0},
	{strings.Repeat("k", 16), "\u212A", 0},
	{"\u017F" + strings.Repeat("x", 16), "s", 0},
	{strings.Repeat("s", 16), "\u017F", 0},
}

var lastIndexAnyTests = []indexTest{
	{"", "", -1},
	{"", "a", -1},
	{"", "abc", -1},
	{"a", "", -1},
	{"a", "b", -1},
	{"a", "a", 0},
	{"\x80", "\xffb", 0},
	{"aaa", "a", 2},
	{"abc", "xyz", -1},
	{"abc", "ab", 1},
	{"ab☺c", "x☺yz", 2},
	{"a☺b☻c☹d", "cx", len("a☺b☻")},
	{"a☺b☻c☹d", "uvw☻xyz", len("a☺b")},
	{"a.RegExp*", ".(|)*+?^$[]", 8},
	{dots + dots + dots, " ", -1},
	{"012abcba210", "\xffb", 6},
	{"012\x80bcb\x80210", "\xffb", 7},
	{"0123456\xcf\x80abc", "\xcfb\x80", 10},

	// Case-insensitive
	{"a", "A", 0},
	{"a☺b☻c☹d", "CX", len("a☺b☻")},
	{"a☺b☻c☹d", "UVW☻XYZ", len("a☺b")},
	{"kkk", "\u212a", 2},
	{"☹", "☹"[:1], -1},
	{"abc" + "☹"[:1], "☹"[:1], len("abc")},

	// ASCII chars that are equal to multi-byte runes
	{"\u212A" + strings.Repeat("x", 16), "k", 0},
	{strings.Repeat("k", 16), "\u212A", 15},
	{"\u017F" + strings.Repeat("x", 16), "s", 0},
	{strings.Repeat("s", 16), "\u017F", 15},
}

func IndexAny(t *testing.T, fn IndexFunc) {
	runIndexTests(t, fn, "IndexAny", indexAnyTests, false)
}

func LastIndexAny(t *testing.T, fn IndexFunc) {
	runIndexTests(t, fn, "LastIndexAny", lastIndexAnyTests, false)
}

var cutTests = []struct {
	s, sep        string
	before, after string
	found         bool
}{
	{"abc", "b", "a", "c", true},
	{"abc", "a", "", "bc", true},
	{"abc", "c", "ab", "", true},
	{"abc", "abc", "", "", true},
	{"abc", "", "", "abc", true},
	{"abc", "d", "abc", "", false},
	{"", "d", "", "", false},
	{"", "", "", "", true},

	// Unicode
	{"αβδ", "ΑΒΔ", "", "", true},
	{"αβδΑΒΔ", "ΑΒΔ", "", "ΑΒΔ", true},
	{"123αβδ456", "ΑΒΔ", "123", "456", true},
	{"\u212aZZZ\u212aABC", "ZKA", "\u212aZZ", "BC", true},

	// TODO: test invalid UTF-8 sequences
	//
	// {"\xed\xa0\x80", string(utf8.RuneError), "", "", true},
	// {"\xed\xa0\x80", string(utf8.RuneError), string(utf8.RuneError), "\xa0\x80", true},
	// {string(utf8.RuneError), "\xed\xa0\x80", "", "", true},
}

func Cut(t *testing.T, fn func(s, sep string) (before, after string, found bool)) {
	for _, tt := range cutTests {
		before, after, found := fn(tt.s, tt.sep)
		if before != tt.before || after != tt.after || found != tt.found {
			t.Errorf("Cut(%q, %q) = %q, %q, %v; want: %q, %q, %v",
				tt.s, tt.sep, before, after, found, tt.before, tt.after, tt.found)
		}
	}
}

var cutPrefixTests = []struct {
	s, sep string
	after  string
	found  bool
}{
	{"abc", "a", "bc", true},
	{"abc", "abc", "", true},
	{"abc", "", "abc", true},
	{"abc", "d", "abc", false},
	{"", "d", "", false},
	{"", "", "", true},

	// Unicode
	{"αβδ", "ΑΒΔ", "", true},
	{"αβδΑΒΔ", "ΑΒΔ", "ΑΒΔ", true},
	{"123αβδ456", "ΑΒΔ", "123αβδ456", false},
	{"kk123", "\u212a\u212a123", "", true},
	{"kk123xyz", "\u212a\u212a123", "xyz", true},
	{"\u212a\u212a123xyz", "kK123", "xyz", true},
}

func CutPrefix(t *testing.T, fn func(s, prefix string) (after string, found bool)) {
	for _, tt := range cutPrefixTests {
		after, found := fn(tt.s, tt.sep)
		if after != tt.after || found != tt.found {
			t.Errorf("CutPrefix(%q, %q) = %q, %v, want %q, %v",
				tt.s, tt.sep, after, found, tt.after, tt.found)
		}
	}
}

var cutSuffixTests = []struct {
	s, sep string
	after  string
	found  bool
}{
	{"abc", "bc", "a", true},
	{"abc", "abc", "", true},
	{"abc", "", "abc", true},
	{"abc", "d", "abc", false},
	{"", "d", "", false},
	{"", "", "", true},

	// Unicode
	{"αβδ", "ΑΒΔ", "", true},
	{"αβδΑΒΔ", "ΑΒΔ", "αβδ", true},
	{"123αβδ456", "ΑΒΔ", "123αβδ456", false},
	{"kk123", "\u212a\u212a123", "", true},
	{"xyzkK123", "\u212a\u212a123", "xyz", true},
}

func CutSuffix(t *testing.T, fn func(s, prefix string) (before string, found bool)) {
	for _, tt := range cutSuffixTests {
		after, found := fn(tt.s, tt.sep)
		if after != tt.after || found != tt.found {
			t.Errorf("CutSuffix(%q, %q) = %q, %v, want %q, %v",
				tt.s, tt.sep, after, found, tt.after, tt.found)
		}
	}
}

// Helper functions
////////////////////////////////////////////////////////////////////////////////

// IndexRunesReference is a slow, but accurate case-insensitive version of strings.Index
func IndexRunesReference(s, sep []rune) int {
	// TODO: The allocations here count for a lot of the test time so
	// try to do this without allocating (aka compare the rune slices).
	if len(s) < len(sep) {
		return -1
	}
	if len(s) == len(sep) {
		if strings.EqualFold(string(s), string(sep)) {
			return 0
		}
		return -1
	}
	rs := append([]rune(nil), s...)
	rsep := append([]rune(nil), sep...)
	for i := 0; i < len(rs); i++ {
		rs[i] = tables.CaseFold(rs[i])
	}
	for i := 0; i < len(rsep); i++ {
		rsep[i] = tables.CaseFold(rsep[i])
	}
	ss := string(rs)
	i := strings.Index(ss, string(rsep))
	if i < 0 {
		return i
	}
	// Case fold conversion can change string length so
	// figure out the index into the original string s.
	n := utf8.RuneCountInString(ss[:i])
	return len(string(s[:n]))
}

func encodedLen(rs []rune) int {
	i := 0
	for _, r := range rs {
		i += utf8.RuneLen(r)
	}
	return i
}

// LastIndexRunesReference is a slow, but accurate case-insensitive version of strings.Index
func LastIndexRunesReference(s, sep []rune) int {
	n := len(sep)
	switch {
	case n == 0:
		return encodedLen(s)
	case n == len(s):
		if strings.EqualFold(string(s), string(sep)) {
			return 0
		}
		return -1
	case n > len(s):
		return -1
	default:
		rs := append([]rune(nil), s...)
		rsep := append([]rune(nil), sep...)
		for i := 0; i < len(rs); i++ {
			rs[i] = tables.CaseFold(rs[i])
		}
		for i := 0; i < len(rsep); i++ {
			rsep[i] = tables.CaseFold(rsep[i])
		}
		ss := string(rs)
		i := strings.LastIndex(ss, string(rsep))
		if i < 0 {
			return i
		}
		// Case fold conversion can change string length so
		// figure out the index into the original string s.
		n := utf8.RuneCountInString(ss[:i])
		return len(string(s[:n]))
	}
}

func HasPrefixRunes(s, prefix []rune) (bool, bool) {
	if len(s) < len(prefix) {
		return false, true
	}
	var i int
	for i = 0; i < len(prefix); i++ {
		sr := tables.CaseFold(s[i])
		pr := tables.CaseFold(prefix[i])
		if !utf8.ValidRune(sr) {
			sr = utf8.RuneError
		}
		if !utf8.ValidRune(pr) {
			pr = utf8.RuneError
		}
		if sr == pr {
			continue
		}
		return false, i == len(s)-1
	}
	return i == len(prefix), i == len(s)
}
