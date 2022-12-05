package strcase

import (
	"flag"
	"fmt"
	"io"
	"math/rand"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"unicode"
	"unicode/utf8"

	"github.com/charlievieth/reonce/recache"
	"github.com/charlievieth/strcase/internal/cstr"
	"golang.org/x/text/unicode/rangetable"
)

type CompareTest struct {
	s, t string
	out  int
}

var CompareTests = []CompareTest{
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
}

func TestCompareReference(t *testing.T) {
	for _, test := range CompareTests {
		if hasUnicode(test.s) || hasUnicode(test.t) {
			continue
		}
		got := cstr.Strcasecmp(test.s, test.t)
		if got != test.out {
			t.Errorf("Strcasecmp(%q, %q) = %d; want: %d", test.s, test.t, got, test.out)
		}
	}
	for _, test := range CompareTests {
		got := cstr.Wcscasecmp(test.s, test.t)
		if got != test.out {
			t.Errorf("Wcscasecmp(%q, %q) = %d; want: %d", test.s, test.t, got, test.out)
		}
	}
}

func TestCompare(t *testing.T) {
	for _, test := range CompareTests {
		got := Compare(test.s, test.t)
		if got != test.out {
			t.Errorf("Compare(%q, %q) = %d; want: %d", test.s, test.t, got, test.out)
		}
	}
}

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
	// {"xx0123456789012345678901234567890123456789012345678901234567890120123456789012345678901234567890123456xxx☻", "0123456789012345678901234567890123456xxx☻", 65},

	{"abc☻", "abc☻", 0},
	{"abc☻", "ABC☻", 0},
	{"123abc☻", "ABC☻", 3},
}

// These tests fail with strcasestr.
var unicodeIndexTests = []IndexTest{
	// Map Kelvin 'K' (U+212A) to lowercase latin 'k'.
	{"abcK@", "k@", 3},

	// Map "Latin capital letter I with dot above" 'İ' to lowercase latin 'i'.
	{"abcİ@", "i@", 3},

	// WARN: dev only
	//
	// Test with a unicode prefix in the substr to make sure the unicode
	// implementation is correct.
	{"abc☻K@", "☻k@", 3},
	{"abc☻İ@", "☻i@", 3},
}

// Execute f on each test case.  funcName should be the name of f; it's used
// in failure reports.
func runIndexTests(t *testing.T, f func(s, sep string) int, funcName string, testCases []IndexTest, noError bool) {
	for _, test := range testCases {
		// t.Logf("%s(%q,%q)", funcName, test.s, test.sep) // WARN WARN WARN
		actual := f(test.s, test.sep)
		if actual != test.out {
			errorf := t.Errorf
			if noError {
				errorf = t.Logf
			}
			if hasUnicode(test.s) || hasUnicode(test.sep) {
				errorf("%s(%q,%q) = %v; want %v\n"+
					"Args:\n"+
					"  s:   %s\n"+
					"  sep: %s\n"+
					"Lower:\n"+
					"  s:   %s\n"+
					"  sep: %s\n",
					funcName, test.s, test.sep, actual, test.out,
					strconv.QuoteToASCII(test.s),
					strconv.QuoteToASCII(test.sep),
					strconv.QuoteToASCII(strings.ToLower(test.s)),
					strconv.QuoteToASCII(strings.ToLower(test.sep)),
				)
			} else {
				errorf("%s(%q,%q) = %v; want %v", funcName, test.s, test.sep, actual, test.out)
			}
		}
	}
}

func TestIndexReference(t *testing.T) {
	runIndexTests(t, cstr.Strcasestr, "Strcasestr", indexTests, false)
}

// WARN: do we need this??
// For compat
func TestIndexRegex(t *testing.T) {
	fn := func(s, sep string) int {
		// WARN: dont need the cache here
		re := recache.MustCompile(`(?i)` + regexp.QuoteMeta(sep))
		x := re.FindIndex([]byte(s))
		if len(x) == 2 {
			return x[0]
		}
		return -1
	}
	runIndexTests(t, fn, "Regexp", indexTests, false)
}

