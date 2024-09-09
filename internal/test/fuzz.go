package test

import (
	crand "crypto/rand"
	"encoding/binary"
	"flag"
	"fmt"
	"io"
	"math/rand"
	"runtime"
	"sort"
	"strings"
	"sync/atomic"
	"testing"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/charlievieth/strcase/internal/tables"
	"github.com/charlievieth/strcase/internal/tables/assigned"
)

func init() {
	if len(assignedRunes) == 0 {
		panic("no assigned runes for Unicode version: " + unicode.Version)
	}
}

var exhaustiveFuzz = flag.Bool("exhaustive", false, "Run exhaustive fuzz tests (slow).")

// All assigned runes for the current Unicode version
var assignedRunes = assigned.AssignedRunes(unicode.Version)

// CEV: Curated list of categories that should include all foldable
// Unicode points. We used to check all Unicode categories but that
// took around 15-20ms.
var multiwidthRunes, foldableRunes = generateRuneTables(
	unicode.Upper,
	unicode.Lower,
	unicode.Title,
	unicode.Mark,
	unicode.Number,
	unicode.Symbol,
)

func FoldableRunes() []rune {
	return foldableRunes
}

func generateRuneTables(tables ...*unicode.RangeTable) ([]rune, []rune) {
	foldable := make([]rune, 0, 4096)
	multiWidth := make([]rune, 0, 128)
	for _, rt := range tables {
		visitTable(rt, func(r rune) {
			if rr := unicode.SimpleFold(r); rr != r {
				n := utf8.RuneLen(r)
				for rr != r {
					if utf8.RuneLen(rr) != n {
						multiWidth = append(multiWidth, rr)
					}
					foldable = append(foldable, rr)
					rr = unicode.SimpleFold(rr)
				}
				return // no point checking Upper/Lower
			}
			// This is slow and currently just adds 'İ' and 'ı' which
			// we don't fold, but keep it in case more odd characters
			// are added in future Unicode versions.
			if rr := unicode.ToLower(r); r != rr {
				if utf8.RuneLen(r) != utf8.RuneLen(rr) {
					multiWidth = append(multiWidth, rr)
				}
				foldable = append(foldable, r, rr)
			} else if rr := unicode.ToUpper(r); r != rr {
				if utf8.RuneLen(r) != utf8.RuneLen(rr) {
					multiWidth = append(multiWidth, rr)
				}
				foldable = append(foldable, r, rr)
			}
		})
	}
	if len(foldable) == 0 || len(multiWidth) == 0 {
		panic(fmt.Sprintf("failed to generate foldable (%d) / multi-width (%d) runes",
			len(foldable), len(multiWidth)))
	}
	return compact(foldable), compact(multiWidth)
}

type byRune []rune

func (b byRune) Len() int           { return len(b) }
func (b byRune) Less(i, j int) bool { return b[i] < b[j] }
func (b byRune) Swap(i, j int)      { b[i], b[j] = b[j], b[i] }

// Backport of slices.Compact
func compact(s []rune) []rune {
	if len(s) < 2 {
		return s
	}
	if !sort.IsSorted(byRune(s)) {
		sort.Sort(byRune(s))
	}
	for k := 1; k < len(s); k++ {
		if s[k] == s[k-1] {
			s2 := s[k:]
			for k2 := 1; k2 < len(s2); k2++ {
				if s2[k2] != s2[k2-1] {
					s[k] = s2[k2]
					k++
				}
			}
			return s[:k]
		}
	}
	return s
}

// visitTable visits all runes in the given RangeTable in order, calling fn for each.
func visitTable(rt *unicode.RangeTable, fn func(rune)) {
	for _, r16 := range rt.R16 {
		for r := rune(r16.Lo); r <= rune(r16.Hi); r += rune(r16.Stride) {
			fn(r)
		}
	}
	for _, r32 := range rt.R32 {
		for r := rune(r32.Lo); r <= rune(r32.Hi); r += rune(r32.Stride) {
			fn(r)
		}
	}
}

