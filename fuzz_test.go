package strcase

import (
	crand "crypto/rand"
	"fmt"
	"math"
	"math/big"
	"math/rand"
	"strconv"
	"strings"
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

// func TestMultiwidthRunes(t *testing.T) {
// 	for _, r := range multiwidthRunes {
// 		if !utf8.ValidRune(r) {
// 			t.Errorf("%U %c\n", r, r)
// 		} else {
// 			fmt.Printf("%U %c\n", r, r)
// 		}
// 	}
// }

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
	rangetable.Visit(unicode.Letter, func(r rune) {
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

// WARN: ignoring 'K' and 'İ' for now
func validRune(r rune) bool {
	// WARN: ignoring 'K' and 'İ' for now
	return utf8.ValidRune(r) && r != utf8.RuneError && r != 'K' && r != 'İ'
}

func randNonControlRune(rr *rand.Rand) (r rune) {
	var lo, hi, stride uint32
	tab := unicodeCategories
	if len(tab.R32) == 0 || rr.Intn(2) < 1 {
		rt := tab.R16[rr.Intn(len(tab.R16))]
		lo = uint32(rt.Lo)
		hi = uint32(rt.Hi)
		stride = uint32(rt.Stride)
	} else {
		rt := tab.R32[rr.Intn(len(tab.R32))]
		lo = rt.Lo
		hi = rt.Hi
		stride = rt.Stride
	}
	d := hi - lo
	if d == 0 {
		r = rune(lo)
	} else {
		m := uint32(rand.Intn(int((hi-lo)/stride) + 1))
		r = rune(lo) + rune(stride*m)
	}
	return r
}

func randASCII(rr *rand.Rand) byte {
	return byte(rand.Intn('~'-' '+1)) + ' '
}

func randRune(rr *rand.Rand) (r rune) {
	for {
		switch f := rr.Float64(); {
		case f < 0.1:
			r = multiwidthRunes[rr.Intn(len(multiwidthRunes))]
		case f < 0.3:
			r = foldableRunes[rr.Intn(len(foldableRunes))]
		case f < 0.6:
			r = randNonControlRune(rr)
		default:
			r = rune(randASCII(rr))
		}
		if validRune(r) {
			return r
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
}

func randString(rr *rand.Rand, n int) string {
	rs := make([]rune, n)
	for {
		for i := range rs {
			rs[i] = randRune(rr)
		}
		s := string(rs)
		if utf8.ValidString(s) {
			return s
		}
		panic(fmt.Sprintf("Generated invalid string: %q", s))
	}
}

func randCaseRunes(rr *rand.Rand, rs []rune) string {
	for i, r := range rs {
		if rr.Float64() < 0.50 {
			if unicode.IsUpper(r) {
				r = unicode.ToLower(r)
			} else {
				r = unicode.ToUpper(r)
			}
			rs[i] = r
		}
	}
	return string(rs)
}

func randCase(rr *rand.Rand, s string) string {
	return randCaseRunes(rr, []rune(s))
}

func replaceChar(rr *rand.Rand, s string) string {
	rs := []rune(s)
	for {
		i := rr.Intn(len(rs))
		r := randRune(rr)
		if unicode.ToLower(r) != unicode.ToLower(rs[i]) {
			rs[i] = r
			break
		}
	}
	return string(rs)
}

func runRandomTest(t *testing.T, fn func(t *testing.T, rr *rand.Rand)) {
	randInt := func() int64 {
		i, err := crand.Int(crand.Reader, big.NewInt(math.MaxInt64))
		if err != nil {
			t.Fatal(err)
		}
		return i.Int64()
	}
	for _, seed := range []int64{
		1,
		time.Now().UnixNano(),
		randInt(),
		randInt(),
	} {
		t.Run(fmt.Sprintf("%d", seed), func(t *testing.T) {
			t.Parallel()
			n := 400
			if testing.Short() {
				n = 50
			}
			rr := rand.New(rand.NewSource(seed))
			for i := 0; i < n; i++ {
				fn(t, rr)
			}
		})
		if t.Failed() && testing.Short() {
			return
		}
	}
}

// indexReference is a slow, but accurate case-insensitive version of strings.Index
func indexReference(s, sep string) int {
	rs := []rune(s)
	rp := []rune(sep)
	if len(rs) < len(rp) {
		return -1
	}

	runesHasPrefix := func(s, prefix []rune) bool {
		if len(s) >= len(prefix) {
			for i := 0; i < len(prefix); i++ {
				if s[i] != prefix[i] && unicode.ToLower(s[i]) != unicode.ToLower(prefix[i]) {
					return false
				}
			}
			return true
		}
		return false
	}

	sp := unicode.ToLower(rp[0])
	for i := 0; i < len(rs); i++ {
		sr := rs[i]
		if sr == sp || unicode.ToLower(sr) == sp {
			if runesHasPrefix(rs[i:], rp) {
				return len(string(rs[:i]))
			}
		}
	}
	return -1
}

func generateIndexArgs(t testing.TB, rr *rand.Rand, ascii bool) (s, sep string, out int) {
	intn := func(n int) int {
		if n <= 0 {
			return 0
		}
		return rr.Intn(n)
	}

	ns := rr.Intn(16) + 1
	if ascii {
		b := make([]byte, ns)
		for i := range b {
			b[i] = randASCII(rr)
		}
		s = string(b)
	} else {
		s = randString(rr, ns)
	}

	// Generate match
	if rr.Float64() < 0.5 {
		rs := []rune(s)
		nsep := intn(len(rs)-1) + 1
		o := intn(len(rs) - nsep)
		sep = randCaseRunes(rr, rs[o:o+nsep])

		for i := 0; i < 128; i++ {
			if out = indexReference(s, sep); out != -1 {
				return s, sep, out
			}
			o = intn(len(rs) - nsep)
			sep = randCaseRunes(rr, rs[o:o+nsep])
		}

		t.Log("Failed to generate Index args: trying again...")
		return generateIndexArgs(t, rr, ascii)
	}

	lower := strings.ToLower(s)
	sep = randString(rr, intn(len(s)/2)+1)
	for strings.Contains(lower, strings.ToLower(sep)) {
		sep = randString(rr, intn(len(s)/2)+1)
	}
	return s, sep, -1
}

// func BenchmarkGenerateIndexArgs(b *testing.B) {
// 	rr := rand.New(rand.NewSource(1))
// 	for i := 0; i < b.N; i++ {
// 		generateIndexArgs(b, rr, false)
// 	}
// }

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

func generateHasPrefixArgs(t *testing.T, rr *rand.Rand) (s, prefix string, match bool) {
	intn := func(n int) int {
		if n <= 0 {
			return 0
		}
		return rr.Intn(n)
	}

	check := func(s, prefix string) bool {
		rs := []rune(s)
		rp := []rune(prefix)
		if len(rs) < len(rp) {
			return false
		}
		for i := range rp {
			if unicode.ToLower(rs[i]) != unicode.ToLower(rp[i]) {
				return false
			}
		}
		return true
	}

	// WARN: just changed this and it might be wrong

	for n := 128; n > 0; n-- {
		ns := rr.Intn(60) + 4
		s = randString(rr, ns)
		rs := []rune(s)
		np := intn(len(rs)-1) + 1
		prefix = randCaseRunes(rr, rs[:np])

		// Generate match
		replace := rr.Float64() < 0.5
		for i := 0; i < 128; i++ {
			if check(s, prefix) {
				return s, prefix, true
			}
			if replace {
				prefix = replaceChar(rr, prefix)
			} else {
				prefix = randCase(rr, s[:np])
			}
		}
	}
	panic("Failed to generate a vaild HasPrefix args")
}

func TestHasPrefixFuzz(t *testing.T) {
	runRandomTest(t, func(t *testing.T, rr *rand.Rand) {
		// WARN: test if the subject is exhausted
		s, prefix, out := generateHasPrefixArgs(t, rr)
		got, _ := hasPrefixUnicode(s, prefix)
		if got != out {
			t.Errorf("hasPrefixUnicode(%q, %q) = %t; want: %t", s, prefix, got, out)
		}
	})
}

// WARN: check returned index !!!
func TestIndexNonASCIIFuzz(t *testing.T) {
	isASCII := func(s string) bool {
		for i := 0; i < len(s); i++ {
			if s[i] >= utf8.RuneSelf {
				return false
			}
		}
		return true
	}

	test := func(t *testing.T, name string, gen func(*rand.Rand, int) string) {
		t.Run(name, func(t *testing.T) {
			runRandomTest(t, func(t *testing.T, rr *rand.Rand) {
				s := gen(rr, rr.Intn(128))
				want := isASCII(s)
				got := IndexNonASCII(s) == -1
				if got != want {
					t.Errorf("IndexNonASCII(%q) = %t want: %t", s, got, want)
				}
			})
		})
	}

	test(t, "Unicode", randString)

	test(t, "ASCII", func(rr *rand.Rand, n int) string {
		b := make([]byte, n)
		for i := range b {
			b[i] = byte(rand.Intn('~' - ' '))
		}
		return string(b)
	})
}
