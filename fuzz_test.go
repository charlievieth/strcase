package strcase

import (
	crand "crypto/rand"
	"flag"
	"fmt"
	"math"
	"math/big"
	"math/rand"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"
	"unicode"
	"unicode/utf8"

	"golang.org/x/text/unicode/rangetable"
)

var multiwidthRunes = [...]rune{
	'\U00000130', // 304: İ => i
	'\U00001E9E', // 7838: ẞ => ß
	'\U00002126', // 8486: Ω => ω
	'\U0000212A', // 8490: K => k
	'\U0000212B', // 8491: Å => å
	'\U00002C62', // 11362: Ɫ => ɫ
	'\U00002C64', // 11364: Ɽ => ɽ
	'\U00002C6D', // 11373: Ɑ => ɑ
	'\U00002C6E', // 11374: Ɱ => ɱ
	'\U00002C6F', // 11375: Ɐ => ɐ
	'\U00002C70', // 11376: Ɒ => ɒ
	'\U00002C7E', // 11390: Ȿ => ȿ
	'\U00002C7F', // 11391: Ɀ => ɀ
	'\U0000A78D', // 42893: Ɥ => ɥ
	'\U0000A7AA', // 42922: Ɦ => ɦ
	'\U0000A7AB', // 42923: Ɜ => ɜ
	'\U0000A7AC', // 42924: Ɡ => ɡ
	'\U0000A7AD', // 42925: Ɬ => ɬ
	'\U0000A7AE', // 42926: Ɪ => ɪ
	'\U0000A7B0', // 42928: Ʞ => ʞ
	'\U0000A7B1', // 42929: Ʇ => ʇ
	'\U0000A7B2', // 42930: Ʝ => ʝ
}

// All multi-width runes
var multiwidthRunesMap = map[rune]bool{
	'\U00000130': true,
	'\U00000131': true,
	'\U0000017F': true,
	'\U0000023A': true,
	'\U0000023E': true,
	'\U0000023F': true,
	'\U00000240': true,
	'\U00000250': true,
	'\U00000251': true,
	'\U00000252': true,
	'\U0000025C': true,
	'\U00000261': true,
	'\U00000265': true,
	'\U00000266': true,
	'\U0000026A': true,
	'\U0000026B': true,
	'\U0000026C': true,
	'\U00000271': true,
	'\U0000027D': true,
	'\U00000282': true,
	'\U00000287': true,
	'\U0000029D': true,
	'\U0000029E': true,
	'\U00001C80': true,
	'\U00001C81': true,
	'\U00001C82': true,
	'\U00001C83': true,
	'\U00001C84': true,
	'\U00001C85': true,
	'\U00001C86': true,
	'\U00001C87': true,
	'\U00001E9E': true,
	'\U00001FBE': true,
	'\U00002126': true,
	'\U0000212A': true,
	'\U0000212B': true,
	'\U00002C62': true,
	'\U00002C64': true,
	'\U00002C65': true,
	'\U00002C66': true,
	'\U00002C6D': true,
	'\U00002C6E': true,
	'\U00002C6F': true,
	'\U00002C70': true,
	'\U00002C7E': true,
	'\U00002C7F': true,
	'\U0000A78D': true,
	'\U0000A7AA': true,
	'\U0000A7AB': true,
	'\U0000A7AC': true,
	'\U0000A7AD': true,
	'\U0000A7AE': true,
	'\U0000A7B0': true,
	'\U0000A7B1': true,
	'\U0000A7B2': true,
}

var foldableRunes []rune

func init() {
	// WARN WARN WARN WARN WARN WARN WARN
	// NEW
	// WARN WARN WARN WARN WARN WARN WARN
	rangetable.Visit(_Foldable, func(r rune) {
		foldableRunes = append(foldableRunes, r)
	})
}

