// Copyright 2023 Charlie Vieth. All rights reserved.
// Use of this source code is governed by the MIT license.

package strcase

import (
	"flag"
	"fmt"
	"math/rand"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"testing"
	"unicode"
	"unicode/utf8"

	"golang.org/x/text/unicode/rangetable"
)

func TestUnicodeVersion(t *testing.T) {
	if UnicodeVersion != unicode.Version {
		t.Fatalf("unicode.Version (%s) != UnicodeVersion (%s):\n"+
			"The version of Unicode included in the version of Go (%s) running this test\n"+
			"does not match the Unicode version the strcase tables were generated with.\n"+
			"\n"+
			"This is likely due to the Unicode version being updated in a newer Go release.\n"+
			"To regenerate the Unicode tables run: `go generate` and check in the changes to\n"+
			"\"tables.go\" and \".tables.json\".\n"+
			"\n"+
			"NOTE: re-generating the Unicode tables can take a few minutes.",
			unicode.Version, UnicodeVersion, runtime.Version())
	}
}

type CompareTest struct {
	s, t string
	out  int
}

var compareTests = []CompareTest{
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
}

func TestCompare(t *testing.T) {
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

	for _, test := range compareTests {
		got := Compare(test.s, test.t)
		if got != test.out {
			t.Errorf("Compare(%q, %q) = %d; want: %d", test.s, test.t, got, test.out)
		}
	}
}

func TestEqualFold(t *testing.T) {
	for _, test := range compareTests {
		want := test.out == 0
		got := EqualFold(test.s, test.t)
		if got != want {
			t.Errorf("EqualFold(%q, %q) = %t; want: %t", test.s, test.t, got, want)
		}
	}
}

type IndexTest struct {
	s   string
	sep string
	out int
}

func filterIndexTests(fn func(t IndexTest) bool, tests ...[]IndexTest) []IndexTest {
	var out []IndexTest
	for _, a := range tests {
		for _, t := range a {
			if fn == nil || fn(t) {
				out = append(out, t)
			}
		}
	}
	return out
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
	{strings.Repeat("ox", 64) + "yox", "oα" + strings.Repeat("ox", maxLen/len("ox")), -1},
	{strings.Repeat("ox", 64) + "oα" + strings.Repeat("ox", maxLen/2), "oα" + strings.Repeat("ox", maxLen/2), 128},

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
	// Test cases found by fuzzing
	{"\x00iK", "iK", 1},
	{"İKKKK\x00iK", "iK", 15},
	{"İKKKKiK", "iK", 14},

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
		unicodeIndexTests = append(unicodeIndexTests, IndexTest{s0, s1, 0}, IndexTest{s1, s0, 0})
	}
}

var lastIndexTests = []IndexTest{
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
func runIndexTests(t *testing.T, f func(s, sep string) int, funcName string, testCases []IndexTest, noError bool) {
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
				// TODO: delete me
				// _, foldable = _FoldMap[r]
				foldable := foldMap(r) != nil
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

// Reference test using regex: this will identify bad test cases and is more
// accurate than our reference Index (since it might have bugs).
func TestIndexRegex(t *testing.T) {
	index := func(s, sep string) int {
		i := regexp.MustCompile(`(?i)` + regexp.QuoteMeta(sep)).FindStringIndex(s)
		if len(i) == 2 {
			return i[0]
		}
		return -1
	}
	tests := filterIndexTests(nil, indexTests, unicodeIndexTests)
	runIndexTests(t, index, "Regexp", tests, false)
}

func TestIndex(t *testing.T) {
	tests := filterIndexTests(nil, indexTests, unicodeIndexTests)

	// Test that the Index tests are valid
	reference := func(s, sep string) int {
		return indexRunesReference([]rune(s), []rune(sep))
	}
	runIndexTests(t, reference, "IndexReference", tests, false)
	if t.Failed() {
		t.Fatal("Reference Index function failed: tests are invalid")
	}
	runIndexTests(t, Index, "Index", tests, false)
}

// Extensively test the handling of Kelvin K since it is three times the size
// of ASCII [Kk] it requires special handling.
func TestIndexKelvin(t *testing.T) {
	const K = "\u212A" // Kelvin

	test := func(t *testing.T, s, substr string, want int) {
		t.Helper()
		if got := Index(s, substr); got != want {
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

func TestContains(t *testing.T) {
	for _, test := range indexTests {
		got := Contains(test.s, test.sep)
		want := test.out >= 0
		if got != want {
			t.Errorf("Contains(%q, %q) = %t; want: %t", test.s, test.sep, got, want)
		}
	}
}

const a32 = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa" // "a" repeated 32 times

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

func TestContainsAny(t *testing.T) {
	for _, ct := range ContainsAnyTests {
		if ContainsAny(ct.str, ct.substr) != ct.expected {
			t.Errorf("ContainsAny(%s, %s) = %v, want %v",
				ct.str, ct.substr, !ct.expected, ct.expected)
		}
	}
}

// Test that the Rabin-Karp functions can handle a haystack (s) that is
// smalled than the needle (sep).
func TestIndexRabinKarpUnicode(t *testing.T) {
	test := func(name string, fn func(s, substr string) int) {
		i := fn("aa", "aaaa")
		if i != -1 {
			t.Fatalf("%s(%q, %q) = %d; want: %d", name, "aa", "aaaa", i, -1)
		}
	}
	test("indexRabinKarpUnicode", indexRabinKarpUnicode)
	test("indexRabinKarpRevUnicode", indexRabinKarpRevUnicode)
}

func TestIndexAllocs(t *testing.T) {
	haystack := "test世界İ"
	allocs := testing.AllocsPerRun(1000, func() {
		if i := Index(haystack, "世界İ"); i != 4 {
			t.Fatalf("'s' at %d; want 4", i)
		}
		if i := Index(haystack, "t世"); i != 3 {
			t.Fatalf("'世' at %d; want 3", i)
		}
		if i := Index(haystack, "test世界İ"); i != 0 {
			t.Fatalf("'İ' at %d; want 0", i)
		}
	})
	if allocs != 0 && testing.CoverMode() == "" {
		t.Errorf("expected no allocations, got %f", allocs)
	}
}

func TestIndexUnicode(t *testing.T) {
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

			tests := append([]IndexTest(nil), indexTests...)
			for i, test := range tests {
				if test.out > 0 {
					test.out = len(r(test.s[:test.out]))
				}
				test.s = r(test.s)
				test.sep = r(test.sep)
				tests[i] = test
			}

			t.Run("Index", func(t *testing.T) {
				runIndexTests(t, Index, "Index", tests, false)
			})

			// TODO: can probably remove this
			t.Run("RabinKarp", func(t *testing.T) {
				fn := func(t IndexTest) bool {
					return len(t.sep) > 0 && len(t.s) > len(t.sep)
				}
				rtests := filterIndexTests(fn, tests)
				runIndexTests(t, indexRabinKarpUnicode, "indexRabinKarpUnicode", rtests, false)
			})
		})
	}
}

func TestLastIndex(t *testing.T) {
	reference := func(s, sep string) int {
		return lastIndexRunesReference([]rune(s), []rune(sep))
	}
	runIndexTests(t, reference, "LastIndexReference", lastIndexTests, false)

	runIndexTests(t, LastIndex, "LastIndex", lastIndexTests, false)
}

type IndexRuneTest struct {
	in   string
	rune rune
	want int
}

var indexRuneTests = []IndexRuneTest{
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
	{"αβδ", 'Α', 0}, // "ΑΒΔ"
	{"αβδ", 'Δ', 4}, // "ΑΒΔ"

	// Case-folding with ASCII
	{"K", 'K', 0},  // U+212A
	{"S", 'ſ', 0},  // U+017F
	{"K", 'k', 0},  // U+006B
	{"ſ", 's', 0},  // U+0073
	{"İ", 'İ', 0},  // U+0130
	{"i", 'İ', -1}, // U+0130
	{"ſS*ք", 'S', 0},
}

func TestIndexRune(t *testing.T) {
	for _, tt := range indexRuneTests {
		if got := IndexRune(tt.in, tt.rune); got != tt.want {
			t.Errorf("IndexRune(%q, %q) = %v; want %v", tt.in, tt.rune, got, tt.want)
		}
	}

	haystack := "test世界İ"
	allocs := testing.AllocsPerRun(1000, func() {
		if i := IndexRune(haystack, 's'); i != 2 {
			t.Fatalf("'s' at %d; want 2", i)
		}
		if i := IndexRune(haystack, '世'); i != 4 {
			t.Fatalf("'世' at %d; want 4", i)
		}
		if i := IndexRune(haystack, 'İ'); i != 10 {
			t.Fatalf("'İ' at %d; want 10", i)
		}
	})
	if allocs != 0 && testing.CoverMode() == "" {
		t.Errorf("expected no allocations, got %f", allocs)
	}
}

func TestIndexRuneCase(t *testing.T) {
	tests := []IndexRuneTest{
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
		if got := indexRuneCase(tt.in, tt.rune); got != tt.want {
			t.Errorf("indexRuneCase(%q, %d) = %v; want %v", tt.in, tt.rune, got, tt.want)
		}
	}
}

func TestContainsRune(t *testing.T) {
	for _, test := range indexRuneTests {
		got := ContainsRune(test.in, test.rune)
		want := test.want >= 0
		if got != want {
			t.Errorf("ContainsRune(%q, 0x%04X) = %t; want: %t", test.in, test.rune, got, want)
		}
	}
}

func TestLastIndexRune(t *testing.T) {
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
		if got := lastIndexRune(tt.in, tt.rune); got != tt.want {
			t.Errorf("lastIndexRune(%q, %q) = %v; want %v", tt.in, tt.rune, got, tt.want)
		}
	}
}

func TestIndexByte(t *testing.T) {
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
		if got := IndexByte(tt.in, tt.char); got != tt.want {
			t.Errorf("IndexByte(%q, %q) = %v; want %v", tt.in, tt.char, got, tt.want)
		}
	}
}