func TestIndex(t *testing.T) {
	tests := append(indexTests, unicodeIndexTests...)
	runIndexTests(t, Index, "Index", tests, false)
}

// WARN: remove once work on Rabin-Karp is done
func TestIndexRabinKarp(t *testing.T) {
	var tests []IndexTest
	for _, t := range indexTests {
		if len(t.sep) == 0 || len(t.sep) >= len(t.s) {
			continue
		}
		tests = append(tests, t)
	}
	t.Run("ASCII", func(t *testing.T) {
		runIndexTests(t, indexRabinKarp, "indexRabinKarp", tests, false)
	})
	t.Run("Unicode", func(t *testing.T) {
		runIndexTests(t, indexRabinKarpUnicode, "indexRabinKarpUnicode", tests, false)
	})
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
			// Make sure strcasestr returns the same result
			t.Run("Strcasestr", func(t *testing.T) {
				runIndexTests(t, cstr.Strcasestr, "Strcasestr", tests, true)
			})
			t.Run("RabinKarp", func(t *testing.T) {
				var rtests []IndexTest
				for _, t := range tests {
					if len(t.sep) > 0 && len(t.sep) <= len(t.s) {
						rtests = append(rtests, t)
					}
				}
				runIndexTests(t, indexRabinKarpUnicode, "indexRabinKarpUnicode", rtests, false)
			})
		})
	}
}

// TODO: use these
//
// Evil strings I discovered fuzzing
var variableWidthIndexTests = []IndexTest{
	{
		s:   "32ⱭⱭⱭⱭⱭ45678890ⱭⱭⱭⱭⱭ234567890ⱭⱭⱭⱭⱭ234567890ⱭⱭⱭⱭⱭ23456789ⱭⱭⱭⱭⱭⱭⱭⱭⱭⱭ",
		sep: "0ⱭⱭⱭⱭⱭ234567890ⱭⱭⱭⱭⱭ234567890ⱭⱭⱭⱭⱭ234567890",
		out: -1,
	},
	{
		s:   "\U000210b4T1\u2126\u2c6e\U00022c89\U000f9204\U000f2fb3\U0010baa5\U0002cd3bS\ud64d\u025c2E",
		sep: "\U000210b4t1\u03c9\u0271\U00022c89\U000f9204",
		out: 0,
	},
	{
		s:   "\u3579\x16\u01306\"\uebd5\u8cd6\u3a9a\U000fbcd8\U00024e9f\u1c81\x1f\u0240\u4f93\uf56a",
		sep: "\u3579\x16i6\"\uebd5\u8cd6\u3a9a\U000fbcd8\U00024e9f\u1c81\x1f\u2c7f",
		out: 0,
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
		s:   "\U0002024a\u0130\U000f3d86?\x11\ua3c5\U0010ab43\U000310a0\u4d8d\x03\U00016907L\U00024e04F\U000f5bb7",
		sep: "i\U000f3d86?\x11\ua3c5\U0010ab43\U000310a0\u4d8d\x03\U00016907l\U00024e04f",
		out: 4,
	},
}

// WARN: DEV ONLY
func TestIndexXXX(t *testing.T) {
	// S:    "𡂴T1ΩⱮ𢲉\U000f9204\U000f2fb3\U0010baa5𬴻S홍ɜ2E"
	// Sep:  "𡂴t1ωɱ𢲉\U000f9204"
	// Got:  -1
	// Want: 0

	// S:    "\U000210b4T1\u2126\u2c6e\U00022c89\U000f9204\U000f2fb3\U0010baa5\U0002cd3bS\ud64d\u025c2E"
	// Sep:  "\U000210b4t1\u03c9\u0271\U00022c89\U000f9204"
	tests := []IndexTest{
		// {
		// 	s:   "\u9ae7[\x17\U0002315a,T\u212a@\x03WH",
		// 	sep: "k@",
		// 	out: 11,
		// },
		{
			s:   "\u11cf\U0001f232\u6ed1\u1e9ez",
			sep: "滑ß",
			out: 7,
		},

		// "\u9ae7[\x17\U0002315a,T\u212a@\x03WH"
	}

	runIndexTests(t, indexReference, "IndexReference", tests, false)
	runIndexTests(t, cstr.Strcasestr, "Strcasestr", tests, true)
	runIndexTests(t, Index, "Index", tests, false)
}