var unicodeCategories = rangetable.Merge([]*unicode.RangeTable{
	// unicode.Cc,     // Cc is the set of Unicode characters in category Cc (Other, control).
	unicode.Cf,     // Cf is the set of Unicode characters in category Cf (Other, format).
	unicode.Co,     // Co is the set of Unicode characters in category Co (Other, private use).
	unicode.Cs,     // Cs is the set of Unicode characters in category Cs (Other, surrogate).
	unicode.Digit,  // Digit is the set of Unicode characters with the "decimal digit" property.
	unicode.Nd,     // Nd is the set of Unicode characters in category Nd (Number, decimal digit).
	unicode.Letter, // Letter/L is the set of Unicode letters, category L.
	unicode.L,
	unicode.Lm,    // Lm is the set of Unicode characters in category Lm (Letter, modifier).
	unicode.Lo,    // Lo is the set of Unicode characters in category Lo (Letter, other).
	unicode.Lower, // Lower is the set of Unicode lower case letters.
	unicode.Ll,    // Ll is the set of Unicode characters in category Ll (Letter, lowercase).
	unicode.Mark,  // Mark/M is the set of Unicode mark characters, category M.
	unicode.M,
	unicode.Mc,     // Mc is the set of Unicode characters in category Mc (Mark, spacing combining).
	unicode.Me,     // Me is the set of Unicode characters in category Me (Mark, enclosing).
	unicode.Mn,     // Mn is the set of Unicode characters in category Mn (Mark, nonspacing).
	unicode.Nl,     // Nl is the set of Unicode characters in category Nl (Number, letter).
	unicode.No,     // No is the set of Unicode characters in category No (Number, other).
	unicode.Number, // Number/N is the set of Unicode number characters, category N.
	unicode.N,
	// unicode.Other, // Other/C is the set of Unicode control and special characters, category C.
	// unicode.C,
	unicode.Pc,    // Pc is the set of Unicode characters in category Pc (Punctuation, connector).
	unicode.Pd,    // Pd is the set of Unicode characters in category Pd (Punctuation, dash).
	unicode.Pe,    // Pe is the set of Unicode characters in category Pe (Punctuation, close).
	unicode.Pf,    // Pf is the set of Unicode characters in category Pf (Punctuation, final quote).
	unicode.Pi,    // Pi is the set of Unicode characters in category Pi (Punctuation, initial quote).
	unicode.Po,    // Po is the set of Unicode characters in category Po (Punctuation, other).
	unicode.Ps,    // Ps is the set of Unicode characters in category Ps (Punctuation, open).
	unicode.Punct, // Punct/P is the set of Unicode punctuation characters, category P.
	unicode.P,
	unicode.Sc,    // Sc is the set of Unicode characters in category Sc (Symbol, currency).
	unicode.Sk,    // Sk is the set of Unicode characters in category Sk (Symbol, modifier).
	unicode.Sm,    // Sm is the set of Unicode characters in category Sm (Symbol, math).
	unicode.So,    // So is the set of Unicode characters in category So (Symbol, other).
	unicode.Space, // Space/Z is the set of Unicode space characters, category Z.
	unicode.Z,
	unicode.Symbol, // Symbol/S is the set of Unicode symbol characters, category S.
	unicode.S,
	unicode.Title, // Title is the set of Unicode title case letters.
	unicode.Lt,    // Lt is the set of Unicode characters in category Lt (Letter, titlecase).
	unicode.Upper, // Upper is the set of Unicode upper case letters.
	unicode.Lu,    // Lu is the set of Unicode characters in category Lu (Letter, uppercase).
	unicode.Zl,    // Zl is the set of Unicode characters in category Zl (Separator, line).
	unicode.Zp,    // Zp is the set of Unicode characters in category Zp (Separator, paragraph).
	unicode.Zs,    // Zs is the set of Unicode characters in category Zs (Separator, space).
}...)

func randASCII(rr *rand.Rand) byte {
	return byte(rr.Intn('~'-' '+1)) + ' '
}

var nonControlRunes = generateNonControlRunes()