func cryptoRandInt(t testing.TB) int64 {
	var b [8]byte
	if _, err := io.ReadFull(crand.Reader, b[:]); err != nil {
		if t != nil {
			t.Fatal(err)
		}
		panic(err)
	}
	return int64(binary.LittleEndian.Uint64(b[:]))
}

func intn(rr *rand.Rand, n int) int {
	if n <= 0 {
		return 0
	}
	return rr.Intn(n)
}

func invalidRune(rr *rand.Rand) rune {
	const surrogateMin = 0xD800
	const surrogateMax = 0xDFFF
	n := rr.Int31n(surrogateMax - surrogateMin)
	if n&1 == 0 {
		return utf8.MaxRune + n + 1
	}
	return n + surrogateMin
}

func randRune(rr *rand.Rand) (r rune) {
	switch n := rr.Intn(100); {
	case n <= 1:
		if n == 1 {
			return rr.Int31n(255-utf8.RuneSelf) + utf8.RuneSelf
		}
		return invalidRune(rr)
	case n <= 3: // 1..2
		// Funky runes 'İ' and 'ı' have upper/lower case forms
		// but do not fold.
		if n&1 != 0 {
			return 'İ'
		}
		return 'ı'
	case n <= 10:
		return '\u212a' // Kelvin K
	case n <= 30:
		return multiwidthRunes[rr.Intn(len(multiwidthRunes))]
	case n <= 50:
		return foldableRunes[rr.Intn(len(foldableRunes))]
	case n <= 90:
		return assignedRunes[rr.Intn(len(assignedRunes))]
	default:
		return rr.Int31n(128)
	}
}

// appendRandRunes appends random runes to rs
func appendRandRunes(rs []rune, rr *rand.Rand, n int) []rune {
	if cap(rs) < n {
		rs = make([]rune, n)
	} else {
		rs = rs[:n]
	}
	for i := 0; i < len(rs); i++ {
		rs[i] = randRune(rr)
	}
	// TODO: append a run of runes with the same last byte
	// Randomly add a run of Kelvin K
	if rr.Float64() < 0.10 {
		n := intn(rr, len(rs)-4)
		for i := n; i < len(rs) && i < n+4; i++ {
			rs[i] = '\u212a'
		}
	}
	return rs
}

// randCaseRune will randomly change the case of rune r
func randCaseRune(rr *rand.Rand, r rune) rune {
	// Change the case 2/3 of the time
	if rr.Int31n(32) < 24 {
		r = unicode.SimpleFold(r)
	}
	return r
}

// changeRuneCase changes the case of runes in rs
func changeRuneCase(rr *rand.Rand, rs []rune) []rune {
	for i, r := range rs {
		rs[i] = randCaseRune(rr, r)
	}
	return rs
}

// nonMatchingRune returns a rune that does not fold to sr
func nonMatchingRune(rr *rand.Rand, sr rune) rune {
	for i := 0; i < 1024; i++ {
		tr := assignedRunes[rr.Intn(len(assignedRunes))]
		if !EqualRune(sr, tr) {
			return tr
		}
	}
	panic("failed to generate a valid replacement")
}

// replaceOneRune replaces one rune in rs with another rune that
// does not fold to the original
func replaceOneRune(rr *rand.Rand, rs []rune) []rune {
	if len(rs) == 0 {
		return rs
	}
	var i int
	if rr.Intn(8) < 6 {
		i = len(rs)/2 + intn(rr, len(rs)/2) // later index
	} else {
		i = rr.Intn(len(rs)) // any index
	}
	rs[i] = nonMatchingRune(rr, rs[i])
	return rs
}

func fuzzNumCPU() int {
	numCPU := runtime.NumCPU()
	if runtime.GOOS == "darwin" && runtime.GOARCH == "arm64" {
		// Avoid using all the cores.
		// NB(charlie): this is really only for my personal dev setup.
		if numCPU >= 8 {
			numCPU -= 2
		}
	}
	if numCPU < 1 {
		numCPU = 1
	}
	return numCPU
}