func TestLastIndexByte(t *testing.T) {
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
		if got := LastIndexByte(tt.in, tt.char); got != tt.want {
			t.Errorf("LastIndexByte(%q, %q) = %v; want %v", tt.in, tt.char, got, tt.want)
		}
	}
}

func TestIndexNonASCII(t *testing.T) {
	index := func(s string) int {
		for i, r := range s {
			if r >= utf8.RuneSelf {
				return i
			}
		}
		return -1
	}

	t.Run("IndexTests", func(t *testing.T) {
		tests := append([]IndexTest(nil), indexTests...)
		tests = append(tests, unicodeIndexTests...)
		for _, test := range tests {
			want := index(test.s)
			got := IndexNonASCII(test.s)
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
			got := IndexNonASCII(s)
			if got != want {
				t.Errorf("IndexNonASCII(long[%d:]) = %d; want: %d", i, got, want)
			}
		}
	})
}

func TestContainsNonASCII(t *testing.T) {
	contains := func(s string) bool {
		for i := 0; i < len(s); i++ {
			if s[i] >= utf8.RuneSelf {
				return true
			}
		}
		return false
	}

	tests := append([]IndexTest(nil), indexTests...)
	tests = append(tests, unicodeIndexTests...)
	for _, test := range tests {
		want := contains(test.s)
		got := ContainsNonASCII(test.s)
		if got != want {
			t.Errorf("ContainsNonASCII(%q) = %t; want: %t", test.s, got, want)
		}
	}
}

type PrefixTest struct {
	s, prefix      string
	out, exhausted bool
}

