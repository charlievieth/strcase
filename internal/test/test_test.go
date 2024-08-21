package test

import (
	"regexp"
	"strings"
	"testing"
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
	runIndexTests(t, index, "Regexp", unicodeIndexTests, false)
}

// Make sure our reference implementation is correct.
func TestIndexRunesReference(t *testing.T) {
	// Test that the Index tests are valid
	reference := func(s, sep string) int {
		return IndexRunesReference([]rune(s), []rune(sep))
	}
	runIndexTests(t, reference, "IndexReference", unicodeIndexTests, false)
}

func TestLastIndexReference(t *testing.T) {
	reference := func(s, sep string) int {
		return LastIndexRunesReference([]rune(s), []rune(sep))
	}
	runIndexTests(t, reference, "LastIndexReference", lastIndexTests, false)
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
