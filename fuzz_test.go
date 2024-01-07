// Copyright 2023 Charlie Vieth. All rights reserved.
// Use of this source code is governed by the MIT license.

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

// TODO: use generated tables
// TODO: remove this
var multiwidthRunes = [...]rune{
	0x006B, // 'k'
	0x0073, // 's'
	0x00DF, // 'ß'
	0x00E5, // 'å'
	0x017F, // 'ſ'
	0x023A, // 'Ⱥ'
	0x023E, // 'Ⱦ'
	0x023F, // 'ȿ'
	0x0240, // 'ɀ'
	0x0250, // 'ɐ'
	0x0251, // 'ɑ'
	0x0252, // 'ɒ'
	0x025C, // 'ɜ'
	0x0261, // 'ɡ'
	0x0265, // 'ɥ'
	0x0266, // 'ɦ'
	0x026A, // 'ɪ'
	0x026B, // 'ɫ'
	0x026C, // 'ɬ'
	0x0271, // 'ɱ'
	0x027D, // 'ɽ'
	0x0282, // 'ʂ'
	0x0287, // 'ʇ'
	0x029D, // 'ʝ'
	0x029E, // 'ʞ'
	0x03B9, // 'ι'
	0x03C9, // 'ω'
	0x0432, // 'в'
	0x0434, // 'д'
	0x043E, // 'о'
	0x0441, // 'с'
	0x0442, // 'т'
	0x044A, // 'ъ'
	0x0463, // 'ѣ'
	0x1C80, // 'ᲀ'
	0x1C81, // 'ᲁ'
	0x1C82, // 'ᲂ'
	0x1C83, // 'ᲃ'
	0x1C84, // 'ᲄ'
	0x1C85, // 'ᲅ'
	0x1C86, // 'ᲆ'
	0x1C87, // 'ᲇ'
	0x1E9E, // 'ẞ'
	0x1FBE, // 'ι'
	0x2126, // 'Ω'
	0x212A, // 'K'
	0x212B, // 'Å'
	0x2C62, // 'Ɫ'
	0x2C64, // 'Ɽ'
	0x2C65, // 'ⱥ'
	0x2C66, // 'ⱦ'
	0x2C6D, // 'Ɑ'
	0x2C6E, // 'Ɱ'
	0x2C6F, // 'Ɐ'
	0x2C70, // 'Ɒ'
	0x2C7E, // 'Ȿ'
	0x2C7F, // 'Ɀ'
	0xA78D, // 'Ɥ'
	0xA7AA, // 'Ɦ'
	0xA7AB, // 'Ɜ'
	0xA7AC, // 'Ɡ'
	0xA7AD, // 'Ɬ'
	0xA7AE, // 'Ɪ'
	0xA7B0, // 'Ʞ'
	0xA7B1, // 'Ʇ'
	0xA7B2, // 'Ʝ'
	0xA7C5, // 'Ʂ'
}

// Excludes categories: Cm Cc, and Other.
var unicodeCategories = rangetable.Merge([]*unicode.RangeTable{
	unicode.Cf,     // Cf is the set of Unicode characters in category Cf (Other, format).
	unicode.Co,     // Co is the set of Unicode characters in category Co (Other, private use).
	unicode.Cs,     // Cs is the set of Unicode characters in category Cs (Other, surrogate).
	unicode.Digit,  // Digit is the set of Unicode characters with the "decimal digit" property.
	unicode.Letter, // Letter/L is the set of Unicode letters, category L.
	unicode.Mark,   // Mark/M is the set of Unicode mark characters, category M.
	unicode.Number, // Number/N is the set of Unicode number characters, category N.
	unicode.Punct,  // Punct/P is the set of Unicode punctuation characters, category P.
	unicode.Space,  // Space/Z is the set of Unicode space characters, category Z.
	unicode.Symbol, // Symbol/S is the set of Unicode symbol characters, category S.
	unicode.Title,  // Title is the set of Unicode title case letters.
	unicode.Upper,  // Upper is the set of Unicode upper case letters.
	unicode.Zl,     // Zl is the set of Unicode characters in category Zl (Separator, line).
	unicode.Zp,     // Zp is the set of Unicode characters in category Zp (Separator paragraph).
	unicode.Zs,     // Zs is the set of Unicode characters in category Zs (Separator, space).
}...)