var prefixTests = []PrefixTest{
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

func TestHasPrefix(t *testing.T) {
	// Make sure the tests cases are valid
	for i, test := range prefixTests {
		s := []rune(test.s)
		prefix := []rune(test.prefix)
		out, exhausted := hasPrefixRunes(s, prefix)
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
		out, exhausted := hasPrefixUnicode(test.s, test.prefix)
		if out != test.out || exhausted != test.exhausted {
			t.Errorf("hasPrefixUnicode(%q, %q) = %t, %t; want: %t, %t", test.s, test.prefix,
				out, exhausted, test.out, test.exhausted)
			t.Error("s:     ", len(test.s), utf8.RuneCountInString(test.s))
			t.Error("prefix:", len(test.prefix), utf8.RuneCountInString(test.prefix))
		}
	}
}

func TestTrimPrefix(t *testing.T) {
	for i, test := range prefixTests {
		want := test.s
		if test.out {
			s := []rune(test.s)
			prefix := []rune(test.prefix)
			if len(prefix) <= len(s) {
				want = string(s[len(prefix):])
			}
		}
		got := TrimPrefix(test.s, test.prefix)
		if got != want {
			t.Errorf("%d: TrimPrefix(%q, %q) = %q; want: %q",
				i, test.s, test.prefix, got, want)
		}
	}
}

type SuffixTest struct {
	s, suffix string
	out       bool
}

var suffixTests = []SuffixTest{
	{"", "", true /*, true*/},
	{"a", "", true /*, false*/},
	{"", "a", false /*, true*/},
	{"1", "2", false /*, true*/},
	{"αβδ", "ΑΒΔ", true /*, true*/},
	{"αβδΑΒΔ", "ΑΒΔ", true /*, false*/},
	{"abc", "xyz", false /*, false*/},
	{"abc", "XYZ", false /*, false*/},
	{"abc", "abc", true /*, true*/},
	{"abc", "abd", false /*, false*/},
	{"aaβ", "☺aβ", false /*, true*/},
	{"☺aβc", "☺aΔ", false /*, false*/},

	{"\u0250\u0250\u0250\u0250\u0250 a", "\u2C6F\u2C6F\u2C6F\u2C6F\u2C6F A", true /*, true*/}, // grows one byte per char
	{"a\u0250\u0250\u0250\u0250\u0250", "A\u2C6F\u2C6F\u2C6F\u2C6F\u2C6F", true /*, true*/},   //
	{"\u2C6D\u2C6D\u2C6D\u2C6D\u2C6D a", "\u0251\u0251\u0251\u0251\u0251 A", true /*, true*/}, // shrinks one byte per char
	{"a\u2C6D\u2C6D\u2C6D\u2C6D\u2C6D", "A\u0251\u0251\u0251\u0251\u0251", true /*, true*/},   // shrinks one byte per char

	// Handle large differences in encoded size ([kK]: 1 vs. 'K' (U+212A): 3 bytes).
	{strings.Repeat("\u212a", 8), strings.Repeat("k", 8), true /*, true*/},
	{strings.Repeat("k", 8), strings.Repeat("\u212a", 8), true /*, true*/},
	{"k-k", "\u212a-\u212a", true /*, true*/},

	{"g^Y3i", "I", true /*, false*/},
	{"G|S&>;C", "&>;C", true /*, false*/},
}

func TestHasSuffix(t *testing.T) {
	// Make sure the tests cases are valid
	for _, test := range suffixTests {
		out := hasSuffixRunes([]rune(test.s), []rune(test.suffix))
		if out != test.out {
			t.Errorf("hasSuffixRunes(%q, %q) = %t; want: %t", test.s, test.suffix, out, test.out)
		}
	}
	if t.Failed() {
		t.Fatal("Invalid tests cases")
	}

	for _, test := range suffixTests {
		out := HasSuffix(test.s, test.suffix)
		if out != test.out {
			t.Errorf("HasSuffix(%q, %q) = %t; want: %t", test.s, test.suffix, out, test.out)
		}
	}
}

func TestTrimSuffix(t *testing.T) {
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
		got := TrimSuffix(test.s, test.suffix)
		if got != want {
			t.Errorf("%d: TrimSuffix(%q, %q) = %q; want: %q",
				i, test.s, test.suffix, got, want)
		}
	}
}

func TestToUpperLower(t *testing.T) {
	fails := 0
	rangetable.Visit(unicodeCategories, func(r rune) {
		l := unicode.ToLower(r)
		u := unicode.ToUpper(r)
		ok := l != u
		uu, ll, found := toUpperLower(r)
		if l != ll || u != uu || ok != found {
			t.Errorf("toUpperLower(%c) = %c, %c, %t want: %c, %c, %t",
				r, ll, uu, found, l, u, ok)
			fails++
		}
		if fails >= 50 {
			t.Fatal("Too many errors:", fails)
		}
	})
}

var CountTests = []struct {
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
}

func TestCount(t *testing.T) {
	for _, tt := range CountTests {
		if num := Count(tt.s, tt.sep); num != tt.num {
			t.Errorf("Count(%q, %q) = %d, want %d", tt.s, tt.sep, num, tt.num)
		}
	}
}

var dots = "1....2....3....4"

var indexAnyTests = []IndexTest{
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

var lastIndexAnyTests = []IndexTest{
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

func TestIndexAny(t *testing.T) {
	runIndexTests(t, IndexAny, "IndexAny", indexAnyTests, false)
}

func TestLastIndexAny(t *testing.T) {
	runIndexTests(t, LastIndexAny, "LastIndexAny", lastIndexAnyTests, false)
}

func BenchmarkCompare(b *testing.B) {
	b.Run("Tests", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			for _, tt := range compareTests {
				if out := Compare(tt.s, tt.t); out != tt.out {
					b.Fatal("wrong result")
				}
			}
		}
	})

	bench := func(b *testing.B, s, t string) {
		b.Helper()
		n := len(s)
		if len(t) < n {
			n = len(t)
		}
		b.SetBytes(int64(n))
		for i := 0; i < b.N; i++ {
			Compare(s, t)
		}
	}

	const s1 = "abcdefghijKz"
	const s2 = "abcDefGhijKz"

	b.Run("ASCII", func(b *testing.B) {
		bench(b, s1, s2)
	})

	b.Run("ASCII_Long", func(b *testing.B) {
		const s = s1 + s1 + s1 + s1 + s1
		const t = s2 + s2 + s2 + s2 + s2
		bench(b, s, t)
	})

	b.Run("UnicodePrefix", func(b *testing.B) {
		// WARN
		const s1 = "AbCdCfghIjKz"
		const s2 = "abcDeFGhijKz"
		bench(b, "αβδ"+s1, "ΑΒΔ"+s2)
	})

	b.Run("UnicodeSuffix", func(b *testing.B) {
		bench(b, s1+"αβδ", s2+"ΑΒΔ")
	})

	b.Run("Russian", func(b *testing.B) {
		b.SetBytes(int64(len(russianText)))
		bench(b, russianText, russianText)
	})
}

func TestCaseFold(t *testing.T) {
	t.Run("Limits", func(t *testing.T) {
		for r := unicode.MaxRune; r < unicode.MaxRune+10; r++ {
			x := caseFold(r)
			if x != r {
				t.Errorf("caseFold(0x%04X) = 0x%04X; want: 0x%04X", r, x, r)
			}
		}
		for r := rune(0); r < ' '; r++ {
			x := caseFold(r)
			if x != r {
				t.Errorf("caseFold(0x%04X) = 0x%04X; want: 0x%04X", r, x, r)
			}
		}
		if r := caseFold(utf8.RuneError); r != utf8.RuneError {
			t.Errorf("caseFold(0x%04X) = 0x%04X; want: 0x%04X", utf8.RuneError, r, utf8.RuneError)
		}
	})
	t.Run("ValidFolds", func(t *testing.T) {
		for _, p := range _CaseFolds {
			if r := caseFold(rune(p.From)); r != rune(p.To) {
				t.Errorf("caseFold(0x%04X) = 0x%04X; want: 0x%04X", rune(p.From), r, rune(p.To))
			}
		}
	})
	t.Run("UnicodeCases", func(t *testing.T) {
		folds := make(map[rune]rune)
		for _, p := range _CaseFolds {
			if p.From != 0 {
				folds[rune(p.From)] = rune(p.To)
			}
		}
		rangetable.Visit(unicodeCategories, func(r rune) {
			if rr, ok := folds[r]; ok {
				r = rr
			}
			if got := caseFold(r); got != r {
				t.Errorf("caseFold(0x%04X) = 0x%04X; want: 0x%04X", r, got, r)
			}
		})
	})
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
}

