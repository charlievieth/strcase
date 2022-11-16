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
	// '\U0000A7C5', // 42949: Ʂ => ʂ
	// '\U0000A7C5', // 42949: Ʂ => ʂ
	// '\U0000A7C5', // 42949: Ʂ => ʂ
	// '\U0000A7C5', // 42949: Ʂ => ʂ
	// '\U0000A7C5', // 42949: Ʂ => ʂ
	'\U0000A7AE', // 42926: Ɪ => ɪ
	// '\U0000A7B0', // 42928: Ʞ => ʞ
	// '\U0000A7B1', // 42929: Ʇ => ʇ
	// '\U0000A7B2', // 42930: Ʝ => ʝ
}

// var multiwidthRunes = [...]rune{
// 	'\U00000130' /* 'İ' */, '\U00000131' /* 'ı' */, '\U0000017F', /* 'ſ' */
// 	'\U0000023A' /* 'Ⱥ' */, '\U0000023E' /* 'Ⱦ' */, '\U0000023F', /* 'ȿ' */
// 	'\U00000240' /* 'ɀ' */, '\U00000250' /* 'ɐ' */, '\U00000251', /* 'ɑ' */
// 	'\U00000252' /* 'ɒ' */, '\U0000025C' /* 'ɜ' */, '\U00000261', /* 'ɡ' */
// 	'\U00000265' /* 'ɥ' */, '\U00000266' /* 'ɦ' */, '\U0000026A', /* 'ɪ' */
// 	'\U0000026B' /* 'ɫ' */, '\U0000026C' /* 'ɬ' */, '\U00000271', /* 'ɱ' */
// 	'\U0000027D' /* 'ɽ' */, '\U00000282' /* 'ʂ' */, '\U00000287', /* 'ʇ' */
// 	'\U0000029D' /* 'ʝ' */, '\U0000029E' /* 'ʞ' */, '\U00001C80', /* 'ᲀ' */
// 	'\U00001C81' /* 'ᲁ' */, '\U00001C82' /* 'ᲂ' */, '\U00001C83', /* 'ᲃ' */
// 	'\U00001C84' /* 'ᲄ' */, '\U00001C85' /* 'ᲅ' */, '\U00001C86', /* 'ᲆ' */
// 	'\U00001C87' /* 'ᲇ' */, '\U00001E9E' /* 'ẞ' */, '\U00001FBE', /* 'ι' */
// 	'\U00002126' /* 'Ω' */, '\U0000212A' /* 'K' */, '\U0000212B', /* 'Å' */
// 	'\U00002C62' /* 'Ɫ' */, '\U00002C64' /* 'Ɽ' */, '\U00002C65', /* 'ⱥ' */
// 	'\U00002C66' /* 'ⱦ' */, '\U00002C6D' /* 'Ɑ' */, '\U00002C6E', /* 'Ɱ' */
// 	'\U00002C6F' /* 'Ɐ' */, '\U00002C70' /* 'Ɒ' */, '\U00002C7E', /* 'Ȿ' */
// 	'\U00002C7F' /* 'Ɀ' */, '\U0000A78D' /* 'Ɥ' */, '\U0000A7AA', /* 'Ɦ' */
// 	'\U0000A7AB' /* 'Ɜ' */, '\U0000A7AC' /* 'Ɡ' */, '\U0000A7AD', /* 'Ɬ' */
// 	'\U0000A7AE' /* 'Ɪ' */, '\U0000A7B0' /* 'Ʞ' */, '\U0000A7B1', /* 'Ʇ' */
// 	'\U0000A7B2', /* 'Ʝ' */
// }

var foldableRunes []rune

func init() {
	var runes []rune
	for _, r16 := range unicode.Letter.R16 {
		for r := r16.Lo; r <= r16.Hi; r += r16.Stride {
			runes = append(runes, rune(r))
		}
	}
	for _, r32 := range unicode.Letter.R32 {
		for r := r32.Lo; r <= r32.Hi; r += r32.Stride {
			runes = append(runes, rune(r))
		}
	}
	foldableRunes = make([]rune, len(runes))
	copy(foldableRunes, runes)
}

var unicodeCategories = func() []*unicode.RangeTable {
	tabs := make([]*unicode.RangeTable, 0, len(unicode.Categories))
	for _, tab := range unicode.Categories {
		tabs = append(tabs, tab)
	}
	return tabs
}()

func validUnicode(r rune) bool {
	return unicode.In(r, unicodeCategories...)
}