func generateNonControlRunes() []rune {
	n := 0
	rangetable.Visit(unicodeCategories, func(rune) {
		n++
	})
	runes := make([]rune, 0, n)
	rangetable.Visit(unicodeCategories, func(r rune) {
		if r >= utf8.RuneSelf && r != utf8.RuneError && utf8.ValidRune(r) {
			runes = append(runes, r)
		}
	})
	return runes
}

func randNonControlRune(rr *rand.Rand) rune {
	return nonControlRunes[rr.Intn(len(nonControlRunes))]
}

func randRune(rr *rand.Rand) (r rune) {
	for {
		switch f := rr.Float64(); {
		case f <= 0.1:
			r = multiwidthRunes[rr.Intn(len(multiwidthRunes))]
		case f <= 0.2:
			// WARN WARN WARN WARN WARN WARN WARN
			// NEW
			// WARN WARN WARN WARN WARN WARN WARN
			r = foldableRunes[rr.Intn(len(foldableRunes))]
		case f <= 0.75:
			r = randNonControlRune(rr)
		default:
			r = rune(randASCII(rr))
		}
		return r
	}
}

func TestRandNonControlRune(t *testing.T) {
	N := 5_000
	if testing.Short() {
		N = 500
	}

	seen := make(map[rune]struct{}, N)
	rr := rand.New(rand.NewSource(1))
	for i := 0; i < N; i++ {
		r := randNonControlRune(rr)
		if _, ok := seen[r]; !ok {
			seen[r] = struct{}{}
		}
	}
	ff := float64(len(seen)) / float64(N) * 100
	if ff < 95.0 {
		t.Errorf("Only generated %d/%d (%.2f%%) random runes: want: %.2f%%",
			len(seen), N, ff, 95.0)
	}

	fails := 0
	for r := range seen {
		if r < utf8.RuneSelf {
			t.Errorf("ASCII: %q", r)
			fails++
		}
		if fails >= 50 {
			t.Fatal("Too many errors:", fails)
		}
	}
}

func TestRandRune(t *testing.T) {
	runRandomTest(t, func(t *testing.T, rr *rand.Rand) {
		// This test is executed between 50 and 400 times
		for i := 0; i < 40; i++ {
			r := randRune(rr)
			if !unicode.Is(unicodeCategories, r) {
				t.Errorf("Invalid: %q (%U)", string(r), r)
			}
		}
	})

	t.Run("Distribution", func(t *testing.T) {
		N := 5_000
		if testing.Short() {
			N = 500
		}

		seen := make(map[rune]int, N)
		rr := rand.New(rand.NewSource(1))
		for i := 0; i < N; i++ {
			seen[randRune(rr)]++
		}
		ff := float64(len(seen)) / float64(N) * 100
		if ff < 55.0 {
			t.Errorf("Only generated %d/%d (%.2f%%) random runes: want: %.1f%%",
				len(seen), N, ff, 55.0)
		}

		// Leave this here for debugging
		if false {
			type RuneCount struct {
				R rune
				N int
			}
			runes := make([]RuneCount, 0, len(seen))
			for r, n := range seen {
				runes = append(runes, RuneCount{r, n})
			}
			sort.Slice(runes, func(i, j int) bool {
				return runes[i].N >= runes[j].N
			})
			for i := 0; i < 10; i++ {
				r := runes[i]
				t.Logf("%d: %q: %d / %.2f%%\n", i, r.R, r.N,
					(float64(r.N)/float64(len(runes)))*100)
			}
		}
	})
}

func randRunes(rr *rand.Rand, n int, ascii bool) []rune {
	rs := make([]rune, n)
	if ascii {
		for i := range rs {
			rs[i] = rune(randASCII(rr))
		}
		return rs
	}

	hard := -1
	if rr.Float64() < 0.05 {
		hard = intn(rr, len(rs)-4)
	}

	for i := 0; i < len(rs); i++ {
		if i == hard {
			for ; i < len(rs); i++ {
				rs[i] = '\u212a'
			}
			continue
		}
		rs[i] = randRune(rr)
	}
	return rs
}