func randomTestSeeds(t *testing.T) []int64 {
	seeds := []int64{
		1,
		time.Now().UnixNano(),
		cryptoRandInt(t),
		cryptoRandInt(t),
	}
	if !testing.Short() {
		numCPU := fuzzNumCPU()
		for i := len(seeds); i < numCPU; i++ {
			seeds = append(seeds, cryptoRandInt(t))
		}
	}
	return seeds
}

func runRandomTest(t *testing.T, fn func(t *fuzzTest)) {
	if *exhaustiveFuzz && testing.Short() {
		t.Fatal(`Cannot combine "-short" and "-exhaustive" flags`)
	}
	// Count is the total number of test iterations to run.
	count := 2_500
	if testing.Short() {
		count /= 2
	}
	seeds := randomTestSeeds(t)
	if *exhaustiveFuzz {
		d := 4_000_000
		count = d / len(seeds)
		t.Logf("N: %d", count)
	}
	for _, seed := range seeds {
		seed := seed
		t.Run(fmt.Sprintf("%d", seed), func(t *testing.T) {
			t.Parallel()
			start := time.Now()
			if testing.Verbose() {
				t.Cleanup(func() { t.Logf("duration: %s", time.Since(start)) })
			}
			tt := newFuzzTest(t, seed)
			for i := 0; i < count; i++ {
				fn(tt)
			}
		})
		if t.Failed() && testing.Short() {
			return
		}
	}
}

type fuzzTest struct {
	testing.TB
	rr *rand.Rand
	// Scratch space for constructing test arguments
	haystack []rune
	needle   []rune
}

func newFuzzTest(t *testing.T, seed int64) *fuzzTest {
	if seed < 0 {
		seed = cryptoRandInt(t)
	}
	return &fuzzTest{
		TB:       &testWrapper{T: t},
		rr:       rand.New(rand.NewSource(seed)),
		haystack: make([]rune, 0, 32),
		needle:   make([]rune, 0, 32),
	}
}

func randSubSlice(rr *rand.Rand, rs []rune, min, max int) ([]rune, int) {
	if min < 0 {
		panic("non-positive min")
	}
	if max > len(rs) {
		panic("max larger than slice")
	}
	if max <= 0 {
		max = len(rs)
	}
	n := intn(rr, max-min) + min
	o := intn(rr, len(rs)-n)
	return rs[o : o+n], o
}

func (t *fuzzTest) IndexArgs(fn func(s, sep []rune) int) (_s, _sep string, out int) {
	const maxLength = 72

	n := t.rr.Intn(maxLength-1) + 1
	s := appendRandRunes(t.haystack[:0], t.rr, n)
	nsep := len(s)
	if nsep > 32 && t.rr.Float64() <= 0.5 {
		nsep = 32 // Test small separators
	}
	orig, _ := randSubSlice(t.rr, s, 1, nsep) // original separator (unmodified)
	sep := append(t.needle[:0], orig...)

	match := t.rr.Float64() <= 0.5
	if !match && t.rr.Float64() <= 0.1 {
		s = append(s, s[t.rr.Intn(len(s))]) // Make sep longer than s
	}
	for i := 0; i < 4; i++ {
		if match {
			// Range over the original
			for i, r := range orig {
				sep[i] = randCaseRune(t.rr, r)
			}
		} else {
			j := t.rr.Intn(len(sep))
			sep[j] = nonMatchingRune(t.rr, sep[j])
		}
		out := fn(s, sep)
		if match == (out >= 0) {
			return string(s), string(sep), out
		}
	}

	panic(fmt.Sprintf("Failed to generate valid Index args match: %t: s: %q sep: %q",
		match, string(s), string(orig)))
}

