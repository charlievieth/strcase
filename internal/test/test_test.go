package test

import (
	"math/rand"
	"regexp"
	"strings"
	"testing"
	"time"
	"unicode/utf8"
)

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
	tests := append(indexTests, unicodeIndexTests...)
	runIndexTests(t, index, "Regexp", tests, false)
}

// Make sure our reference implementation is correct.
func TestIndexRunesReference(t *testing.T) {
	// Test that the Index tests are valid
	reference := func(s, sep string) int {
		return IndexRunesReference([]rune(s), []rune(sep))
	}
	tests := append(indexTests, unicodeIndexTests...)
	runIndexTests(t, reference, "IndexReference", tests, false)
}

// Reference test using regex: this will identify bad test cases and is more
// accurate than our reference LastIndex (since it might have bugs).
func TestLastIndexRegex(t *testing.T) {
	index := func(s, sep string) int {
		a := regexp.MustCompile(`(?i)`+regexp.QuoteMeta(sep)).FindAllStringIndex(s, -1)
		if len(a) != 0 {
			return a[len(a)-1][0]
		}
		return -1
	}
	runIndexTests(t, index, "Regexp", lastIndexTests, false)
}

func TestLastIndexReference(t *testing.T) {
	reference := func(s, sep string) int {
		return LastIndexRunesReference([]rune(s), []rune(sep))
	}
	runIndexTests(t, reference, "LastIndexReference", lastIndexTests, false)
}

func TestLastIndexReference_XXX(t *testing.T) {
	want := 0
	got := IndexRunesReference([]rune{'\uFFFD', '\uFFFD', '\u0061', '\u0043', '\uFFFD'}, []rune{'\uFFFD'})
	if got != want {
		t.Errorf("got: %d want: %d", got, want)
	}
}

func TestHasPrefix(t *testing.T) {
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
}

func TestEqualRune(t *testing.T) {
	rr := rand.New(rand.NewSource(time.Now().UnixNano()))
	fails := 0
	invalid := 0
	for i := 0; i < 100_000; i++ {
		sr := rr.Int31n(utf8.MaxRune * 2)
		tr := sr
		switch n := rr.Int31n(100); {
		case n <= 10:
			tr = utf8.RuneError
		case n <= 50:
			tr = rr.Int31n(utf8.MaxRune * 2)
		}
		if !utf8.ValidRune(sr) || !utf8.ValidRune(tr) {
			invalid++
		}
		got := EqualRune(sr, tr)
		want := strings.EqualFold(string(sr), string(tr))
		if got != want {
			t.Errorf("EqualRune(%U, %U) = %t; want: %t", sr, tr, got, want)
			fails++
			if fails >= 50 {
				t.Fatal("Too many errors")
			}
		}
	}
}

func TestEncodedLen(t *testing.T) {
	s := []rune{'a', utf8.RuneError, 'Ꝃ', '𞤔', utf8.MaxRune + 1, utf8.UTFMax + 1}
	want := len(string(s))
	got := EncodedLen(s)
	if got != want {
		t.Errorf("EncodedLen(%q) = %d want: %d", string(s), got, want)
	}
}