func randCaseRune(rr *rand.Rand, r rune, ascii bool) rune {
	ff := rr.Float64()
	switch {
	case ascii:
		if ff < 0.50 {
			if 'a' <= r && r <= 'z' {
				r -= 'a' - 'A'
			} else if 'A' <= r && r <= 'Z' {
				r += 'a' - 'A'
			}
		}
	case ff < 0.3:
		runes := make([]rune, 0, 3)
		sr := unicode.SimpleFold(r)
		for sr != r {
			runes = append(runes, sr)
			sr = unicode.SimpleFold(sr)
		}
		if len(runes) > 0 {
			r = runes[rr.Intn(len(runes))]
		}
	case ff < 0.7:
		if unicode.IsUpper(r) {
			r = unicode.ToLower(r)
		} else {
			r = unicode.ToUpper(r)
		}
	}
	return r
}

// TODO: replace usages to handle runes
func randCaseRunes(rr *rand.Rand, rs []rune, ascii bool) (ro []rune) {
	ro = make([]rune, len(rs))
	copy(ro, rs)
	for i, r := range rs {
		ro[i] = randCaseRune(rr, r, ascii)
	}
	return ro
}

func replaceOneRune(rr *rand.Rand, rs []rune, ascii bool) (ro []rune) {
	for n := 0; n < 128; n++ {
		i := rr.Intn(len(rs))
		var r rune
		if ascii {
			r = rune(randASCII(rr))
		} else {
			r = randRune(rr)
		}
		if r != rs[i] && unicode.ToLower(r) != unicode.ToLower(rs[i]) {
			ro = make([]rune, len(rs))
			copy(ro, rs)
			ro[i] = r
			return ro
		}
	}
	panic("failed to generate a valid replacement")
}

// var randomRuns = flag.Int("fuzz-count", 0, "Number of random tests to execute.")

var exhaustiveFuzz = flag.Bool("exhaustive", false, "Run exhaustive fuzz tests (slow).")

func runRandomTest(t *testing.T, fn func(t *testing.T, rr *rand.Rand)) {
	randInt := func() int64 {
		i, err := crand.Int(crand.Reader, big.NewInt(math.MaxInt64))
		if err != nil {
			t.Fatal(err)
		}
		return i.Int64()
	}
	n := 500
	if testing.Short() {
		n = 50
	}
	seeds := []int64{
		1,
		time.Now().UnixNano(),
		randInt(),
		randInt(),
	}
	if *exhaustiveFuzz {
		if testing.Short() {
			t.Fatal(`Cannot combine "-short" and "-exhaustive" flags`)
		}
		d := 500_000 * len(seeds)
		numCPU := runtime.NumCPU()
		if numCPU == 10 && runtime.GOOS == "darwin" {
			numCPU = 8
		}
		for i := len(seeds); i < numCPU; i++ {
			seeds = append(seeds, randInt())
		}
		n = d / len(seeds)
	}
	for _, seed := range seeds {
		seed := seed
		t.Run(fmt.Sprintf("%d", seed), func(t *testing.T) {
			t.Parallel()
			start := time.Now()
			rr := rand.New(rand.NewSource(seed))
			for i := 0; i < n; i++ {
				fn(t, rr)
			}
			if testing.Verbose() {
				t.Logf("duration: %s", time.Since(start))
			}
		})
		if t.Failed() && testing.Short() {
			return
		}
	}
}