func TestIndexCase(t *testing.T) {
	tests := append([]IndexTest(nil), indexTests...)
	for i, test := range tests {
		tests[i].sep = strings.ToUpper(test.sep)
	}
	runIndexTests(t, Index, "Index", tests, false)
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
	}
	for _, tt := range tests {
		if got := IndexRune(tt.in, tt.rune); got != tt.want {
			t.Errorf("IndexRune(%q, %d) = %v; want %v", tt.in, tt.rune, got, tt.want)
		}
	}

	haystack := "test世界"
	allocs := testing.AllocsPerRun(1000, func() {
		if i := IndexRune(haystack, 's'); i != 2 {
			t.Fatalf("'s' at %d; want 2", i)
		}
		if i := IndexRune(haystack, '世'); i != 4 {
			t.Fatalf("'世' at %d; want 4", i)
		}
	})
	if allocs != 0 && testing.CoverMode() == "" {
		t.Errorf("expected no allocations, got %f", allocs)
	}
}

func rangeMapToRangeTable(maps ...map[string]*unicode.RangeTable) *unicode.RangeTable {
	n := 0
	for _, m := range maps {
		n += len(m)
	}
	tabs := make([]*unicode.RangeTable, 0, n)
	for _, m := range maps {
		for _, t := range m {
			tabs = append(tabs, t)
		}
	}
	return rangetable.Merge(tabs...)
}

func printRangeTable(w io.Writer, name string, rt *unicode.RangeTable) {
	fmt.Fprintf(w, "var %s = &unicode.RangeTable{\n", name)
	if len(rt.R16) == 0 {
		fmt.Fprintln(w, "\tR16: []unicode.Range16{},")
	} else {
		fmt.Fprintln(w, "\tR16: []unicode.Range16{")
		for _, r := range rt.R16 {
			// %#04x
			// fmt.Fprintf(w, "\t\t{0x%x, 0x%x, %d},\n", r.Lo, r.Hi, r.Stride)
			fmt.Fprintf(w, "\t\t{%#04x, %#04x, %d},\n", r.Lo, r.Hi, r.Stride)
		}
		fmt.Fprintln(w, "\t},")
	}
	if len(rt.R32) == 0 {
		fmt.Fprintln(w, "\tR32: []unicode.Range32{},")
	} else {
		fmt.Fprintln(w, "\tR32: []unicode.Range32{")
		for _, r := range rt.R32 {
			fmt.Fprintf(w, "\t\t{%#06x, %#06x, %d},\n", r.Lo, r.Hi, r.Stride)
		}
		fmt.Fprintln(w, "\t},")
	}
	fmt.Fprintln(w, "}")
}

func BenchmarkMustLower(b *testing.B) {
	// Αβδ
	var runes = [4]rune{'Α', 'ß', 'å', '☺'}
	for i := 0; i < b.N; i++ {
		mustLower(runes[i%len(runes)])
		// mustLower('☺')
	}
}