func randRune(rr *rand.Rand) (r rune) {
	for i := 0; ; i++ {
		switch f := rr.Float64(); {
		case f < 0.1:
			r = multiwidthRunes[rr.Intn(len(multiwidthRunes))]
		case f < 0.3:
			r = foldableRunes[rr.Intn(len(foldableRunes))]
		case f < 0.6:
			for {
				r = rune(rr.Intn(unicode.MaxRune))
				if validUnicode(r) {
					break
				}
			}
		default:
			r = rune(rand.Intn('~' - ' '))
		}
		if validUnicode(r) {
			return r
		}
		if i > 1024 {
			panic("Failed to generate a vaild unicode rune")
		}
	}
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
		panic("WAT")
	}
}

func randCase(rr *rand.Rand, s string) string {
	rs := []rune(s)
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

func generateStringFn(t *testing.T, rr *rand.Rand, n int, fn func(s string) bool) string {
	t.Helper()
	s := randString(rr, n)
	for i := 0; !fn(s); i++ {
		s = randString(rr, n)
		if i >= 512 {
			t.Fatal("failed to generates string")
		}
	}
	return s
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
	crint := func() int64 {
		i, err := crand.Int(crand.Reader, big.NewInt(math.MaxInt64))
		if err != nil {
			t.Fatal(err)
		}
		return i.Int64()
	}
	for _, seed := range []int64{
		1,
		time.Now().UnixNano(),
		crint(),
		crint(),
	} {
		t.Run(fmt.Sprintf("%d", seed), func(t *testing.T) {
			fn(t, rand.New(rand.NewSource(seed)))
		})
		if t.Failed() {
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

	l0 := unicode.ToLower(rp[0])
	n := len(rs) - len(rp) + 1
	for i := 0; i < n; i++ {
		r := rs[0]
		if unicode.ToLower(r) == l0 {
			// WARN: this appears to be broken for multiwidth runes
			if strings.HasPrefix(strings.ToLower(string(rs[i:])), strings.ToLower(sep)) {
				return len(string(rs[:i]))
			}
		}
	}
	return -1
}

func generateIndexArgs(t *testing.T, rr *rand.Rand) (s, sep string, out int) {
	intn := func(n int) int {
		if n <= 0 {
			return 0
		}
		return rr.Intn(n)
	}

	ns := rr.Intn(16) + 1
	s = generateStringFn(t, rr, ns, utf8.ValidString)
	lower := strings.ToLower(s)

	// Generate match
	if rr.Float64() < 0.5 {
		nsep := intn(len(s)-1) + 1
		o := intn(len(s) - nsep)
		sep = randCase(rr, s[o:o+nsep])

		// for i := 0; strings.Index(lower, strings.ToLower(sep)) == -1; i++ {
		for i := 0; indexReference(s, sep) == -1; i++ {
			o = intn(len(s) - nsep)
			sep = randCase(rr, s[o:o+nsep])
			if i > 128 {
				return generateIndexArgs(t, rr)
			}
		}
		return s, sep, o
	}
	sep = randString(rr, intn(len(s)/2)+1)
	for strings.Index(lower, strings.ToLower(sep)) != -1 {
		sep = randString(rr, intn(len(s)/2)+1)
	}
	return s, sep, -1
}

func TestIndexFuzz(t *testing.T) {
	runRandomTest(t, func(t *testing.T, rr *rand.Rand) {
		for i := 0; i < 200; i++ {
			s, sep, out := generateIndexArgs(t, rr)
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
				// t.Errorf("Index(%q, %q) = %d; want: %d", s, sep, got, out)
			}
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
		srs := []rune(s)
		prs := []rune(prefix)
		if len(srs) < len(prs) {
			return false
		}
		for i := range prs {
			if unicode.ToLower(srs[i]) != unicode.ToLower(prs[i]) {
				return false
			}
		}
		return true
	}

	ns := rr.Intn(60) + 4
	s = randString(rr, ns)
	np := intn(len(s)-1) + 1
	prefix = randCase(rr, s[:np])

	// Generate match
	if rr.Float64() < 0.5 {
		for i := 0; !check(s, prefix); i++ {
			prefix = randCase(rr, s[:np])
			if i > 128 {
				return generateHasPrefixArgs(t, rr)
			}
		}
		return s, prefix, true
	}
	prefix = replaceChar(rr, prefix)
	return s, prefix, false
}

func TestHasPrefixFuzz(t *testing.T) {
	runRandomTest(t, func(t *testing.T, rr *rand.Rand) {
		for i := 0; i < 200; i++ {
			// WARN WARN WARN: test Exhausted
			s, prefix, out := generateHasPrefixArgs(t, rr)
			got, _ := hasPrefixUnicode(s, prefix)
			if got != out {
				t.Errorf("hasPrefixUnicode(%q, %q) = %t; want: %t", s, prefix, got, out)
			}
		}
	})
}