// indexRunesReference is a slow, but accurate case-insensitive version of strings.Index
func indexRunesReference(s, sep []rune) int {
	runesHasPrefix := func(s, prefix []rune) bool {
		if len(s) < len(prefix) {
			return false
		}
		for i := 0; i < len(prefix); i++ {
			// if s[i] != prefix[i] && unicode.ToLower(s[i]) != unicode.ToLower(prefix[i]) {
			if unicode.ToLower(s[i]) != unicode.ToLower(prefix[i]) {
				return false
			}
		}
		return true
	}

	// runeLen := func(s []rune) (n int) {
	// 	for _, r := range s {
	// 		if r < utf8.RuneSelf {
	// 			n++
	// 		} else {
	// 			n += utf8.RuneLen(r)
	// 		}
	// 	}
	// 	return n
	// }

	n := len(sep)
	if n == 0 {
		return 0
	}
	if n > len(s) {
		return -1
	}
	sp := unicode.ToLower(sep[0])
	for i := 0; i < len(s); i++ {
		sr := s[i]
		if sr == sp || unicode.ToLower(sr) == sp {
			if runesHasPrefix(s[i:], sep) {
				return len(string(s[:i]))
				// return runeLen()
			}
		}
	}
	return -1
}

func intn(rr *rand.Rand, n int) int {
	if n <= 0 {
		return 0
	}
	return rr.Intn(n)
}

func generateIndexArgs(t testing.TB, rr *rand.Rand, ascii bool) (_s, _sep string, out int) {

	match := rr.Float64() < 0.5

	for lim := 32; lim > 0; lim-- {
		ns := rr.Intn(64) + 1
		s := randRunes(rr, ns, ascii)
		nsep := intn(rr, len(s)-1) + 1
		o := intn(rr, len(s)-nsep)
		for i := 0; i < 4; i++ {
			sep := s[o : o+nsep]
			if match {
				sep = randCaseRunes(rr, sep, ascii)
			} else {
				sep = replaceOneRune(rr, sep, ascii)
			}
			out := indexRunesReference(s, sep)
			if (match && out >= 0) || (!match && out == -1) {
				return string(s), string(sep), out
			}
		}
	}

	panic("Failed to generate valid Index args")
}

func TestIndexFuzz(t *testing.T) {
	runRandomTest(t, func(t *testing.T, rr *rand.Rand) {
		s, sep, out := generateIndexArgs(t, rr, false)
		got := Index(s, sep)
		if got != out {
			t.Errorf("Index\n"+
				"S:    %q\n"+
				"Sep:  %q\n"+
				"Got:  %d\n"+
				"Want: %d\n"+
				"\n"+
				"S:    %s\n"+
				"Sep:  %s\n"+
				"\n",
				s, sep, got, out, strconv.QuoteToASCII(s), strconv.QuoteToASCII(sep))
		}
	})
}

func TestIndexFuzzASCII(t *testing.T) {
	runRandomTest(t, func(t *testing.T, rr *rand.Rand) {
		s, sep, out := generateIndexArgs(t, rr, true)
		got := Index(s, sep)
		if got != out {
			t.Errorf("Index\n"+
				"S:    %q\n"+
				"Sep:  %q\n"+
				"Got:  %d\n"+
				"Want: %d\n"+
				"\n"+
				"S:    %s\n"+
				"Sep:  %s\n"+
				"\n",
				s, sep, got, out, strconv.QuoteToASCII(s), strconv.QuoteToASCII(sep))
		}
	})
}

func generateIndexRuneArgs(t testing.TB, rr *rand.Rand) (string, rune, int) {
	// WARN WARN WARN WARN
	// Using folds

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

	index := func(s []rune, r rune) int {
		ff := folds(r)
		for i, rr := range s {
			for _, rf := range ff {
				if rf == rr {
					return len(string(s[:i]))
				}
			}
		}
		// l := unicode.ToLower(r)
		// for i, rr := range s {
		// 	if rr == r || unicode.ToLower(rr) == l {
		// 		return len(string(s[:i]))
		// 	}
		// }
		return -1
	}
	contains := func(s []rune, r rune) bool { return index(s, r) != -1 }

	match := rr.Float64() < 0.5

	ns := rr.Intn(12) + 4
	s := randRunes(rr, ns, false)
	if !match {
		r := randRune(rr)
		for contains(s, r) {
			r = randRune(rr)
		}
		return string(s), r, -1
	}

	for lim := 64; lim > 0; lim-- {
		var r rune
		i := intn(rr, len(s))
		r = randCaseRune(rr, s[i], false)
		for n := 0; n < len(s)*2; n++ {
			i = intn(rr, len(s))
			r = randCaseRune(rr, s[i], false)
			s[i] = r                         // WARN
			return string(s), r, index(s, r) // WARN
			// if r == s[i] {
			// 	continue
			// }
			// if r != s[i] && !contains(s, r) {
			// 	s[i] = r
			// 	return string(s), r, index(s, r)
			// }
		}
		// Regenerate string
		ns = rr.Intn(15) + 1
		s = randRunes(rr, ns, false)
	}

	panic("Failed to generate valid IndexRune args")
}

