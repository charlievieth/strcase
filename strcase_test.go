package strcase

import (
	"flag"
	"fmt"
	"math/rand"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"unicode"
	"unicode/utf8"

	"github.com/charlievieth/strcase/internal/cstr"
	"golang.org/x/text/unicode/rangetable"
)

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
}

func mustHaveCstr(t testing.TB) {
	if !cstr.Enabled {
		t.Skipf("cstr: package not available on platform: %s/%s",
			runtime.GOOS, runtime.GOARCH)
		panic("unreachable")
	}
}

func TestCompare(t *testing.T) {
	for _, test := range compareTests {
		got := Compare(test.s, test.t)
		if got != test.out {
			t.Errorf("Compare(%q, %q) = %d; want: %d", test.s, test.t, got, test.out)
		}
	}
}

func hasUnicode(s string) bool {
	for _, r := range s {
		if r >= utf8.RuneSelf {
			return true
		}
	}
	return false
}

func TestCompareReference(t *testing.T) {
	mustHaveCstr(t)

	t.Run("Strcasecmp", func(t *testing.T) {
		for _, test := range compareTests {
			if hasUnicode(test.s) || hasUnicode(test.t) {
				continue
			}
			got := cstr.Strcasecmp(test.s, test.t)
			if got != test.out {
				t.Errorf("Strcasecmp(%q, %q) = %d; want: %d", test.s, test.t, got, test.out)
			}
		}
	})

	t.Run("Wcscasecmp", func(t *testing.T) {
		for _, test := range compareTests {
			got := cstr.Wcscasecmp(test.s, test.t)
			if got != test.out {
				t.Errorf("Wcscasecmp(%q, %q) = %d; want: %d", test.s, test.t, got, test.out)
			}
		}
	})
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

	// Tests discovered with fuzzing
	{"4=K ", "=\u212a", 1},
	{"I", "\u0131", -1},
	{"aßẛ", "ß", 1},

	{"\u0250\u0250\u0250\u0250\u0250 a", "\u2C6F\u2C6F\u2C6F\u2C6F\u2C6F A", 0}, // grows one byte per char
	{"a\u0250\u0250\u0250\u0250\u0250", "A\u2C6F\u2C6F\u2C6F\u2C6F\u2C6F", 0},   //
	{"\u2C6D\u2C6D\u2C6D\u2C6D\u2C6D a", "\u0251\u0251\u0251\u0251\u0251 A", 0}, // shrinks one byte per char
	{"a\u2C6D\u2C6D\u2C6D\u2C6D\u2C6D", "A\u0251\u0251\u0251\u0251\u0251", 0},   // shrinks one byte per char
	{"abc\u2C6D\u2C6D\u2C6D\u2C6D\u2C6D", "\u0251\u0251\u0251\u0251\u0251", 3},
	{"ΑΒΔ\u2C6D\u2C6D\u2C6D\u2C6D\u2C6D", "\u0251\u0251\u0251\u0251\u0251", len("ΑΒΔ")},
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
				_, foldable = _FoldMap[r]
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

// Test that Index and C's Strcasestr return the same result.
func TestIndexStrcasestr(t *testing.T) {
	mustHaveCstr(t)
	filter := func(it IndexTest) bool {
		return !hasUnicode(it.s) && !hasUnicode(it.sep)
	}
	tests := filterIndexTests(filter, indexTests)
	runIndexTests(t, cstr.Strcasestr, "Strcasestr", tests, true)
}

// WARN: do we need this??
func TestIndexRegex(t *testing.T) {
	index := func(s, sep string) int {
		i := regexp.MustCompile(`(?i)` + regexp.QuoteMeta(sep)).FindStringIndex(s)
		if len(i) == 2 {
			return i[0]
		}
		return -1
	}
	// Ignore 'İ' since it does not match under Unicode folding
	filter := func(t IndexTest) bool {
		return !strings.Contains(t.s, "İ") && !strings.Contains(t.sep, "İ")
	}
	tests := filterIndexTests(filter, indexTests, unicodeIndexTests)
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

func TestIndexRune(t *testing.T) {
	tests := []struct {
		in   string
		rune rune
		want int
	}{
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
	}
	for _, tt := range tests {
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

func TestIndexByte(t *testing.T) {
	tests := []struct {
		in   string
		char byte
		want int
	}{
		// Case-folding with ASCII
		{"K", 'k', 0},
		{"K", 'K', 0},
		{"ſ", 's', 0},
		{"ſ", 'S', 0},
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

	tests := filterIndexTests(nil, indexTests, unicodeIndexTests)
	tw := &testWrapper{T: t}
	for _, test := range tests {
		want := index(test.s)
		got := IndexNonASCII(test.s)
		if got != want {
			tw.Errorf("IndexNonASCII(%q) = %d; want: %d", test.s, got, want)
		}
	}

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
			tw.Errorf("IndexNonASCII(long[%d:]) = %d; want: %d", i, got, want)
		}
	}
}

type PrefixTest struct {
	s, prefix      string
	out, exhausted bool
}

var prefixTests = []PrefixTest{
	{"", "", true, true},
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

// WARN: DELETE ME
func TestHasSuffixUnicocde(t *testing.T) {
	type SuffixTest struct {
		s, suffix string
		match     bool
		i         int
	}
	tests := []SuffixTest{
		{"f", "f", true, 0},
		{"foo", "foo", true, 0},
		{"foo", "o", true, 2},
		{"foo", "", true, 3},
		{"foobar", "bar", true, len("foo")},
		{"foobar", "baz", false, 0},
		{"ΑΒΔ", "ΑΒΔ", true, 0},
		{"αβδΑΒΔ", "ΑΒΔ", true, len("αβδ")},
		{"abc☻K@", "☻k@", true, 3},

		{"\u0250\u0250\u0250\u0250\u0250 a", "\u2C6F\u2C6F\u2C6F\u2C6F\u2C6F A", true, 0}, // grows one byte per char
		{"a\u0250\u0250\u0250\u0250\u0250", "A\u2C6F\u2C6F\u2C6F\u2C6F\u2C6F", true, 0},   //
		{"\u2C6D\u2C6D\u2C6D\u2C6D\u2C6D a", "\u0251\u0251\u0251\u0251\u0251 A", true, 0}, // shrinks one byte per char
		{"a\u2C6D\u2C6D\u2C6D\u2C6D\u2C6D", "A\u0251\u0251\u0251\u0251\u0251", true, 0},   // shrinks one byte per char
		{"abc\u2C6D\u2C6D\u2C6D\u2C6D\u2C6D", "\u0251\u0251\u0251\u0251\u0251", true, 3},
		{"ΑΒΔ\u2C6D\u2C6D\u2C6D\u2C6D\u2C6D", "\u0251\u0251\u0251\u0251\u0251", true, len("ΑΒΔ")},
	}
	for _, test := range tests {
		match, _, i := hasSuffixUnicode(test.s, test.suffix)
		if match != test.match || i != test.i {
			t.Errorf("HasSuffix(%q, %q) = %t, _, %d; want: %t, _, %d", test.s, test.suffix, match, i, test.match, test.i)
		}
	}
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

func BenchmarkBruteForceIndexASCII(b *testing.B) {
	if out := bruteForceIndexASCII("x012345678x0a2b4c6d8e", "0A2B4C6D8E"); out != 11 {
		b.Fatalf("bruteForceIndexASCII(%q, %q) = %d; want %d", "x012345678x0123456789", "0123456789", out, 11)
	}
	for i := 0; i < b.N; i++ {
		bruteForceIndexASCII("x012345678x0123456789", "0123456789")
	}
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

	const s1 = "abcdefghijKz"
	const s2 = "abcDefGhijKz"

	b.Run("ASCII", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			Compare(s1, s2)
		}
	})

	b.Run("ASCII_Long", func(b *testing.B) {
		const s = s1 + s1 + s1 + s1 + s1
		const t = s2 + s2 + s2 + s2 + s2
		for i := 0; i < b.N; i++ {
			Compare(s, t)
		}
	})

	b.Run("UnicodePrefix", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			Compare("αβδ"+s1, "ΑΒΔ"+s2)
		}
	})

	b.Run("UnicodeSuffix", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			Compare(s1+"αβδ", s2+"ΑΒΔ")
		}
	})
}