func TestCut(t *testing.T) {
	for _, tt := range cutTests {
		before, after, found := Cut(tt.s, tt.sep)
		if before != tt.before || after != tt.after || found != tt.found {
			t.Errorf("Cut(%q, %q) = %q, %q, %v, want %q, %q, %v",
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

func TestCutPrefix(t *testing.T) {
	for _, tt := range cutPrefixTests {
		after, found := CutPrefix(tt.s, tt.sep)
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

func TestCutSuffix(t *testing.T) {
	for _, tt := range cutSuffixTests {
		after, found := CutSuffix(tt.s, tt.sep)
		if after != tt.after || found != tt.found {
			t.Errorf("CutSuffix(%q, %q) = %q, %v, want %q, %v",
				tt.s, tt.sep, after, found, tt.after, tt.found)
		}
	}
}

// Ensure that strings.EqualFold does not match 'İ' (U+0130) and ASCII 'i' or 'I'.
// This is mostly a sanity check.
func TestLatinCapitalLetterIWithDotAbove(t *testing.T) {
	if strings.EqualFold("İ", "i") {
		t.Errorf("strings.EqualFold(%q, %q) = true; want: false", "İ", "i")
	}
	if strings.EqualFold("İ", "I") {
		t.Errorf("strings.EqualFold(%q, %q) = true; want: false", "İ", "I")
	}
	if Compare("İ", "i") == 0 {
		t.Errorf("Compare(%q, %q) = true; want: false", "İ", "i")
	}
	if Compare("İ", "I") == 0 {
		t.Errorf("Compare(%q, %q) = true; want: false", "İ", "I")
	}
}

const benchmarkString = "some_text=some☺value"

const benchmarkString = "some_text=some☺value"

// WARN: dev only
func BenchmarkIndexRuneRussian(b *testing.B) {
	want := strings.IndexRune(russianText, 'ж')
	if got := IndexRune(russianText, 'ж'); got != want {
		b.Fatalf("got: %d want: %d", got, want)
	}
	b.SetBytes(int64(len(russianText)))
	for i := 0; i < b.N; i++ {
		IndexRune(russianText, 'ж')
	}
}

func BenchmarkIndexRune(b *testing.B) {
	// const str = benchmarkString + "\u212a"
	const str = benchmarkString + "k"
	// const str = benchmarkString + string(rune(0x212A))
	// if got := IndexRune(benchmarkString, '☺'); got != 14 {
	if got := IndexRune(str, rune(0x212A)); got != 22 {
		b.Fatalf("wrong index: expected 14, got=%d", got)
	}
	for i := 0; i < b.N; i++ {
		IndexRune(benchmarkString, '☺')
	}
}

func BenchmarkIndexRuneFastPath(b *testing.B) {
	if got := IndexRune(benchmarkString, 'v'); got != 17 {
		b.Fatalf("wrong index: expected 17, got=%d", got)
	}
	for i := 0; i < b.N; i++ {
		IndexRune(benchmarkString, 'v')
	}
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

var indexSizes = []int{10, 32, 4 << 10, 4 << 20, 64 << 20}

func benchBytesUnicode(b *testing.B, sizes []int, f func(b *testing.B, n int, s string)) {
	// These character all have the same second byte (0x90)
	const _s = "𐀀𐀁𐀂𐀃𐀄𐀅𐀆𐀇𐀈𐀉𐀊𐀋𐀍𐀎𐀏𐀐𐀑𐀒𐀓𐀔𐀕𐀖𐀗𐀘𐀙𐀚𐀛𐀜𐀝𐀞𐀟𐀠"
	const s = _s + _s + _s + _s + _s + _s + _s + _s + _s + _s + _s + _s + _s + _s + _s + _s // 2048
	for _, n := range sizes {
		b.Run(valName(n), func(b *testing.B) {
			if len(bmbuf) < n {
				bmbuf = make([]byte, n)
			}
			p := bmbuf
			for len(p) > 0 {
				i := copy(p, s)
				p = p[i:]
			}
			copy(bmbuf[len(bmbuf)-len("𐀤"):], "𐀤")
			b.SetBytes(int64(n))
			f(b, n, string(bmbuf))
		})
	}
}

func bmIndexRune(index func(string, rune) int) func(b *testing.B, n int, s string) {
	return func(b *testing.B, n int, s string) {
		for i := 0; i < b.N; i++ {
			j := index(s, '𐀤')
			if j != n-4 {
				b.Fatal("bad index", j)
			}
		}
	}
}

// Torture test IndexRune. This is useful for calculating the cutover
// for when we should switch to strings.Index in indexRuneCase.
func BenchmarkIndexRuneTorture_Bytes(b *testing.B) {
	b.Log("WARN: this only tests runes that are 4 bytes!")
	if *benchStdLib {
		benchBytesUnicode(b, indexSizes, bmIndexRune(strings.IndexRune))
	} else {
		benchBytesUnicode(b, indexSizes, bmIndexRune(IndexRune))
	}
}

func BenchmarkIndexByte(b *testing.B) {
	const ch = 'V'
	if got := IndexByte(benchmarkString, ch); got != 17 {
		b.Fatalf("wrong index: expected 17, got=%d", got)
	}
	b.SetBytes(int64(len(benchmarkString)))
	for i := 0; i < b.N; i++ {
		IndexByte(benchmarkString, ch)
	}
}

func BenchmarkIndexByteEmpty(b *testing.B) {
	const ch = 'V'
	for i := 0; i < b.N; i++ {
		IndexByte("", ch)
	}
}

// Benchmark the handling of [KkSs] which require a check for their
// equivalent Unicode folds.
func BenchmarkIndexByteLongSpecial(b *testing.B) {
	var bmbuf []byte

	bmIndexByte := func(index func(string, byte) int) func(b *testing.B, n int) {
		return func(b *testing.B, n int) {
			buf := bmbuf[0:n]
			buf[n/2] = 's'
			copy(buf[n-2:], "ſ")
			s := string(buf)
			// We scan the first half of the string twice but the match occurs
			// in the first half so using that index here seems more fair than
			// using the full length of the string as number of bytes processed.
			b.SetBytes(int64(index(s, 's')))
			for i := 0; i < b.N; i++ {
				j := index(s, 's')
				if j != n/2 {
					b.Fatal("bad index", j)
				}
			}
			buf[n/2] = '\x00'
			buf[n-2] = '\x00'
			buf[n-1] = '\x00'
		}
	}

	benchBytes := func(b *testing.B, sizes []int, f func(b *testing.B, n int)) {
		for _, n := range sizes {
			b.Run(valName(n), func(b *testing.B) {
				if len(bmbuf) < n {
					bmbuf = make([]byte, n)
				}
				f(b, n)
			})
		}
	}

	benchBytes(b, indexSizes, bmIndexByte(IndexByte))
}

func BenchmarkLastIndexByte(b *testing.B) {
	if testing.Short() {
		b.Skip("short test")
	}
	const ch = 'S'
	if got := LastIndexByte(benchmarkString, ch); got != 10 {
		b.Fatalf("wrong index: expected 10, got=%d", got)
	}
	s := "b" + strings.Repeat("a", 128)
	c := byte('B')
	for i := 0; i < b.N; i++ {
		LastIndexByte(s, c)
	}
}

// WARN
var benchStdLib = flag.Bool("stdlib", false, "Use strings.Index in benchmarks (for comparison)")

// WARN: this is not really fair because of strings.ToLower
func benchmarkIndex(b *testing.B, s, substr string) {
	if *benchStdLib {
		n := strings.Index(strings.ToLower(s), strings.ToLower(substr))
		if o := Index(s, substr); n != o {
			b.Errorf("strings.Index(%q, %q) = %d; want: %d", s, substr, n, o)
		}
		if n >= 0 {
			b.SetBytes(int64(len(s) + len(substr)))
		} else {
			b.SetBytes(int64(len(s)))
		}
		for i := 0; i < b.N; i++ {
			strings.Index(strings.ToLower(s), strings.ToLower(substr))
		}
	} else {
		if n := Index(s, substr); n >= 0 {
			b.SetBytes(int64(len(s) + len(substr)))
		} else {
			b.SetBytes(int64(len(s)))
		}
		for i := 0; i < b.N; i++ {
			Index(s, substr)
		}
	}
}

func BenchmarkIndex(b *testing.B) {
	if got := Index(benchmarkString, "v"); got != 17 {
		b.Fatalf("wrong index: expected 17, got=%d", got)
	}
	benchmarkIndex(b, benchmarkString, "v")
}

func BenchmarkLastIndex(b *testing.B) {
	if got := LastIndex(benchmarkString, "v"); got != 17 {
		b.Fatalf("wrong index: expected 17, got=%d", got)
	}
	for i := 0; i < b.N; i++ {
		LastIndex(benchmarkString, "v")
	}
}

// Thanks to variable length encoding it's possible the needle
// to be larger than the haystack.
func BenchmarkLastIndexNeedleExceedsHaystack(b *testing.B) {
	s := strings.Repeat("ab", 1024)
	substr := "z" + s
	i1 := strings.LastIndex(s, substr)
	i2 := LastIndex(s, substr)
	if i1 != i2 {
		b.Fatalf("wrong index: expected: %d, got: %d", i1, i2)
	}
	// Can't compare perf to the stdlib because we have to scan
	// the whole string and not just bail at the length mismatch.
	b.SetBytes(int64(len(s)))
	for i := 0; i < b.N; i++ {
		LastIndex(s, substr)
	}
}

func BenchmarkIndexNeedleLongerThanSubject(b *testing.B) {
	const s = benchmarkString
	b.Run("FirstRuneEqual", func(b *testing.B) {
		substr := s + "-"
		benchmarkIndex(b, s, substr)
	})
	b.Run("FirstRuneNotEqual", func(b *testing.B) {
		substr := "-" + s
		benchmarkIndex(b, s, substr)
	})
}

const russianText = `Владимир Маяковский родился в селе Багдади[10] Кутаисской
	губернии Российской империи, в обедневшей дворянской семье[11] Владимира
	Константиновича Маяковского (1857—1906), служившего лесничим третьего
	разряда в Эриванской губернии, а с 1889 г. — в Багдатском лесничестве.
	Маяковский вёл род от запорожских казаков, прадед отца поэта Кирилл
	Маяковский был полковым есаулом Черноморских войск, что дало ему право
	получить звание дворянина[12]. Мать поэта, Александра Алексеевна Павленко
	(1867−1954), из рода кубанских казаков, родилась на Кубани, в станице
	Терновской. В поэме «Владикавказ — Тифлис» 1924 года Маяковский называет
	себя «грузином». О себе Маяковский сказал в 1927 году: «Родился я в
	1894[13] году на Кавказе. Отец был казак, мать — украинка. Первый язык —
	грузинский. Так сказать, между тремя культурами» (из интервью пражской
	газете «Prager Presse»)[14]. Бабушка по отцовской линии, Ефросинья Осиповна
	Данилевская, — двоюродная сестра автора исторических романов Г. П.
	Данилевского, родом из запорожских казаков. У Маяковского было две сестры:
	Людмила (1884—1972) и Ольга (1890—1949) и два брата: Константин (умер в
	трёхлетнем возрасте от скарлатины) и Александр (умер во младенчестве).`

var (
	russianUpper = strings.ToUpper(russianText)
	russianLower = strings.ToLower(russianText)
)

func BenchmarkIndexRussian(b *testing.B) {
	benchmarkIndex(b, russianText, "младенчестве")
}

// Pathological worst-case.
func BenchmarkIndexLateMatchLargeNeedle(b *testing.B) {
	bench := func(b *testing.B, s1, s2, s3 string) {
		m := strings.Repeat(s1, 100/len(s1))
		haystack := strings.Repeat(m+s2, 300) + m + s3
		needle := m + s3
		benchmarkIndex(b, haystack, needle)
	}
	b.Run("Latin", func(b *testing.B) {
		bench(b, "AB", "C", "D")
	})
	b.Run("Cyrillic", func(b *testing.B) {
		bench(b, "А̀ВЄ", "Ж", "Њ")
	})
	b.Run("Han", func(b *testing.B) {
		bench(b, "遠方", "來", "矣")
	})
}

// Pathological worst-case. Consistency here is a good thing.
func BenchmarkIndexLateMatchSmallNeedle(b *testing.B) {
	bench := func(b *testing.B, s1, s2 string) {
		s := strings.Repeat(s1, 1_000/len(s1)) + s2
		rs := []rune(s)
		for i := 2; i <= 64; i *= 2 {
			b.Run(strconv.Itoa(i), func(b *testing.B) {
				benchmarkIndex(b, s, string(rs[len(rs)-i:]))
			})
		}
	}
	b.Run("Numeric", func(b *testing.B) {
		bench(b, "123", "4")
	})
	b.Run("Latin", func(b *testing.B) {
		bench(b, "abc", "d")
	})
	b.Run("Cyrillic", func(b *testing.B) {
		bench(b, "А̀ВЄ", "Њ")
	})
	b.Run("Han", func(b *testing.B) {
		bench(b, "遠方", "來")
	})
}

// Pathological worst-case. Consistency here is a good thing.
func BenchmarkIndexEarlyMatchSmallNeedle(b *testing.B) {
	bench := func(b *testing.B, s1, s2 string) {
		for i := 2; i <= 32; i += 2 {
			s := strings.Repeat(s1, i) + s2
			substr := s1 + s2
			b.Run(strconv.Itoa(i), func(b *testing.B) {
				benchmarkIndex(b, s, substr)
			})
		}
	}
	b.Run("Latin", func(b *testing.B) {
		bench(b, "AB", "C")
	})
	b.Run("Cyrillic", func(b *testing.B) {
		bench(b, "А̀В", "Њ")
	})
	b.Run("Han", func(b *testing.B) {
		bench(b, "遠方", "來")
	})
}

// Thanks to variable length encoding it's possible the needle
// to be larger than the haystack.
func BenchmarkIndexNeedleExceedsHaystack(b *testing.B) {
	s := strings.Repeat("А̀В", 32*1024)
	substr := s + s[:len(s)/2] + "z"
	i1 := strings.Index(s, substr)
	i2 := Index(s, substr)
	if i1 != i2 {
		b.Fatalf("wrong index: expected: %d, got: %d", i1, i2)
	}
	// Can't compare perf to the stdlib because we have to scan
	// the whole string and not just bail at the length mismatch.
	b.SetBytes(int64(len(s)))
	for i := 0; i < b.N; i++ {
		Index(s, substr)
	}
}

// Pathological worst-case. Consistency here is a good thing.
func BenchmarkLastIndexLateMatchSmallNeedle(b *testing.B) {
	bench := func(b *testing.B, s1, s2 string) {
		s := s2 + strings.Repeat(s1, 1_000/len(s1))
		rs := []rune(s)
		for _, i := range []int{2, 16, 32} {
			b.Run(strconv.Itoa(i), func(b *testing.B) {
				b.SetBytes(int64(len(s)))
				substr := string(rs[:i])
				for i := 0; i < b.N; i++ {
					if j := LastIndex(s, substr); j != 0 {
						b.Fatalf("LastIndex(%q, %q) = %d; want: %d", s, substr, j, 0)
					}
				}
			})
		}
	}
	b.Run("Cyrillic", func(b *testing.B) {
		bench(b, "А̀ВЄ", "Њ")
	})
	b.Run("Han", func(b *testing.B) {
		bench(b, "遠方", "來")
	})
}

func makeBenchInputHard() string {
	tokens := [...]string{
		"<a>", "<p>", "<b>", "<strong>",
		"</a>", "</p>", "</b>", "</strong>",
		"hello", "world",
	}
	x := make([]byte, 0, 1<<20)
	for {
		i := rand.Intn(len(tokens))
		if len(x)+len(tokens[i]) >= 1<<20 {
			break
		}
		x = append(x, tokens[i]...)
	}
	return string(x)
}

var benchInputHard = makeBenchInputHard()

func benchmarkIndexHard(b *testing.B, sep string) {
	benchmarkIndex(b, benchInputHard, sep)
}

func benchmarkLastIndexHard(b *testing.B, sep string) {
	i := LastIndex(benchInputHard, sep)
	if i < 0 {
		b.SetBytes(int64(len(benchInputHard)))
	} else {
		b.SetBytes(int64(i + len(sep)))
	}
	for i := 0; i < b.N; i++ {
		LastIndex(benchInputHard, sep)
	}
}

func BenchmarkIndexHard1(b *testing.B) { benchmarkIndexHard(b, "<>") }
func BenchmarkIndexHard2(b *testing.B) { benchmarkIndexHard(b, "</pre>") }
func BenchmarkIndexHard3(b *testing.B) { benchmarkIndexHard(b, "<b>hello world</b>") }
func BenchmarkIndexHard4(b *testing.B) {
	benchmarkIndexHard(b, "<pre><b>hello</b><strong>world</strong></pre>")
}

func BenchmarkLastIndexHard1(b *testing.B) { benchmarkLastIndexHard(b, "<>") }
func BenchmarkLastIndexHard2(b *testing.B) { benchmarkLastIndexHard(b, "</pre>") }
func BenchmarkLastIndexHard3(b *testing.B) { benchmarkLastIndexHard(b, "<b>hello world</b>") }

var (
	benchInputTorture  = strings.Repeat("ABC", 1<<10) + "123" + strings.Repeat("ABC", 1<<10)
	benchNeedleTorture = strings.Repeat("ABC", 1<<10+1)

	benchInputTortureUnicode  = strings.Repeat("ΑΒΔ", 1<<10) + "123" + strings.Repeat("ΑΒΔ", 1<<10)
	benchNeedleTortureUnicode = strings.Repeat("ΑΒΔ", 1<<10+1)
)

func BenchmarkIndexTorture(b *testing.B) {
	benchmarkIndex(b, benchInputTorture, benchNeedleTorture)
}

func BenchmarkIndexTortureUnicode(b *testing.B) {
	benchmarkIndex(b, benchInputTortureUnicode, benchNeedleTortureUnicode)
}

func BenchmarkIndexPeriodic(b *testing.B) {
	key := "aa"
	for _, skip := range [...]int{2, 4, 8, 16, 32, 64} {
		b.Run(fmt.Sprintf("IndexPeriodic%d", skip), func(b *testing.B) {
			s := strings.Repeat("a"+strings.Repeat(" ", skip-1), 1<<16/skip)
			benchmarkIndex(b, s, key)
		})
	}
}

func BenchmarkIndexPeriodicUnicode(b *testing.B) {
	key := "αa"
	for _, skip := range [...]int{2, 4, 8, 16, 32, 64} {
		b.Run(fmt.Sprintf("IndexPeriodic%d", skip), func(b *testing.B) {
			s := strings.Repeat("α"+strings.Repeat(" ", skip-1), 1<<16/skip)
			benchmarkIndex(b, s, key)
		})
	}
}

func BenchmarkIndexNonASCII(b *testing.B) {
	for _, size := range indexSizes {
		b.Run(valName(size), func(b *testing.B) {
			s := strings.Repeat("a", size-1) + string(rune(utf8.RuneSelf))
			if i := IndexNonASCII(s); i < 0 {
				b.Fatalf("IndexNonASCII(%q) = -1", s)
				return
			}
			b.SetBytes(int64(len(s)))
			for i := 0; i < b.N; i++ {
				IndexNonASCII(s)
			}
		})
	}
}

func BenchmarkHasPrefixASCII(b *testing.B) {
	s := strings.Repeat("a", 64)
	if !HasPrefix(s, s) {
		b.Fatalf("HasPrefix(%[1]q, %[1]q) = false; want: true", s)
	}
	b.SetBytes(int64(len(s)))
	for i := 0; i < b.N; i++ {
		HasPrefix(s, s)
	}
}

func BenchmarkHasPrefix(b *testing.B) {
	if !HasPrefix(benchmarkString, benchmarkString) {
		b.Fatalf("HasPrefix(%[1]q, %[1]q) = false; want: true", benchmarkString)
	}
	b.SetBytes(int64(len(benchmarkString)))
	for i := 0; i < b.N; i++ {
		HasPrefix(benchmarkString, benchmarkString)
	}
}

func BenchmarkHasPrefixUnicode(b *testing.B) {
	const prefix = "Владимир Маяковский родился"
	b.SetBytes(int64(len(prefix)))
	for i := 0; i < b.N; i++ {
		HasPrefix(prefix, "Владимир МАЯКОВСКИЙ родился")
	}
}

func BenchmarkHasPrefixTests(b *testing.B) {
	for i := 0; i < b.N; i++ {
		for _, test := range prefixTests {
			HasPrefix(test.s, test.prefix)
		}
	}
}

func BenchmarkHasPrefixHard(b *testing.B) {
	if !HasPrefix(benchInputHard, benchInputHard) {
		b.Fatalf("HasPrefix(%[1]q, %[1]q) = false; want: true", benchInputHard)
	}
	b.SetBytes(int64(len(benchInputHard)))
	for i := 0; i < b.N; i++ {
		HasPrefix(benchInputHard, benchInputHard)
	}
}

func BenchmarkHasPrefixRussian(b *testing.B) {
	if !HasPrefix(russianLower, russianUpper) {
		b.Fatalf("HasPrefix(%[1]q, %[1]q) = false; want: true", russianText)
	}
	b.SetBytes(int64(len(russianLower)))
	for i := 0; i < b.N; i++ {
		HasPrefix(russianLower, russianUpper)
	}
}

func BenchmarkHasPrefixLonger(b *testing.B) {
	prefix := strings.Repeat("\u212a", 32)
	s := strings.Repeat("k", 32)
	if !HasPrefix(s, prefix) {
		b.Fatalf("HasPrefix(%q, %q) = false; want: true", s, prefix)
	}

	b.Run("Equal", func(b *testing.B) {
		b.SetBytes(int64(len(prefix)))
		for i := 0; i < b.N; i++ {
			HasPrefix(s, prefix)
		}
	})

	b.Run("ShortCircuitSize", func(b *testing.B) {
		prefix := prefix + "\u212a"
		b.SetBytes(int64(len(prefix)))
		for i := 0; i < b.N; i++ {
			HasPrefix(s, prefix)
		}
	})

	// Benchmark the overhead of checking for Kelvin
	b.Run("KelvinCheck", func(b *testing.B) {
		s := s + "\u212a"
		b.SetBytes(int64(len(s)))
		for i := 0; i < b.N; i++ {
			containsKelvin(s)
		}
	})
}

// TODO: need to compare against the stdlib
func BenchmarkHasSuffix(b *testing.B) {
	if !HasSuffix(benchmarkString, benchmarkString) {
		b.Fatalf("HasSuffix(%[1]q, %[1]q) = false; want: true", benchmarkString)
	}
	for i := 0; i < b.N; i++ {
		HasSuffix(benchmarkString, benchmarkString)
	}
}

// TODO: match the logic of HasPrefix
// TODO: need to compare against the stdlib
func BenchmarkHasSuffixRussian(b *testing.B) {
	if !HasSuffix(russianLower, russianUpper) {
		b.Fatalf("HasSuffix(%[1]q, %[1]q) = false; want: true", russianText)
	}
	b.SetBytes(int64(len(russianLower)))
	for i := 0; i < b.N; i++ {
		HasSuffix(russianLower, russianUpper)
	}
}

func benchmarkIndexAny(b *testing.B, s, chars string) {
	i1 := strings.IndexAny(s, chars)
	i2 := IndexAny(s, chars)
	if i1 != i2 {
		b.Fatalf("strings.IndexAny != IndexAny: %d != %d", i1, i2)
	}
	min := len(s)
	for i, r := range chars {
		o := strings.IndexRune(s, r)
		if 0 <= o && o < min {
			min = i + utf8.RuneLen(r) // Include the length of the matched rune
		}
	}
	bytes := int64(min)
	if *benchStdLib {
		b.SetBytes(bytes)
		for i := 0; i < b.N; i++ {
			strings.IndexAny(s, chars)
		}
	} else {
		b.SetBytes(bytes)
		for i := 0; i < b.N; i++ {
			IndexAny(s, chars)
		}
	}
}

func BenchmarkIndexAnyASCII(b *testing.B) {
	x := strings.Repeat("#", 2048) // Never matches set
	cs := "0123456789abcdefghijklmnopqrstuvwxyz0123456789abcdefghijklmnopqrstuvwxyz"
	for k := 1; k <= 2048; k <<= 4 {
		for j := 1; j <= 64; j <<= 1 {
			b.Run(fmt.Sprintf("%d:%d", k, j), func(b *testing.B) {
				benchmarkIndexAny(b, x[:k], cs[:j])
			})
		}
	}
}

func BenchmarkIndexAnyUTF8(b *testing.B) {
	x := strings.Repeat("#", 2048) // Never matches set
	// TODO: use a more diverse string (diff languages)
	cs := "你好世界, hello world. 你好世界, hello world. 你好世界, hello world."
	for k := 1; k <= 2048; k <<= 4 {
		for j := 1; j <= 64; j <<= 1 {
			b.Run(fmt.Sprintf("%d:%d", k, j), func(b *testing.B) {
				var chars string
				n := j
				for i, r := range cs {
					n--
					if n <= 0 {
						chars = cs[:i+utf8.RuneLen(r)]
						break
					}
				}
				benchmarkIndexAny(b, x[:k], chars)
			})
		}
	}
}

func benchmarkLastIndexAny(b *testing.B, s, chars string) {
	i1 := strings.LastIndexAny(s, chars)
	i2 := LastIndexAny(s, chars)
	if i1 != i2 {
		b.Fatalf("strings.LastIndexAny != LastIndexAny: %d != %d", i1, i2)
	}
	// TODO: make sure the logic here is correct
	i := strings.LastIndexAny(s, chars)
	if i < 0 {
		i = 0
	}
	bytes := int64(len(s) - i)
	if *benchStdLib {
		b.SetBytes(bytes)
		for i := 0; i < b.N; i++ {
			strings.LastIndexAny(s, chars)
		}
	} else {
		b.SetBytes(bytes)
		for i := 0; i < b.N; i++ {
			LastIndexAny(s, chars)
		}
	}
}

func BenchmarkLastIndexAnyASCII(b *testing.B) {
	x := strings.Repeat("#", 2048) // Never matches set
	cs := "0123456789abcdefghijklmnopqrstuvwxyz0123456789abcdefghijklmnopqrstuvwxyz"
	for k := 1; k <= 2048; k <<= 4 {
		for j := 1; j <= 64; j <<= 1 {
			b.Run(fmt.Sprintf("%d:%d", k, j), func(b *testing.B) {
				for i := 0; i < b.N; i++ {
					benchmarkLastIndexAny(b, x[:k], cs[:j])
				}
			})
		}
	}
}

func BenchmarkLastIndexAnyUTF8(b *testing.B) {
	x := strings.Repeat("#", 2048) // Never matches set
	cs := "你好世界, hello world. 你好世界, hello world. 你好世界, hello world."
	for k := 1; k <= 2048; k <<= 4 {
		for j := 1; j <= 64; j <<= 1 {
			b.Run(fmt.Sprintf("%d:%d", k, j), func(b *testing.B) {
				for i := 0; i < b.N; i++ {
					benchmarkLastIndexAny(b, x[:k], cs[:j])
				}
			})
		}
	}
}

// Micro-benchmarks for caseFold

var caseFoldBenchmarkRunes = [16]rune{
	0xA7C9,
	0xA696,
	0x03A7,
	0x021E,
	0x03A3,
	0x01B5,
	0x01A6,
	0xABBC,
	0xA72C,
	0x1F8E,
	0x0056,
	0x016E,
	0x1E86,
	0x1C92,
	0x0555,
	0x0544,
}

var caseFoldBenchmarkAll []rune

func loadCaseFoldBenchmarkAll() {
	if caseFoldBenchmarkAll != nil {
		return
	}
	n := 0
	for _, p := range _CaseFolds {
		if p.From != 0 {
			n++
		}
	}
	a := make([]rune, n)
	i := 0
	for _, p := range _CaseFolds {
		if p.From != 0 {
			a[i] = rune(p.From)
			i++
		}
	}
	// Make sure the slice is consistently sorted before
	// randomizing order. This is relevant because the
	// order of slice elements may change.
	sort.Slice(a, func(i, j int) bool {
		return a[i] < a[j]
	})
	rr := rand.New(rand.NewSource(12345))
	rr.Shuffle(len(a), func(i, j int) {
		a[i], a[j] = a[j], a[i]
	})
	caseFoldBenchmarkAll = a
}

func BenchmarkCaseFold(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = caseFold(caseFoldBenchmarkRunes[i%len(caseFoldBenchmarkRunes)])
	}
}

func BenchmarkCaseFoldAll(b *testing.B) {
	loadCaseFoldBenchmarkAll()
	for i := 0; i < b.N; i++ {
		for j := i; j < len(caseFoldBenchmarkAll) && j < b.N; j++ {
			_ = caseFold(caseFoldBenchmarkAll[j])
		}
	}
}

// Micro-benchmarks for toUpperLower

var toUpperLowerBenchmarkRunes = [16]rune{
	0xA68A,
	0x0204,
	0x04EC,
	0x00D0,
	0x0053,
	0xA698,
	0x1F1A,
	0x038E,
	0x1F1B,
	0x2126,
	0x16E47,
	0x01D1,
	0x13CC,
	0x01BC,
	0x048E,
	0x0386,
}

var toUpperLowerBenchmarkAll []rune

// WARN: this is not deterministic
func loadToUpperLowerBenchmarkAll() {
	if toUpperLowerBenchmarkAll != nil {
		return
	}
	a := make([]rune, len(_UpperLower))
	for i, p := range _UpperLower {
		a[i] = rune(p[0])
	}
	// Make sure the slice is consistently sorted before
	// randomizing order. This is relevant because the
	// order of slice elements may change.
	sort.Slice(a, func(i, j int) bool {
		return a[i] < a[j]
	})
	rr := rand.New(rand.NewSource(12345))
	rr.Shuffle(len(a), func(i, j int) {
		a[i], a[j] = a[j], a[i]
	})
	toUpperLowerBenchmarkAll = a[:256]
}

func BenchmarkToUpperLower(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _, _ = toUpperLower(toUpperLowerBenchmarkRunes[i%len(toUpperLowerBenchmarkRunes)])
	}
}

func BenchmarkToUpperLowerAll(b *testing.B) {
	loadToUpperLowerBenchmarkAll()
	for i := 0; i < b.N; i++ {
		for _, r := range toUpperLowerBenchmarkAll {
			_, _, _ = toUpperLower(r)
		}
	}
}

func BenchmarkNonLetterASCII(b *testing.B) {
	base := "!\"#$%&'()*+,-./0123456789:;<=>?@[\\]^_`{|}~"
	base += base + base + base
	for _, size := range []int{4, 8, 16, 24, 32, 64, 128} {
		b.Run(fmt.Sprint(size), func(b *testing.B) {
			s := base[:size]
			b.SetBytes(int64(len(s)))
			for i := 0; i < b.N; i++ {
				nonLetterASCII(s)
			}
		})
	}
}
