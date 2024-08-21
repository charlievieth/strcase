package bytcase

import (
	"testing"
	"unicode/utf8"

	"github.com/charlievieth/strcase/internal/test"
)

func TestCompare(t *testing.T) {
	test.Compare(t, test.ByteIndexFunc(Compare))
}

func TestEqualFold(t *testing.T) {
	test.EqualFold(t, test.ByteContainsFunc(EqualFold))
}

func TestIndex(t *testing.T) {
	test.Index(t, test.ByteIndexFunc(Index))
}

func TestIndexUnicode(t *testing.T) {
	test.IndexUnicode(t, test.ByteIndexFunc(Index))
}

// Test our use of bytealg.IndexString
func TestIndexNumeric(t *testing.T) {
	test.IndexNumeric(t, test.ByteIndexFunc(Index))
}

// Extensively test the handling of Kelvin K since it is three times the size
// of ASCII [Kk] it requires special handling.
func TestIndexKelvin(t *testing.T) {
	test.IndexKelvin(t, test.ByteIndexFunc(Index))
}

// Test the Rabin-Karp fallback logic directly since not all test cases will
// trigger it.
func TestRabinKarp(t *testing.T) {
	test.Index(t, test.WrapRabinKarp(
		test.ByteIndexFunc(indexRabinKarpUnicode),
	))
}

// Test the Rabin-Karp fallback logic directly since not all test cases will
// trigger it.
func TestRabinKarpUnicode(t *testing.T) {
	test.IndexUnicode(t, test.WrapRabinKarp(
		test.ByteIndexFunc(indexRabinKarpUnicode),
	))
}

func TestBruteForceIndexUnicode(t *testing.T) {
	test.IndexUnicode(t, func(s, substr string) int {
		n := len(substr)
		var size int
		if n > 0 {
			if substr[0] < utf8.RuneSelf {
				size = 1
			} else {
				_, size = utf8.DecodeRuneInString(substr)
			}
		}
		if len(s) == 0 || len(substr) == 0 || n == size {
			// Can't use brute-force here
			return Index([]byte(s), []byte(substr))
		}
		return bruteForceIndexUnicode([]byte(s), []byte(substr))
	})
}

func TestIndexAllocs(t *testing.T) {
	var (
		haystack = []byte("test世界İ")
		n1       = []byte("世界İ")
		n2       = []byte("t世")
		n3       = []byte("test世界İ")
	)
	allocs := testing.AllocsPerRun(1000, func() {
		if i := Index(haystack, n1); i != 4 {
			t.Fatalf("'s' at %d; want 4", i)
		}
		if i := Index(haystack, n2); i != 3 {
			t.Fatalf("'世' at %d; want 3", i)
		}
		if i := Index(haystack, n3); i != 0 {
			t.Fatalf("'İ' at %d; want 0", i)
		}
	})
	if allocs != 0 {
		t.Errorf("expected no allocations, got %f", allocs)
	}
}

func TestContains(t *testing.T) {
	test.Contains(t, test.ByteContainsFunc(Contains))
}

func TestContainsAny(t *testing.T) {
	test.ContainsAny(t, test.ByteContainsFunc(ContainsAny))
}

func TestLastIndex(t *testing.T) {
	test.LastIndex(t, test.ByteIndexFunc(LastIndex))
}

func TestIndexRune(t *testing.T) {
	test.IndexRune(t, test.ByteIndexRuneFunc(IndexRune))
}

func TestIndexRuneAllocs(t *testing.T) {
	haystack := []byte("test世界İ")
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
	if allocs != 0 {
		t.Errorf("expected no allocations, got %f", allocs)
	}
}

func TestIndexRuneCase(t *testing.T) {
	test.IndexRuneCase(t, test.ByteIndexRuneFunc(indexRuneCase))
}

func TestContainsRune(t *testing.T) {
	test.ContainsRune(t, func(s string, r rune) bool {
		return ContainsRune([]byte(s), r)
	})
}

func TestLastIndexRune(t *testing.T) {
	test.LastIndexRune(t, test.ByteIndexRuneFunc(lastIndexRune))
}

func TestIndexByte(t *testing.T) {
	test.IndexByte(t, test.ByteIndexByte(IndexByte))
}

func TestLastIndexByte(t *testing.T) {
	test.LastIndexByte(t, test.ByteIndexByte(LastIndexByte))
}

func TestIndexNonASCII(t *testing.T) {
	test.IndexNonASCII(t, func(s string) int {
		return IndexNonASCII([]byte(s))
	})
}

func TestContainsNonASCII(t *testing.T) {
	test.ContainsNonASCII(t, func(s string) bool {
		return ContainsNonASCII([]byte(s))
	})
}

func TestHasPrefix(t *testing.T) {
	test.HasPrefix(t, test.BytePrefixFunc(hasPrefixUnicode))
}

func TestTrimPrefix(t *testing.T) {
	test.TrimPrefix(t, test.ByteTrimFunc(TrimPrefix))
}

func TestHasSuffix(t *testing.T) {
	test.HasSuffix(t, func(s, suffix string) bool {
		return HasSuffix([]byte(s), []byte(suffix))
	})
}

func TestTrimSuffix(t *testing.T) {
	test.TrimSuffix(t, test.ByteTrimFunc(TrimSuffix))
}

func TestCount(t *testing.T) {
	test.Count(t, test.ByteIndexFunc(Count))
}

func TestTestIndexAny(t *testing.T) {
	test.IndexAny(t, test.ByteIndexFunc(IndexAny))
}

func TestTestLastIndexAny(t *testing.T) {
	test.LastIndexAny(t, test.ByteIndexFunc(LastIndexAny))
}

func TestCut(t *testing.T) {
	test.Cut(t, func(s, sep string) (before, after string, found bool) {
		b, a, ok := Cut([]byte(s), []byte(sep))
		return string(b), string(a), ok
	})
}

func TestCutPrefix(t *testing.T) {
	test.CutPrefix(t, func(s, sep string) (after string, found bool) {
		b, ok := CutPrefix([]byte(s), []byte(sep))
		return string(b), ok
	})
}

func TestCutSuffix(t *testing.T) {
	test.CutSuffix(t, func(s, sep string) (before string, found bool) {
		b, ok := CutSuffix([]byte(s), []byte(sep))
		return string(b), ok
	})
}