func TestIndexRuneFuzz(t *testing.T) {
	var fails int32
	runRandomTest(t, func(t *testing.T, rr *rand.Rand) {
		s, r, out := generateIndexRuneArgs(t, rr)
		got := IndexRune(s, r)
		if got != out {
			_, foldable := _FoldMap[r]
			t.Errorf("IndexRune\n"+
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
				s, r, got, out,
				foldable,
				strconv.QuoteToASCII(s),
				strconv.QuoteToASCII(string(r)),
				strconv.QuoteToASCII(strings.ToLower(s)),
				strconv.QuoteToASCII(strings.ToLower(string(r))),
			)
			if n := atomic.AddInt32(&fails, 1); n >= 10 {
				if n == 10 {
					t.Fatal("Too many errors:", n)
				} else {
					t.FailNow() // exit
				}
			}
		}
	})
}

func generateHasPrefixArgs(t *testing.T, rr *rand.Rand, ascii bool) (_s, _prefix string, match, exhausted bool) {
	// Reference implementation of HasPrefix
	hasPrefixRunes := func(s, prefix []rune) (bool, bool) {
		if len(s) < len(prefix) {
			return false, true
		}
		var i int
		for i = 0; i < len(prefix); i++ {
			if s[i] != prefix[i] && unicode.ToLower(s[i]) != unicode.ToLower(prefix[i]) {
				return false, i == len(s)-1
			}
		}
		return true, i == len(s)
	}

	match = rr.Float64() <= 0.5

	for lim := 32; lim > 0; lim-- {
		s := randRunes(rr, rr.Intn(30)+2, ascii)

		for i := 0; i < 4; i++ {
			np := intn(rr, len(s)-1) + 1
			prefix := s[:np]
			if match {
				prefix = randCaseRunes(rr, prefix, ascii)
			} else {
				if rr.Float64() >= 0.75 {
					prefix = replaceOneRune(rr, prefix, ascii)
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

func TestHasPrefixFuzz(t *testing.T) {
	test := func(t *testing.T, ascii bool) {
		runRandomTest(t, func(t *testing.T, rr *rand.Rand) {
			s, prefix, out, exhausted := generateHasPrefixArgs(t, rr, ascii)
			got, ex := hasPrefixUnicode(s, prefix)
			if got != out || ex != exhausted {
				t.Errorf("hasPrefixUnicode(%q, %q) = %t, %t; want: %t, %t", s, prefix, got, ex, out, exhausted)
			}
		})
	}

	t.Run("Unicode", func(t *testing.T) { test(t, false) })
	t.Run("ASCII", func(t *testing.T) { test(t, true) })
}

// WARN: check returned index !!!
func TestIndexNonASCIIFuzz(t *testing.T) {
	genArgs := func(t *testing.T, rr *rand.Rand, ascii bool) (string, bool) {
		s := randRunes(rr, rr.Intn(256)+1, true)
		isASCII := rr.Float64() <= 0.5
		if !isASCII {
			i := rr.Intn(len(s))
			r := randRune(rr)
			for r < utf8.RuneSelf {
				r = randRune(rr)
			}
			s[i] = r
		}
		return string(s), isASCII
	}

	test := func(t *testing.T, ascii bool) {
		runRandomTest(t, func(t *testing.T, rr *rand.Rand) {
			s, want := genArgs(t, rr, ascii)
			got := IndexNonASCII(s) == -1
			if got != want {
				t.Errorf("IndexNonASCII(%q) = %t want: %t", s, got, want)
			}
		})
	}

	t.Run("Unicode", func(t *testing.T) { test(t, false) })
	t.Run("ASCII", func(t *testing.T) { test(t, true) })
}

func generateCompareArgs(t *testing.T, rr *rand.Rand, ascii bool) (string, string, int) {
	compareRunes := func(s, t []rune) int {
		for i := 0; i < len(s) && i < len(t); i++ {
			sr := unicode.ToLower(s[i])
			tr := unicode.ToLower(t[i])
			if sr != tr {
				if sr > tr {
					return 1
				}
				if sr < tr {
					return -1
				}
				return 0
				// return clamp(int(sr) - int(tr))
			}
		}
		return clamp(len(s) - len(t))
	}

	match := rr.Float64() <= 0.5

	for lim := 32; lim > 0; lim-- {
		s0 := randRunes(rr, rr.Intn(14)+2, ascii)
		for i := 0; i < 16; i++ {
			var s1 []rune
			if match {
				s1 = randCaseRunes(rr, s0, ascii)
			} else {
				// Change length
				switch ff := rr.Float64(); {
				case ff <= 0.25:
					s1 = s0[:rr.Intn(len(s0))]
				case ff <= 0.50:
					s1 = append(s0, s0[:rr.Intn(len(s0))]...)
				case ff <= 0.75:
					s1 = replaceOneRune(rr, s0, ascii)
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

func TestCompareFuzz(t *testing.T) {
	test := func(t *testing.T, ascii bool) {
		fn := func(t *testing.T, rr *rand.Rand) {
			s0, s1, want := generateCompareArgs(t, rr, ascii)
			got := Compare(s0, s1)
			if got != want {
				t.Errorf("Compare(%q, %q) = %d; want: %d\n"+
					"Args:\n"+
					"  s:   %s\n"+
					"  sep: %s\n"+
					"Lower:\n"+
					"  s:   %s\n"+
					"  sep: %s\n",
					s0, s1, got, want,
					strconv.QuoteToASCII(s0),
					strconv.QuoteToASCII(s1),
					strconv.QuoteToASCII(strings.ToLower(s0)),
					strconv.QuoteToASCII(strings.ToLower(s1)),
				)
			}
		}
		runRandomTest(t, fn)
	}

	t.Run("Unicode", func(t *testing.T) { test(t, false) })
	t.Run("ASCII", func(t *testing.T) { test(t, true) })
}

// Fuzz test indexRabinKarpFuzz since it is annoying to generate tests that
// always take this code path in Index.
func TestIndexRabinKarpFuzz(t *testing.T) {

	// valid returns true if s contains 2 or more runes, which matches how
	// we call indexRabinKarpUnicode from Index.
	valid := func(s string) bool {
		return len(s) >= 4 || utf8.RuneCountInString(s) >= 2
	}

	runRandomTest(t, func(t *testing.T, rr *rand.Rand) {
		var s, sep string
		var out int
		for {
			s, sep, out = generateIndexArgs(t, rr, false)
			if valid(s) && valid(sep) {
				break
			}
		}
		got := indexRabinKarpUnicode(s, sep)
		if got != out {
			t.Errorf("indexRabinKarpUnicode\n"+
				"S:    %q\n"+
				"Sep:  %q\n"+
				"Got:  %d\n"+
				"Want: %d\n"+
				"\n"+
				"S:    %s\n"+
				"Sep:  %s\n"+
				"\n",
				s, sep, got, out, strconv.QuoteToASCII(s), strconv.QuoteToASCII(sep))
		}
	})
}