func (t *fuzzTest) CompareArgs() (_s, _sep string, out int) {
	clamp := func(n int) int {
		if n < 0 {
			return -1
		}
		if n > 0 {
			return 1
		}
		return 0
	}
	compareRunes := func(s, t []rune) int {
		for i := 0; i < len(s) && i < len(t); i++ {
			sr := tables.CaseFold(s[i])
			tr := tables.CaseFold(t[i])
			// Convert invalid runes to RuneError since that
			// is what utf8.DecodeRuneInString does.
			if !utf8.ValidRune(sr) {
				sr = utf8.RuneError
			}
			if !utf8.ValidRune(tr) {
				tr = utf8.RuneError
			}
			if sr != tr {
				return clamp(int(sr) - int(tr))
			}
		}
		return clamp(len(s) - len(t))
	}

	match := t.rr.Float64() <= 0.5

	for lim := 32; lim > 0; lim-- {
		s0 := appendRandRunes(t.haystack[:0], t.rr, t.rr.Intn(14)+2)
		for i := 0; i < 16; i++ {
			s1 := append(t.needle[:0], s0...)
			if match {
				n := t.rr.Intn(4) + 1
				for i, r := range s1 {
					if i%n == 0 {
						s1[i] = randCaseRune(t.rr, r)
					}
				}
			} else {
				// Change length
				switch n := t.rr.Intn(100); {
				case n <= 25:
					s1 = s1[:t.rr.Intn(len(s1))]
				case n <= 50:
					s1 = append(s1, s1[:t.rr.Intn(len(s1))]...)
				case n <= 75:
					s1 = replaceOneRune(t.rr, s1)
				}
			}
			ret := compareRunes(s0, s1)
			if match && ret != 0 {
				continue
			}
			return string(s0), string(s1), ret
		}
	}

	panic("Failed to generate vaild Compare args")
}

func CompareFuzz(t *testing.T, fn func(s0, s1 string) int) {
	runRandomTest(t, func(t *fuzzTest) {
		s0, s1, want := t.CompareArgs()
		got := fn(s0, s1)
		if got != want {
			t.Errorf("Compare\n"+
				"S:    %q\n"+
				"Sep:  %q\n"+
				"Got:  %d\n"+
				"Want: %d\n"+
				"\n"+
				"ASCII:\n"+
				"S:    %+q\n"+
				"Sep:  %+q\n"+
				"\n"+
				"Lower:\n"+
				"S:    %+q\n"+
				"Sep:  %+q\n"+
				"\n",
				s0, s1, got, want,
				s0, s1,
				strings.ToLower(s0), strings.ToLower(s1),
			)
		}
		if got == 0 && !strings.EqualFold(s0, s1) {
			t.Errorf("Compare(%q, %q) = 0 but EqualFold() = false", s0, s1)
		}
	})
}

func IndexFuzz(t *testing.T, fn IndexFunc) {
	runRandomTest(t, func(t *fuzzTest) {
		s, sep, out := t.IndexArgs(IndexRunesReference)
		got := fn(s, sep)
		if got != out {
			// Make sure that our calculated index is correct using the
			// a slow but accurate regex.
			actual := indexRegex(s, sep)
			if out != actual {
				t.Errorf("Invalid generated test: got: %d want: %d actual: %d\n"+
					"S:        %q\n"+
					"Sep:      %q\n"+
					"Got:      %d\n"+
					"Expected: %d\n"+
					"Actual:   %d\n"+
					"\n"+
					"ASCII:\n"+
					"S:    %+q\n"+
					"Sep:  %+q\n"+
					"\n"+
					"Lower:\n"+
					"S:    %+q\n"+
					"Sep:  %+q\n"+
					"\n",
					got, out, actual,
					s, sep, got, out, actual,
					s, sep,
					strings.ToLower(s), strings.ToLower(sep))
				return
			}
		}
		if got != out {
			t.Errorf("Index\n"+
				"S:    %q\n"+
				"Sep:  %q\n"+
				"Got:  %d\n"+
				"Want: %d\n"+
				"\n"+
				"ASCII:\n"+
				"S:    %+q\n"+
				"Sep:  %+q\n"+
				"\n"+
				"Lower:\n"+
				"S:    %+q\n"+
				"Sep:  %+q\n"+
				"\n",
				s, sep, got, out,
				s, sep,
				strings.ToLower(s), strings.ToLower(sep),
			)
		}
	})
}