func TestMustLower(t *testing.T) {
	folds := func(sr rune) []rune {
		r := unicode.SimpleFold(sr)
		runes := make([]rune, 1, 2)
		runes[0] = sr
		for r != sr {
			runes = append(runes, r)
			r = unicode.SimpleFold(r)
		}
		return runes
	}

	var runes []rune
	rangetable.Visit(unicodeCategories, func(r rune) {
		if ff := folds(r); len(ff) > 2 {
			runes = append(runes, ff...)
			return
		}
		lr := unicode.ToLower(r)
		if r >= utf8.RuneSelf && lr < utf8.RuneSelf {
			runes = append(runes, r, lr, unicode.ToUpper(lr))
		}
	})

	if len(runes) == 0 {
		t.Fatal("Failed to generate any runes!")
	}

	fails := 0
	// WARN: I think something is wrong here
	runes = append(runes, 'ẞ', 'ß') // WARN
	table := rangetable.New(runes...)
	{
		// printRangeTable(os.Stdout, "_MustLower", table)
		// return
	}

	lower := make(map[rune]bool)
	rangetable.Visit(table, func(r rune) { lower[r] = true })

	rangetable.Visit(unicodeCategories, func(r rune) {
		want := lower[r]
		got := mustLower(r)
		if want != got {
			t.Errorf("mustLower(%[1]c) = %[2]t; want: %[3]t\n"+
				"  lower: %[4]q (%[4]U) upper: %[5]q (%[5]U)",
				r, got, want, unicode.ToLower(r), unicode.ToUpper(r))
			fails++
		}
		if fails >= 50 {
			t.Fatal("Too many errors:", fails)
		}
	})
}

// type byRune []rune
//
// func (r byRune) Len() int           { return len(r) }
// func (r byRune) Less(i, j int) bool { return r[i] < r[j] }
// func (r byRune) Swap(i, j int)      { r[i], r[j] = r[j], r[i] }

func allFolds() *unicode.RangeTable {
	folds := func(sr rune) []rune {
		r := unicode.SimpleFold(sr)
		runes := make([]rune, 1, 2)
		runes[0] = sr
		for r != sr {
			runes = append(runes, r)
			r = unicode.SimpleFold(r)
		}
		return runes
	}

	runes := make([]rune, 0, 64)
	rangetable.Visit(unicodeCategories, func(r rune) {
		if ff := folds(r); len(ff) > 2 {
			runes = append(runes, ff...)
		}
		if r >= utf8.RuneSelf && unicode.ToLower(r) < utf8.RuneSelf {
			runes = append(runes, r)
		}
		// lr := unicode.ToLower(r)
		// if r >= utf8.RuneSelf && lr < utf8.RuneSelf {
		// 	runes = append(runes, r, lr, unicode.ToUpper(lr))
		// }
	})

	return rangetable.New(runes...)
}

func TestHasFolds(t *testing.T) {

	// folds := rangeMapToRangeTable(unicode.FoldScript)
	fails := 0
	rangetable.Visit(allFolds(), func(r rune) {
		// WARN
		// t.Logf("r: %c\n", r)
		if !hasFolds(r) {
			t.Errorf("hasFolds(%c) = %t", r, false)
			fails++
		}
		if fails >= 50 {
			t.Fatal("Too many errors:", fails)
		}
	})
}