const benchmarkString = "some_text=some☺value"

func BenchmarkIndexRabinKarpUnicode(b *testing.B) {
	if indexRabinKarpUnicode(benchmarkString, "☺value") == -1 {
		b.Fatal("invalid")
	}
	for i := 0; i < b.N; i++ {
		indexRabinKarpUnicode(benchmarkString, "☺value")
	}
}

func BenchmarkIndexRune(b *testing.B) {
	if got := IndexRune(benchmarkString, '☺'); got != 14 {
		b.Fatalf("wrong index: expected 14, got=%d", got)
	}
	for i := 0; i < b.N; i++ {
		IndexRune(benchmarkString, '☺')
	}
}

var benchmarkLongString = strings.Repeat(" ", 100) + benchmarkString

func BenchmarkIndexRuneLongString(b *testing.B) {
	if got := IndexRune(benchmarkLongString, '☺'); got != 114 {
		b.Fatalf("wrong index: expected 114, got=%d", got)
	}
	for i := 0; i < b.N; i++ {
		IndexRune(benchmarkLongString, '☺')
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

func BenchmarkIndexByte(b *testing.B) {
	const ch = 'V'
	if got := IndexByte(benchmarkString, ch); got != 17 {
		b.Fatalf("wrong index: expected 17, got=%d", got)
	}
	for i := 0; i < b.N; i++ {
		IndexByte(benchmarkString, ch)
	}
}

const _s = "|0123456789abcdefghijklmnopqrstu_wxyzABCDEFGHIJKLMNOPQRSTU_WXYZ|" // 64

const benchmarkStringLong = "" +
	_s + _s + _s + _s + _s + _s + _s + _s + // 512
	"V" +
	_s + _s + _s + _s + _s + _s + _s + _s + // 512
	"v"

func BenchmarkIndexByteLong(b *testing.B) {
	const ch = 'V'
	if got := IndexByte(benchmarkStringLong, ch); got != 512 {
		b.Fatalf("wrong index: expected 17, got=%d", got)
	}
	for i := 0; i < b.N; i++ {
		IndexByte(benchmarkStringLong, ch)
	}
}

func BenchmarkLastIndexByte(b *testing.B) {
	// TODO: we don't really need this benchmark
	if testing.Short() {
		b.Skip("short test")
	}
	const ch = 'S'
	if got := LastIndexByte(benchmarkString, ch); got != 10 {
		b.Fatalf("wrong index: expected 10, got=%d", got)
	}
	for i := 0; i < b.N; i++ {
		LastIndexByte(benchmarkString, ch)
	}
}

// WARN
var benchStdLib = flag.Bool("stdlib", false, "Use strings.Index in benchmarks (for comparison)")

func benchmarkIndex(b *testing.B, s, substr string) {
	if *benchStdLib {
		s := strings.ToLower(s)
		substr := strings.ToLower(substr)
		for i := 0; i < b.N; i++ {
			strings.Index(s, substr)
		}
	} else {
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
	if got := Index(benchmarkString, "v"); got != 17 {
		b.Fatalf("wrong index: expected 17, got=%d", got)
	}
	for i := 0; i < b.N; i++ {
		LastIndex(benchmarkString, "v")
	}
}

// TODO: rename
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

func BenchmarkIndexRussian(b *testing.B) {
	// https://ru.wikipedia.org/wiki/%D0%9C%D0%B0%D1%8F%D0%BA%D0%BE%D0%B2%D1%81%D0%BA%D0%B8%D0%B9,_%D0%92%D0%BB%D0%B0%D0%B4%D0%B8%D0%BC%D0%B8%D1%80_%D0%92%D0%BB%D0%B0%D0%B4%D0%B8%D0%BC%D0%B8%D1%80%D0%BE%D0%B2%D0%B8%D1%87

	const text = `Владимир Маяковский родился в селе Багдади[10] Кутаисской
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

	// if bruteForceIndexUnicode(text, "МЛАДЕНЧЕСТВЕ") == -1 {
	// 	b.Fatal("NOPE")
	// }
	for i := 0; i < b.N; i++ {
		// s := text
		// substr := "МЛАДЕНЧЕСТВЕ"
		// strings.Index(strings.ToLower(s), strings.ToLower(substr))
		if Index(text, "МЛАДЕНЧЕСТВЕ") == -1 {
			b.Fatal("FAIL")
		}
		// Index(text, "Δ"+"младенчестве")
		// bruteForceIndexUnicode(text, "Δ"+"младенчестве")
		// strings.Index(text, "младенчестве")
	}

	// младенчестве
	// МЛАДЕНЧЕСТВЕ
	// владимир маяковский
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
			// s := strings.Repeat("Α"+strings.Repeat(" ", skip-1), 1<<16/skip)
			benchmarkIndex(b, s, key)
		})
	}
}

func BenchmarkIndexNonASCII(b *testing.B) {
	for _, size := range [...]int{4, 8, 16, 32, 64, 128, 256} {
		b.Run(fmt.Sprintf("%d", size), func(b *testing.B) {
			s := strings.Repeat("a", size-1) + string(rune(utf8.RuneSelf))
			if i := IndexNonASCII(s); i < 0 {
				b.Fatalf("IndexNonASCII(%q) = -1", s)
				return
			}
			for i := 0; i < b.N; i++ {
				IndexNonASCII(s)
			}
		})
	}
}

func BenchmarkHasPrefix(b *testing.B) {
	if !HasPrefix(benchmarkString, benchmarkString) {
		b.Fatalf("HasPrefix(%[1]q, %[1]q) = false; want: true", benchmarkString)
	}
	for i := 0; i < b.N; i++ {
		HasPrefix(benchmarkString, benchmarkString)
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
	for i := 0; i < b.N; i++ {
		HasPrefix(benchInputHard, benchInputHard)
	}
}

func BenchmarkHasPrefixLonger(b *testing.B) {
	prefix := strings.Repeat("\u212a", 32)
	s := strings.Repeat("k", 32)
	if !HasPrefix(s, prefix) {
		b.Fatalf("HasPrefix(%q, %q) = false; want: true", s, prefix)
	}

	b.Run("Equal", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			HasPrefix(s, prefix)
		}
	})

	b.Run("ShortCircuitSize", func(b *testing.B) {
		prefix := prefix + "\u212a"
		for i := 0; i < b.N; i++ {
			HasPrefix(s, prefix)
		}
	})

	// Benchmark the overhead of checking for Kelvin
	b.Run("KelvinCheck", func(b *testing.B) {
		s := s + "\u212a"
		for i := 0; i < b.N; i++ {
			strings.Contains(s, string('\u212A'))
		}
	})
}

func BenchmarkHasSuffix(b *testing.B) {
	if !HasSuffix(benchmarkString, benchmarkString) {
		b.Fatalf("HasSuffix(%[1]q, %[1]q) = false; want: true", benchmarkString)
	}
	for i := 0; i < b.N; i++ {
		HasSuffix(benchmarkString, benchmarkString)
	}
}