func LastIndexFuzz(t *testing.T, fn IndexFunc) {
	runRandomTest(t, func(t *fuzzTest) {
		s, sep, out := t.IndexArgs(LastIndexRunesReference)
		got := fn(s, sep)
		if got != out {
			// Make sure that our calculated index is correct using the
			// a slow but accurate regex.
			actual := lastIndexRegex(s, sep)
			if out != actual {
				t.Errorf("Invalid generated test: got: %d want: %d actual: %d\n"+
					"S:        %q\n"+
					"Sep:      %q\n"+
					"Got:      %d\n"+
					"Expected: %d\n"+
					"Actual:   %d\n"+
					"\n"+
					"ASCII:\n"+
					"S:    %+q\n"+
					"Sep:  %+q\n"+
					"\n"+
					"Lower:\n"+
					"S:    %+q\n"+
					"Sep:  %+q\n"+
					"\n",
					got, out, actual,
					s, sep, got, out, actual,
					s, sep,
					strings.ToLower(s), strings.ToLower(sep))
				return
			}
		}
		if got != out {
			t.Errorf("Index\n"+
				"S:    %q\n"+
				"Sep:  %q\n"+
				"Got:  %d\n"+
				"Want: %d\n"+
				"\n"+
				"ASCII:\n"+
				"S:    %+q\n"+
				"Sep:  %+q\n"+
				"\n"+
				"Lower:\n"+
				"S:    %+q\n"+
				"Sep:  %+q\n"+
				"\n",
				s, sep, got, out,
				s, sep,
				strings.ToLower(s), strings.ToLower(sep),
			)
		}
	})
}

func EqualFoldFuzz(t *testing.T, fns ...TestFunc) {
	// Test that we match strings.EqualFold
	runRandomTest(t, func(t *fuzzTest) {
		n := t.rr.Intn(30) + 2
		r0 := appendRandRunes(t.haystack[:0], t.rr, n)
		r1 := append(t.needle[:0], r0...)
		if t.rr.Float64() <= 0.5 {
			r1 = changeRuneCase(t.rr, r1)
		} else {
			r1 = replaceOneRune(t.rr, r1)
		}
		s0 := string(r0)
		s1 := string(r1)
		want := strings.EqualFold(s0, s1)
		for _, d := range fns {
			if got := d.Contains(s0, s1); got != want {
				t.Errorf("%s(%q, %q) = %t; want: %t", d.Name, s0, s1, got, want)
			}
		}
	})
}

func (t *fuzzTest) HasPrefixArgs() (_s, _prefix string, match, exhausted bool) {
	match = t.rr.Float64() <= 0.5

	for lim := 32; lim > 0; lim-- {
		s := appendRandRunes(t.haystack[:0], t.rr, t.rr.Intn(16)+1)
		for i := 0; i < 4; i++ {
			np := intn(t.rr, len(s)-1) + 1
			prefix := append(t.needle[:0], s[:np]...)
			if match {
				prefix = changeRuneCase(t.rr, prefix)
			} else {
				if t.rr.Float64() >= 0.75 {
					prefix = replaceOneRune(t.rr, prefix)
				} else {
					prefix = append(s, s[:np]...) // len(prefix) > len(s)
				}
			}
			if got, exh := hasPrefixRunes(s, prefix); got == match {
				return string(s), string(prefix), match, exh
			}
		}
	}

	panic("Failed to generate vaild HasPrefix args")
}

func HasPrefixFuzz(t *testing.T, fn func(s0, s1 string) (bool, bool)) {
	runRandomTest(t, func(t *fuzzTest) {
		s, prefix, want, exhausted := t.HasPrefixArgs()
		got, ex := fn(s, prefix)
		if got != want || ex != exhausted {
			t.Errorf("HasPrefix\n"+
				"S:      %q\n"+
				"Prefix: %q\n"+
				"Got:    %t, %t\n"+
				"Want:   %t, %t\n"+
				"\n"+
				"ASCII:\n"+
				"S:    %+q\n"+
				"Sep:  %+q\n"+
				"\n"+
				"Lower:\n"+
				"S:    %+q\n"+
				"Sep:  %+q\n"+
				"\n",
				s, prefix, got, ex, want, exhausted,
				s, prefix,
				strings.ToLower(s), strings.ToLower(prefix),
			)
		}
	})
}