func TestIndexNonASCII(t *testing.T) {
	reference := func(s string) int {
		for i, r := range s {
			if r >= utf8.RuneSelf {
				return i
			}
		}
		return -1
	}
	_ = reference

	for _, test := range indexTests {
		want := hasUnicode(test.s)
		n := IndexNonASCII(test.s)
		got := n >= 0
		if got != want {
			t.Errorf("IndexNonASCII(%q) = %t / %d; want: %t", test.s, got, n, want)
		}
	}

	// TODO: actually return the correct index
	long := strings.Repeat("a", 4096) + "β"
	for i := 0; i < len(long); i++ {
		n := IndexNonASCII(long[i:])
		if n == -1 {
			t.Fatalf("IndexNonASCII(%q) = %d", long[i:], n)
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
	{"αβδ", "ΑΒΔ", true, true},
	{"αβδΑΒΔ", "ΑΒΔ", true, false},
	{"abc", "xyz", false, false},
	{"abc", "XYZ", false, false},
	{"abc", "abc", true, true},
	{"abcd", "abc", true, false},
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

	// Test with runes that
	{
		strings.ToLower(string(multiwidthRunes[:])), // len: 36
		strings.ToUpper(string(multiwidthRunes[:])), // len: 54
		true,
		true,
	},
	{
		strings.ToUpper(string(multiwidthRunes[:])) + string(multiwidthRunes[:]),
		strings.ToLower(string(multiwidthRunes[:])) + string(multiwidthRunes[:]),
		true,
		true,
	},
	{
		strings.ToLower(string(multiwidthRunes[:len(multiwidthRunes)-1])),
		strings.ToUpper(string(multiwidthRunes[:])),
		false,
		true,
	},
	{
		strings.ToUpper(string(multiwidthRunes[:len(multiwidthRunes)-1])),
		strings.ToLower(string(multiwidthRunes[:])),
		false,
		true,
	},
}

func TestHasPrefixUnicode(t *testing.T) {
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

// func TestBruteForceIndexASCII(t *testing.T) {
// 	for _, test := range indexTests {
// 		if len(test.s) > maxLen || len(test.sep) > maxBruteForce || test.sep == "" {
// 			continue
// 		}
// 		out := bruteForceIndexASCII(test.s, test.sep)
// 		if out != test.out {
// 			t.Errorf("bruteForceIndexASCII(%q, %q) = %d; want %d", test.s, test.sep, out, test.out)
// 		}
// 	}
// }

// {"x012345678x0123456789", "0123456789", 11},

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
			for _, tt := range CompareTests {
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
	s := strings.Repeat(strings.ReplaceAll(benchmarkString, "☺", "K" /* Kelvin */), 16) + "x"
	benchmarkIndex(b, s, "Kvaluex")
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

func BenchmarkIndexHard1(b *testing.B) { benchmarkIndexHard(b, "<>") }
func BenchmarkIndexHard2(b *testing.B) { benchmarkIndexHard(b, "</pre>") }
func BenchmarkIndexHard3(b *testing.B) { benchmarkIndexHard(b, "<b>hello world</b>") }
func BenchmarkIndexHard4(b *testing.B) {
	benchmarkIndexHard(b, "<pre><b>hello</b><strong>world</strong></pre>")
}

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
	const str = "xx0123456789012345678901234567890123456789012345678901234567890120123456789012345678901234567890123456xxx"
	for i := 0; i < b.N; i++ {
		IndexNonASCII(str)
	}
}

// func benchmarkHashPrefix(b *testing.B, s, prefix string) {
// 	for i := 0; i < b.N; i++ {
// 		hasPrefixUnicode(s, prefix)
// 	}
// }

func BenchmarkHasPrefix(b *testing.B) {
	for i := 0; i < b.N; i++ {
		hasPrefixUnicode(benchmarkString, benchmarkString)
	}
}

func BenchmarkHasPrefixTests(b *testing.B) {
	for i := 0; i < b.N; i++ {
		for _, test := range prefixTests {
			hasPrefixUnicode(test.s, test.prefix)
		}
	}
}

func BenchmarkHasPrefixHard(b *testing.B) {
	for i := 0; i < b.N; i++ {
		hasPrefixUnicode(benchInputHard, benchInputHard)
	}
}

// WARN: DELETE ME
func BenchmarkBruteForceIndexUnicode(b *testing.B) {
	const u = "ΒΔΑ" + "ΒΔΑ" + "ΒΔΑ" + "ΒΔΑ" + "ΒΔΑ"
	const s = u + "some_text=someΑΒΔvalue"
	const sep = "ΑΒΔ"
	for i := 0; i < b.N; i++ {
		if bruteForceIndexUnicode(s, sep) == -1 {
			b.Fatal("WAT")
		}
	}
}

func BenchmarkIsASCII(b *testing.B) {
	for i := 0; i < b.N; i++ {
		isASCII(benchmarkString)
	}
}