// TODO: generate these
var foldableRunes = generateFoldableRunes()

func generateFoldableRunes() []rune {
	n := 0
	for _, p := range _CaseFolds {
		if p.From != 0 {
			n++
		}
	}
	a := make([]rune, 0, n*2)
	for _, p := range _CaseFolds {
		if p.From != 0 {
			a = append(a, rune(p.From), rune(p.To))
		}
	}
	sort.Slice(a, func(i, j int) bool {
		return a[i] < a[j]
	})
	return a
}

func randASCII(rr *rand.Rand) byte {
	return byte(rr.Intn('~'-' '+1)) + ' '
}

func randRune(rr *rand.Rand) (r rune) {
	switch f := rr.Float64(); {
	case f <= 0.01:
		return invalidRune(rr)
	case f <= 0.05:
		return 'İ'
	case f <= 0.1:
		return multiwidthRunes[rr.Intn(len(multiwidthRunes))]
	case f <= 0.2:
		// TODO: is this correct?
		return foldableRunes[rr.Intn(len(foldableRunes))]
	case f <= 0.75:
		return rune(rr.Int31n(unicode.MaxRune)) // may be invalid
	default:
		return rune(randASCII(rr))
	}
}

func TestRandRune(t *testing.T) {
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

var invalidRunes = flag.Bool("invalid", false, "Run fuzz tests with invalid runes.")

func invalidRune(rr *rand.Rand) rune {
	const surrogateMin = 0xD800
	const surrogateMax = 0xDFFF
	return rune(rr.Intn(surrogateMax-surrogateMin) + surrogateMin)
}

func TestInvalidRune(t *testing.T) {
	rr := rand.New(rand.NewSource(time.Now().UnixNano()))
	for i := 0; i < 10_000; i++ {
		r := invalidRune(rr)
		if utf8.ValidRune(r) {
			t.Fatalf("utf8.ValidRune(%q) = %t; want: %t", r, true, false)
		}
	}
}

// TODO: use a randomly shuffled slice of all runes?
func appendRandRunes(rs []rune, rr *rand.Rand, n int, ascii bool) []rune {
	if rs == nil || cap(rs) < n {
		rs = make([]rune, n)
	} else {
		rs = rs[:n]
	}
	if *invalidRunes {
		for i := range rs {
			rs[i] = rune(rr.Int31n(unicode.MaxRune))
		}
		// Add an invalid rune half of the time
		if i := rr.Intn(len(rs)*2 + 1); i < len(rs) {
			rs[i] = invalidRune(rr)
		}
		return rs
	}
	if ascii {
		for i := range rs {
			rs[i] = rune(randASCII(rr))
		}
		return rs
	}
	hard := len(rs)
	if rr.Float64() < 0.05 {
		hard = intn(rr, len(rs)-4)
	}
	for i := 0; i < len(rs); i++ {
		if i == hard {
			j := i + 4
			for ; i < j && i < len(rs); i++ {
				rs[i] = '\u212a'
			}
			continue
		}
		rs[i] = randRune(rr)
	}
	return rs
}

// TODO: clean this up

func randRunes(rr *rand.Rand, n int, ascii bool) []rune {
	return appendRandRunes(nil, rr, n, ascii)
	// rs := make([]rune, n)
	// if *invalidRunes {
	// 	for i := range rs {
	// 		rs[i] = rune(rr.Int31())
	// 	}
	// 	return rs
	// }
	// if ascii {
	// 	for i := range rs {
	// 		rs[i] = rune(randASCII(rr))
	// 	}
	// 	return rs
	// }
	// hard := len(rs)
	// if rr.Float64() < 0.05 {
	// 	hard = intn(rr, len(rs)-4)
	// }
	// for i := 0; i < len(rs); i++ {
	// 	if i == hard {
	// 		j := i + 4
	// 		for ; i < j && i < len(rs); i++ {
	// 			rs[i] = '\u212a'
	// 		}
	// 		continue
	// 	}
	// 	rs[i] = randRune(rr)
	// }
	// return rs
}

func TestEqualFoldFuzz(t *testing.T) {
	// Test that we match strings.EqualFold
	runRandomTest(t, func(t testing.TB, rr *rand.Rand) {
		n := rr.Intn(15) + 1
		r0 := randRunes(rr, n, false)
		s0 := string(r0)
		s1 := string(randCaseRunes(rr, r0, false))

		want := strings.EqualFold(s0, s1)

		if got := HasPrefix(s0, s1); got != want {
			t.Errorf("HasPrefix(%q, %q) = %t; want: %t", s0, s1, got, want)
		}

		if got := HasSuffix(s0, s1); got != want {
			t.Errorf("HasSuffix(%q, %q) = %t; want: %t", s0, s1, got, want)
		}

		if got := Index(s0, s1) == 0; got != want {
			t.Errorf("Index(%q, %q) = %t; want: %t", s0, s1, got, want)
		}

		if got := EqualFold(s0, s1); got != want {
			t.Errorf("EqualFold(%q, %q) = %t; want: %t", s0, s1, got, want)
		}
	})
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
	case ff < 0.4:
		var runes [4]rune
		sr := unicode.SimpleFold(r)
		i := 0
		for sr != r {
			runes[i] = sr
			i++
			sr = unicode.SimpleFold(sr)
		}
		switch {
		case i == 1:
			r = runes[0]
		case i > 1:
			r = runes[rr.Intn(i)]
		}
	case ff < 0.7:
		if u, l, ok := toUpperLower(r); ok {
			if r != u {
				r = u
			} else if r != l {
				r = l
			}
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

func randCaseRunesInPlace(rr *rand.Rand, rs, ro []rune, ascii bool) []rune {
	ro = append(ro[:0], rs...)
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

func replaceOneRuneInPlace(rr *rand.Rand, rs, ro []rune, ascii bool) []rune {
	ro = append(ro[:0], rs...)
	for n := 0; n < 128; n++ {
		i := rr.Intn(len(rs))
		var r rune
		if ascii {
			r = rune(randASCII(rr))
		} else {
			r = randRune(rr)
		}
		if r != rs[i] && unicode.ToLower(r) != unicode.ToLower(rs[i]) {
			ro[i] = r
			return ro
		}
	}
	panic("failed to generate a valid replacement")
}

type testWrapper struct {
	*testing.T
	fails int32
}

func (c *testWrapper) check() {
	c.T.Helper()
	if n := atomic.AddInt32(&c.fails, 1); n >= 10 {
		if n == 10 {
			c.T.Fatal("Too many errors:", n)
		} else {
			c.T.FailNow()
		}
	}
}

func (c *testWrapper) Error(args ...interface{}) {
	c.T.Helper()
	c.T.Error(args...)
	c.check()
}

func (c *testWrapper) Errorf(format string, args ...interface{}) {
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

func (c *testWrapper) Fatal(args ...interface{}) {
	c.T.Helper()
	c.T.Fatal(args...)
	c.check()
}

func (c *testWrapper) Fatalf(format string, args ...interface{}) {
	c.T.Helper()
	c.T.Fatalf(format, args...)
	c.check()
}

var exhaustiveFuzz = flag.Bool("exhaustive", false, "Run exhaustive fuzz tests (slow).")

func cryptoRandInt(t testing.TB) int64 {
	var err error
	var bi *big.Int
	for i := 0; i < 4; i++ {
		bi, err = crand.Int(crand.Reader, big.NewInt(math.MaxInt64))
		if err == nil {
			break
		}
	}
	if err != nil {
		if t == nil {
			panic(err)
		}
		t.Fatal(err)
		panic("unreachable")
	}
	return bi.Int64()
}

func runRandomTest(t *testing.T, fn func(t testing.TB, rr *rand.Rand)) {
	n := 2_500
	if testing.Short() {
		n = 100
	}
	seeds := []int64{
		1,
		time.Now().UnixNano(),
		cryptoRandInt(t),
		cryptoRandInt(t),
	}
	if *exhaustiveFuzz {
		if testing.Short() {
			t.Fatal(`Cannot combine "-short" and "-exhaustive" flags`)
		}
		d := 1_000_000 * len(seeds)
		numCPU := runtime.NumCPU()
		if runtime.GOOS == "darwin" && runtime.GOARCH == "arm64" {
			// Avoid using low-power cores.
			if numCPU >= 8 {
				numCPU -= 2
			}
		}
		for i := len(seeds); i < numCPU; i++ {
			seeds = append(seeds, cryptoRandInt(t))
		}
		n = d / len(seeds)
		t.Logf("N: %d", n)
	}
	for _, seed := range seeds {
		seed := seed
		t.Run(fmt.Sprintf("%d", seed), func(t *testing.T) {
			t.Parallel()
			start := time.Now()
			tb := &testWrapper{T: t}
			rr := rand.New(rand.NewSource(seed))
			for i := 0; i < n; i++ {
				fn(tb, rr)
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
	rs := append([]rune(nil), s...)
	rsep := append([]rune(nil), sep...)
	for i := 0; i < len(rs); i++ {
		rs[i] = caseFold(rs[i])
	}
	for i := 0; i < len(rsep); i++ {
		rsep[i] = caseFold(rsep[i])
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

// lastIndexRunesReference is a slow, but accurate case-insensitive version of strings.Index
func lastIndexRunesReference(s, sep []rune) int {
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
			rs[i] = caseFold(rs[i])
		}
		for i := 0; i < len(rsep); i++ {
			rsep[i] = caseFold(rsep[i])
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

func intn(rr *rand.Rand, n int) int {
	if n <= 0 {
		return 0
	}
	return rr.Intn(n)
}

// WARN: dev only
func generateBruteForceIndexArgs(t testing.TB, rr *rand.Rand) (_s, _sep string, out int) {

	match := rr.Float64() < 0.5

	for lim := 32; lim > 0; lim-- {
		ns := rr.Intn(10) + 2
		s := randRunes(rr, ns, false)
		nsep := intn(rr, len(s)-2) + 2
		o := intn(rr, len(s)-nsep)
		for i := 0; i < 4; i++ {
			sep := s[o : o+nsep]
			if match {
				sep = randCaseRunes(rr, sep, false)
			} else {
				sep = replaceOneRune(rr, sep, false)
			}
			out := indexRunesReference(s, sep)
			if (match && out >= 0) || (!match && out == -1) {
				return string(s), string(sep), out
			}
		}
	}

	panic("Failed to generate valid Index args")
}

// WARN: dev only
func TestBruteForceIndexUnicodeFuzz(t *testing.T) {
	runRandomTest(t, func(t testing.TB, rr *rand.Rand) {
		s, sep, out := generateBruteForceIndexArgs(t, rr)
		got := bruteForceIndexUnicode(s, sep)
		if got != out {
			t.Errorf("bruteForceIndexUnicode\n"+
				"S:    %q\n"+
				"Sep:  %q\n"+
				"Got:  %d\n"+
				"Want: %d\n"+
				"\n"+
				"S:    %s\n"+
				"Sep:  %s\n"+
				"Lower:\n"+
				"S:    %s\n"+
				"Sep:  %s\n"+
				"\n",
				s, sep, got, out,
				strconv.QuoteToASCII(s),
				strconv.QuoteToASCII(sep),
				strconv.QuoteToASCII(strings.ToLower(s)),
				strconv.QuoteToASCII(strings.ToLower(sep)),
			)
		}
	})
}

func generateIndexArgs(t testing.TB, rr *rand.Rand, ascii bool) (_s, _sep string, out int) {
	const maxLength = 42

	match := rr.Float64() < 0.5
	for lim := maxLength; lim > 0; lim-- {
		ns := rr.Intn(maxLength) + 1
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
	runRandomTest(t, func(t testing.TB, rr *rand.Rand) {
		s, sep, out := generateIndexArgs(t, rr, false)
		got := Index(s, sep)
		if got != out {
			t.Errorf("Index\n"+
				"S:    %q\n"+
				"Sep:  %q\n"+
				"Got:  %d\n"+
				"Want: %d\n"+
				"\n"+
				"S:    %+q\n"+
				"Sep:  %+q\n"+
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

func TestIndexFuzzASCII(t *testing.T) {
	runRandomTest(t, func(t testing.TB, rr *rand.Rand) {
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

// TODO: merge with generateIndexArgs
func generateLastIndexArgs(t testing.TB, rr *rand.Rand, ascii bool) (_s, _sep string, out int) {
	match := rr.Float64() < 0.5

	for lim := 32; lim > 0; lim-- {
		// WARN WARN WARN
		ns := rr.Intn(32) + 1
		// ns := rr.Intn(64) + 1
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
			out := lastIndexRunesReference(s, sep)
			if (match && out >= 0) || (!match && out == -1) {
				return string(s), string(sep), out
			}
		}
	}

	panic("Failed to generate valid Index args")
}

func TestLastIndexFuzz(t *testing.T) {
	runRandomTest(t, func(t testing.TB, rr *rand.Rand) {
		s, sep, out := generateLastIndexArgs(t, rr, false)
		got := LastIndex(s, sep)
		if got != out {
			t.Errorf("LastIndex\n"+
				"S:    %q\n"+
				"Sep:  %q\n"+
				"Got:  %d\n"+
				"Want: %d\n"+
				"\n"+
				"S:    %s\n"+
				"Sep:  %s\n"+
				"Lower:\n"+
				"S:    %s\n"+
				"Sep:  %s\n"+
				"\n",
				s, sep, got, out,
				strconv.QuoteToASCII(s),
				strconv.QuoteToASCII(sep),
				strconv.QuoteToASCII(strings.ToLower(s)),
				strconv.QuoteToASCII(strings.ToLower(sep)),
			)
		}
	})
}

func generateIndexRuneArgs(t testing.TB, rr *rand.Rand) (string, rune, int) {
	index := func(s []rune, r rune) int {
		switch {
		case r == utf8.RuneError:
			for i, r := range string(s) {
				if r == utf8.RuneError {
					return i
				}
			}
			return -1
		case !utf8.ValidRune(r):
			return -1
		default:
			for i, rr := range s {
				if rr == r || caseFold(rr) == caseFold(r) {
					return len(string(s[:i]))
				}
			}
			return -1
		}
	}
	contains := func(s []rune, r rune) bool {
		return index(s, r) != -1
	}

	match := rr.Float64() < 0.5

	ns := rr.Intn(16) + 1
	s := randRunes(rr, ns, false)
	if !match {
		r := randRune(rr)
		for contains(s, r) {
			r = randRune(rr)
		}
		return string(s), r, -1
	} else {
		i := intn(rr, len(s))
		r := randCaseRune(rr, s[i], false)
		s[i] = r
		return string(s), r, index(s, r)
	}
}

func TestIndexRuneFuzz(t *testing.T) {
	runRandomTest(t, func(t testing.TB, rr *rand.Rand) {
		s, r, want := generateIndexRuneArgs(t, rr)
		got := IndexRune(s, r)
		if got != want {
			t.Errorf("IndexRune\n"+
				"S:    %q\n"+
				"Sep:  %q (%+q)\n"+
				"Got:  %d\n"+
				"Want: %d\n"+
				"\n"+
				"S:    %+q\n"+
				"Sep:  %+q\n"+
				"Lower:\n"+
				"S:    %+q\n"+
				"Sep:  %+q\n"+
				"\n",
				s, r, r, got, want,
				s, r,
				strings.ToLower(s), unicode.ToLower(r),
			)
		}
	})
}

func hasPrefixRunes(s, prefix []rune) (bool, bool) {
	if len(s) < len(prefix) {
		return false, true
	}
	var i int
	for i = 0; i < len(prefix); i++ {
		sr := caseFold(s[i])
		pr := caseFold(prefix[i])
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

func generateHasPrefixArgs(t testing.TB, rr *rand.Rand, ascii bool) (_s, _prefix string, match, exhausted bool) {
	match = rr.Float64() <= 0.5

	for lim := 32; lim > 0; lim-- {
		s := randRunes(rr, rr.Intn(15)+1, ascii)
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
		runRandomTest(t, func(t testing.TB, rr *rand.Rand) {
			s, prefix, out, exhausted := generateHasPrefixArgs(t, rr, ascii)
			got, ex := hasPrefixUnicode(s, prefix)
			if got != out || ex != exhausted {
				// t.Errorf("hasPrefixUnicode(%q, %q) = %t, %t; want: %t, %t", s, prefix, got, ex, out, exhausted)

				t.Errorf("hasPrefixUnicode\n"+
					"Got:     %t, %t\n"+
					"Want:    %t, %t\n"+
					"S:       %q\n"+
					"Prefix:  %q\n"+
					"\n"+
					"S:       %s\n"+
					"Prefix:  %s\n"+
					"\n",
					got, ex, out, exhausted, s, prefix,
					strconv.QuoteToASCII(s), strconv.QuoteToASCII(prefix))
			}
		})
	}

	t.Run("Unicode", func(t *testing.T) { test(t, false) })
	t.Run("ASCII", func(t *testing.T) { test(t, true) })
}

// WARN: remove once no longer needed
func hasSuffixRunes(s, suffix []rune) bool {
	return len(s) >= len(suffix) &&
		strings.EqualFold(string(s[len(s)-len(suffix):]), string(suffix))
}

func generateHasSuffixArgs(t testing.TB, rr *rand.Rand, ascii bool) (string, string, bool) {
	match := rr.Float64() <= 0.5

	for lim := 32; lim > 0; lim-- {
		s := randRunes(rr, rr.Intn(15)+1, ascii)
		for i := 0; i < 4; i++ {
			np := intn(rr, len(s)-1)
			suffix := s[np:]
			if match {
				suffix = randCaseRunes(rr, suffix, ascii)
			} else {
				if rr.Float64() >= 0.75 {
					suffix = replaceOneRune(rr, suffix, ascii)
				} else {
					suffix = append(s, s[:np]...) // len(suffix) > len(s)
				}
			}
			if got := hasSuffixRunes(s, suffix); got == match {
				return string(s), string(suffix), match
			}
		}
	}

	panic("Failed to generate vaild HasPrefix args")
}

func TestHasSuffixFuzz(t *testing.T) {
	test := func(t *testing.T, ascii bool) {
		runRandomTest(t, func(t testing.TB, rr *rand.Rand) {
			s, suffix, out := generateHasSuffixArgs(t, rr, ascii)
			got := HasSuffix(s, suffix)
			if got != out {
				t.Errorf("HasSuffix\n"+
					"Got:     %t\n"+
					"Want:    %t\n"+
					"S:       %q\n"+
					"Suffix:  %q\n"+
					"\n"+
					"S:       %s\n"+
					"Suffix:  %s\n"+
					"\n",
					got, out, s, suffix,
					strconv.QuoteToASCII(s), strconv.QuoteToASCII(suffix),
				)
			}
		})
	}

	t.Run("Unicode", func(t *testing.T) { test(t, false) })
	t.Run("ASCII", func(t *testing.T) { test(t, true) })
}

func generateCompareArgs(t testing.TB, rr *rand.Rand, ascii bool) (string, string, int) {
	compareRunes := func(s, t []rune) int {
		for i := 0; i < len(s) && i < len(t); i++ {
			sr := caseFold(s[i])
			tr := caseFold(t[i])
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
		fn := func(t testing.TB, rr *rand.Rand) {
			s0, s1, want := generateCompareArgs(t, rr, ascii)
			got := Compare(s0, s1)
			if got != want {
				t.Errorf("Compare\n"+
					"S:    %q\n"+
					"Sep:  %q\n"+
					"Got:  %d\n"+
					"Want: %d\n"+
					"\n"+
					"S:    %+q\n"+
					"Sep:  %+q\n"+
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
		}
		runRandomTest(t, fn)
	}

	t.Run("Unicode", func(t *testing.T) { test(t, false) })
	t.Run("ASCII", func(t *testing.T) { test(t, true) })
}

// TODO: this is almost identical to generateIndexArgs - merge
func generateIndexRabinKarpArgs(t testing.TB, rr *rand.Rand, ascii bool) (_s, _sep string, out int) {

	match := rr.Float64() < 0.5

	s := make([]rune, 32)
	sep := make([]rune, 32)
	for lim := 32; lim > 0; lim-- {
		ns := rr.Intn(30) + 2
		// s := randRunes(rr, ns, ascii)
		s = appendRandRunes(s[:0], rr, ns, ascii)
		nsep := intn(rr, len(s)-2) + 2
		o := intn(rr, len(s)-nsep)
		for i := 0; i < 4; i++ {
			// sep := s[o : o+nsep]
			// if match {
			// 	sep = randCaseRunes(rr, sep, ascii)
			// } else {
			// 	sep = replaceOneRune(rr, sep, ascii)
			// }
			xsep := s[o : o+nsep]
			if match {
				sep = randCaseRunesInPlace(rr, xsep, sep, ascii)
			} else {
				sep = replaceOneRuneInPlace(rr, xsep, sep, ascii)
			}
			out := indexRunesReference(s, sep)
			if (match && out >= 0) || (!match && out == -1) {
				return string(s), string(sep), out
			}
		}
	}

	panic("Failed to generate valid Index args")
}

// TODO: delete this test
//
// Fuzz test indexRabinKarpFuzz since it is annoying to generate tests that
// always take this code path in Index.
func TestIndexRabinKarpFuzz(t *testing.T) {
	// valid returns true if s contains 2 or more runes, which matches how
	// we call indexRabinKarpUnicode from Index.
	valid := func(s string) bool {
		if len(s) >= 4 {
			return true
		}
		n := 0
		for range s {
			n++
			if n >= 2 {
				return true
			}
		}
		return false
	}

	runRandomTest(t, func(t testing.TB, rr *rand.Rand) {
		var s, sep string
		var out int
		for {
			s, sep, out = generateIndexRabinKarpArgs(t, rr, false)
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

// TODO: use this
// type FuzzConfig struct {
// 	MinSize   int
// 	MaxSize   int
// 	SepSize   int
// 	SepOffset int
// 	Reference func(s, sep []rune) bool
// }
//
// func (c *FuzzConfig) Generate(t testing.TB, rr *rand.Rand) (s, sep []rune) {
// 	return nil, nil
// }

// func runesEqual(s, t []rune) bool {
// 	if len(s) != len(t) {
// 		return false
// 	}
// 	for i := 0; i < len(s); i++ {
// 		sr := s[i]
// 		tr := t[i]
// 		if tr == sr {
// 			continue
// 		}
// 		if tr < sr {
// 			tr, sr = sr, tr
// 		}
// 		if tr < utf8.RuneSelf {
// 			if 'A' <= sr && sr <= 'Z' && tr == sr+'a'-'A' {
// 				continue
// 			}
// 			return false
// 		}
// 		r := unicode.SimpleFold(sr)
// 		for r != sr && r < tr {
// 			r = unicode.SimpleFold(r)
// 		}
// 		if r == tr {
// 			continue
// 		}
// 		return false
// 	}
// 	return true
// }