func (t *fuzzTest) HasSuffixArgs() (string, string, bool) {
	hasSuffix := func(s, suffix []rune) bool {
		if len(s) < len(suffix) {
			return false
		}
		return EqualRuneSlice(s[len(s)-len(suffix):], suffix)
	}

	match := t.rr.Float64() <= 0.5

	for lim := 32; lim > 0; lim-- {
		s := appendRandRunes(t.haystack[:0], t.rr, t.rr.Intn(16)+1)
		for i := 0; i < 4; i++ {
			np := intn(t.rr, len(s)-1)
			suffix := append(t.needle[:0], s[np:]...)
			if match {
				suffix = changeRuneCase(t.rr, suffix)
			} else {
				if t.rr.Float64() >= 0.75 {
					suffix = replaceOneRune(t.rr, suffix)
				} else {
					suffix = append(s, s[:np]...) // len(suffix) > len(s)
				}
			}
			if got := hasSuffix(s, suffix); got == match {
				return string(s), string(suffix), match
			}
		}
	}

	panic("Failed to generate vaild HasSuffix args")
}

func HasSuffixFuzz(t *testing.T, fn ContainsFunc) {
	runRandomTest(t, func(t *fuzzTest) {
		s, suffix, want := t.HasSuffixArgs()
		got := fn(s, suffix)
		if got != want {
			actual := hasSuffixRegex(s, suffix)
			if actual != want {
				t.Errorf("Invalid generated test: got: %t want: %t actual: %t\n"+
					"S:        %q\n"+
					"Suffix:   %q\n"+
					"Got:      %t\n"+
					"Expected: %t\n"+
					"Actual:   %t\n"+
					"\n"+
					"ASCII:\n"+
					"S:      %+q\n"+
					"Suffix: %+q\n"+
					"\n"+
					"Lower:\n"+
					"S:      %+q\n"+
					"Suffix: %+q\n"+
					"\n",
					got, want, actual,
					s, suffix, got, want, actual,
					s, suffix,
					strings.ToLower(s), strings.ToLower(suffix),
				)
				want = actual
			}
			t.Errorf("HasSuffix\n"+
				"S:      %q\n"+
				"Prefix: %q\n"+
				"Got:    %t\n"+
				"Want:   %t\n"+
				"\n"+
				"ASCII:\n"+
				"S:    %+q\n"+
				"Sep:  %+q\n"+
				"\n"+
				"Lower:\n"+
				"S:    %+q\n"+
				"Sep:  %+q\n"+
				"\n",
				s, suffix, got, want,
				s, suffix,
				strings.ToLower(s), strings.ToLower(suffix),
			)
		}
	})
}

var _ testing.TB = (*testWrapper)(nil)

// A testWrapper wraps a testing.T and will immediately fail the test
// if more that N errors occur.
type testWrapper struct {
	*testing.T
	fails int32
}

func (c *testWrapper) check() {
	c.T.Helper()
	if n := atomic.AddInt32(&c.fails, 1); n >= 10 {
		// We run tests in parallel so only call Fatal on the
		// test that crossed the threshold.
		if n == 10 {
			c.T.Fatal("Too many errors:", n)
		} else {
			c.T.FailNow() // Abort subsequent tests
		}
		panic(fmt.Sprintf("aborting test: too many errors: %d", n)) // unreachable
	}
}

func (c *testWrapper) Error(args ...any) {
	c.T.Helper()
	c.T.Error(args...)
	c.check()
}

func (c *testWrapper) Errorf(format string, args ...any) {
	c.T.Helper()
	c.T.Errorf(format, args...)
	c.check()
}

func (c *testWrapper) Fail() {
	c.T.Helper()
	c.T.Fail()
	c.check()
}

func (c *testWrapper) FailNow() {
	c.T.Helper()
	c.T.FailNow()
	c.check()
}

func (c *testWrapper) Fatal(args ...any) {
	c.T.Helper()
	c.T.Fatal(args...)
	c.check()
}

func (c *testWrapper) Fatalf(format string, args ...any) {
	c.T.Helper()
	c.T.Fatalf(format, args...)
	c.check()
}
