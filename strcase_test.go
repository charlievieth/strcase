// Copyright 2023 Charlie Vieth. All rights reserved.
// Use of this source code is governed by the MIT license.

package strcase

import (
	"flag"
	"fmt"
	"math/rand"
	"sort"
	"strconv"
	"strings"
	"testing"
	"unicode"
	"unicode/utf8"
	"unsafe"

	"github.com/charlievieth/strcase/internal/tables"
	"github.com/charlievieth/strcase/internal/test"
)

func TestUnicodeVersion(t *testing.T) {
	test.UnicodeVersion(t, UnicodeVersion)
}

func TestCompare(t *testing.T) {
	test.Compare(t, Compare)
}

func TestEqualFold(t *testing.T) {
	test.EqualFold(t, EqualFold)
}

func TestIndex(t *testing.T) {
	test.Index(t, Index)
}

func TestIndexUnicode(t *testing.T) {
	test.IndexUnicode(t, Index)
}

// Test our use of bytealg.IndexString
func TestIndexNumeric(t *testing.T) {
	test.IndexNumeric(t, Index)
}

// Extensively test the handling of Kelvin K since it is three times the size
// of ASCII [Kk] it requires special handling.
func TestIndexKelvin(t *testing.T) {
	test.IndexKelvin(t, Index)
}

// Test the Rabin-Karp fallback logic directly since not all test cases will
// trigger it.
func TestRabinKarp(t *testing.T) {
	test.Index(t, test.WrapRabinKarp(indexRabinKarpUnicode))
}

// Test the Rabin-Karp fallback logic directly since not all test cases will
// trigger it.
func TestRabinKarpUnicode(t *testing.T) {
	test.IndexUnicode(t, test.WrapRabinKarp(indexRabinKarpUnicode))
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
			return Index(s, substr)
		}
		return bruteForceIndexUnicode(s, substr)
	})
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
	if allocs != 0 {
		t.Errorf("expected no allocations, got %f", allocs)
	}
}

func TestContains(t *testing.T) {
	test.Contains(t, Contains)
}

func TestContainsAny(t *testing.T) {
	test.ContainsAny(t, ContainsAny)
}

func TestLastIndex(t *testing.T) {
	test.LastIndex(t, LastIndex)
}

func TestIndexRune(t *testing.T) {
	test.IndexRune(t, IndexRune)
}

func TestIndexRuneAllocs(t *testing.T) {
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
	if allocs != 0 {
		t.Errorf("expected no allocations, got %f", allocs)
	}
}

func TestIndexRuneCase(t *testing.T) {
	test.IndexRuneCase(t, indexRuneCase)
}

func TestContainsRune(t *testing.T) {
	test.ContainsRune(t, ContainsRune)
}

func TestLastIndexRune(t *testing.T) {
	test.LastIndexRune(t, lastIndexRune)
}

func TestIndexByte(t *testing.T) {
	test.IndexByte(t, IndexByte)
}

func TestLastIndexByte(t *testing.T) {
	test.LastIndexByte(t, LastIndexByte)
}

func TestIndexNonASCII(t *testing.T) {
	test.IndexNonASCII(t, IndexNonASCII)
}

func TestContainsNonASCII(t *testing.T) {
	test.ContainsNonASCII(t, ContainsNonASCII)
}

func TestHasPrefix(t *testing.T) {
	test.HasPrefix(t, hasPrefixUnicode)
}

func TestTrimPrefix(t *testing.T) {
	test.TrimPrefix(t, TrimPrefix)
}

func TestHasSuffix(t *testing.T) {
	test.HasSuffix(t, HasSuffix)
}

func TestTrimSuffix(t *testing.T) {
	test.TrimSuffix(t, TrimSuffix)
}

func TestCount(t *testing.T) {
	test.Count(t, Count)
}

func TestTestIndexAny(t *testing.T) {
	test.IndexAny(t, IndexAny)
}

func TestTestLastIndexAny(t *testing.T) {
	test.LastIndexAny(t, LastIndexAny)
}

func TestCut(t *testing.T) {
	test.Cut(t, Cut)
}

func TestCutPrefix(t *testing.T) {
	test.CutPrefix(t, CutPrefix)
}

func TestCutSuffix(t *testing.T) {
	test.CutSuffix(t, CutSuffix)
}

func TestToUpperLower(t *testing.T) {
	fails := 0
	for _, rt := range unicode.Categories {
		visitTable(rt, func(r rune) {
			l := unicode.ToLower(r)
			u := unicode.ToUpper(r)
			ok := l != u
			uu, ll, found := tables.ToUpperLower(r)
			if l != ll || u != uu || ok != found {
				t.Errorf("ToUpperLower(%c) = %c, %c, %t want: %c, %c, %t",
					r, ll, uu, found, l, u, ok)
				fails++
			}
			if fails >= 50 {
				t.Fatal("Too many errors:", fails)
			}
		})
	}
}

func BenchmarkCompare(b *testing.B) {
	bench := func(b *testing.B, s, t string) {
		b.Helper()
		n := len(s)
		if len(t) < n {
			n = len(t)
		}
		b.SetBytes(int64(n))
		for i := 0; i < b.N; i++ {
			Compare(s, t)
		}
	}

	const s1 = "abcdefghijKz"
	const s2 = "abcDefGhijKz"

	b.Run("ASCII", func(b *testing.B) {
		bench(b, s1, s2)
	})

	b.Run("ASCII_Long", func(b *testing.B) {
		const s = s1 + s1 + s1 + s1 + s1
		const t = s2 + s2 + s2 + s2 + s2
		bench(b, s, t)
	})

	b.Run("UnicodePrefix", func(b *testing.B) {
		// WARN
		const s1 = "AbCdCfghIjKz"
		const s2 = "abcDeFGhijKz"
		bench(b, "αβδ"+s1, "ΑΒΔ"+s2)
	})

	b.Run("UnicodeSuffix", func(b *testing.B) {
		bench(b, s1+"αβδ", s2+"ΑΒΔ")
	})

	b.Run("Russian", func(b *testing.B) {
		b.SetBytes(int64(len(russianText)))
		bench(b, russianText, russianText)
	})
}

func TestCaseFold(t *testing.T) {
	t.Run("Limits", func(t *testing.T) {
		for r := unicode.MaxRune; r < unicode.MaxRune+10; r++ {
			x := tables.CaseFold(r)
			if x != r {
				t.Errorf("caseFold(0x%04X) = 0x%04X; want: 0x%04X", r, x, r)
			}
		}
		for r := rune(0); r < ' '; r++ {
			x := tables.CaseFold(r)
			if x != r {
				t.Errorf("caseFold(0x%04X) = 0x%04X; want: 0x%04X", r, x, r)
			}
		}
		if r := tables.CaseFold(utf8.RuneError); r != utf8.RuneError {
			t.Errorf("caseFold(0x%04X) = 0x%04X; want: 0x%04X", utf8.RuneError, r, utf8.RuneError)
		}
	})
}

// Ensure that strings.EqualFold does not match 'İ' (U+0130) and ASCII 'i' or 'I'.
// This is mostly a sanity check.
func TestLatinCapitalLetterIWithDotAbove(t *testing.T) {
	if strings.EqualFold("İ", "i") {
		t.Errorf("strings.EqualFold(%q, %q) = true; want: false", "İ", "i")
	}
	if strings.EqualFold("İ", "I") {
		t.Errorf("strings.EqualFold(%q, %q) = true; want: false", "İ", "I")
	}
	if Compare("İ", "i") == 0 {
		t.Errorf("Compare(%q, %q) = true; want: false", "İ", "i")
	}
	if Compare("İ", "I") == 0 {
		t.Errorf("Compare(%q, %q) = true; want: false", "İ", "I")
	}
}

func TestNonLetterASCII(t *testing.T) {
	tests := []struct {
		s    string
		want bool
	}{
		{"", true},
		{"1234", true},
		{"1a", false},
		{"1A", false},
	}
	for _, test := range tests {
		got := nonLetterASCII(test.s)
		if got != test.want {
			t.Errorf("nonLetterASCII(%q) = %t; want: %t", test.s, got, test.want)
		}
	}
}

const benchmarkString = "some_text=some☺value"

// WARN: dev only
func BenchmarkIndexRuneRussian(b *testing.B) {
	want := strings.IndexRune(russianText, 'ж')
	if got := IndexRune(russianText, 'ж'); got != want {
		b.Fatalf("got: %d want: %d", got, want)
	}
	b.SetBytes(int64(len(russianText)))
	for i := 0; i < b.N; i++ {
		IndexRune(russianText, 'ж')
	}
}

func BenchmarkIndexRune(b *testing.B) {
	// const str = benchmarkString + "\u212a"
	const str = benchmarkString + "k"
	// const str = benchmarkString + string(rune(0x212A))
	// if got := IndexRune(benchmarkString, '☺'); got != 14 {
	if got := IndexRune(str, rune(0x212A)); got != 22 {
		b.Fatalf("wrong index: expected 14, got=%d", got)
	}
	for i := 0; i < b.N; i++ {
		IndexRune(benchmarkString, '☺')
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

// Benchmark buffer
var bmbuf []byte

func valName(x int) string {
	if s := x >> 20; s<<20 == x {
		return fmt.Sprintf("%dM", s)
	}
	if s := x >> 10; s<<10 == x {
		return fmt.Sprintf("%dK", s)
	}
	return fmt.Sprint(x)
}

var indexSizes = []int{10, 32, 4 << 10, 4 << 20, 64 << 20}

func benchBytesUnicode(b *testing.B, sizes []int, f func(b *testing.B, n int, s string)) {
	// These character all have the same second byte (0x90)
	const _s = "𐀀𐀁𐀂𐀃𐀄𐀅𐀆𐀇𐀈𐀉𐀊𐀋𐀍𐀎𐀏𐀐𐀑𐀒𐀓𐀔𐀕𐀖𐀗𐀘𐀙𐀚𐀛𐀜𐀝𐀞𐀟𐀠"
	const s = _s + _s + _s + _s + _s + _s + _s + _s + _s + _s + _s + _s + _s + _s + _s + _s // 2048
	for _, n := range sizes {
		b.Run(valName(n), func(b *testing.B) {
			if len(bmbuf) < n {
				bmbuf = make([]byte, n)
			}
			for i := 0; i < n; {
				i += copy(bmbuf[i:], s)
			}
			copy(bmbuf[n-len("𐀤"):], "𐀤")
			b.SetBytes(int64(n))
			f(b, n, string(bmbuf))
		})
	}
}

func bmIndexRune(index func(string, rune) int) func(b *testing.B, n int, s string) {
	return func(b *testing.B, n int, s string) {
		// Sanity check since I got this wrong in the past
		want := strings.IndexRune(s, '𐀤')
		got := index(s, '𐀤')
		if want != got {
			b.Fatalf("bad index %d want: %d", got, want)
		}
		if got != n-4 {
			b.Fatalf("bad index %d want: %d", got, n-4)
		}
		for i := 0; i < b.N; i++ {
			_ = index(s, '𐀤')
		}
	}
}

func benchBytes(b *testing.B, sizes []int, f func(b *testing.B, n int)) {
	for _, n := range sizes {
		b.Run(valName(n), func(b *testing.B) {
			if len(bmbuf) < n {
				bmbuf = make([]byte, n)
			}
			b.SetBytes(int64(n))
			f(b, n)
		})
	}
}

func bmIndexRuneCaseUnicode(rt *unicode.RangeTable, needle rune) func(b *testing.B, n int) {
	var rs []rune
	for _, r16 := range rt.R16 {
		for r := rune(r16.Lo); r <= rune(r16.Hi); r += rune(r16.Stride) {
			if r != needle {
				rs = append(rs, r)
			}
		}
	}
	for _, r32 := range rt.R32 {
		for r := rune(r32.Lo); r <= rune(r32.Hi); r += rune(r32.Stride) {
			if r != needle {
				rs = append(rs, r)
			}
		}
	}
	// Shuffle the runes so that they are not in descending order.
	// The sort is deterministic since this is used for benchmarks,
	// which need to be repeatable.
	rr := rand.New(rand.NewSource(1))
	rr.Shuffle(len(rs), func(i, j int) {
		rs[i], rs[j] = rs[j], rs[i]
	})
	uchars := string(rs)

	return func(b *testing.B, n int) {
		buf := bmbuf[0:n]
		o := copy(buf, uchars)
		for o < len(buf) {
			o += copy(buf[o:], uchars)
		}

		// Make space for the needle rune at the end of buf.
		m := utf8.RuneLen(needle)
		for o := m; o > 0; {
			_, sz := utf8.DecodeLastRune(buf)
			copy(buf[len(buf)-sz:], "\x00\x00\x00\x00")
			buf = buf[:len(buf)-sz]
			o -= sz
		}
		buf = utf8.AppendRune(buf[:n-m], needle)
		s := *(*string)(unsafe.Pointer(&buf))

		n -= m // adjust for rune len
		for i := 0; i < b.N; i++ {
			j := indexRuneCase(s, needle)
			if j != n {
				b.Fatal("bad index", j)
			}
		}
		for i := range buf {
			buf[i] = 0
		}
	}
}

func BenchmarkIndexRuneCaseUnicode(b *testing.B) {
	b.Run("Latin", func(b *testing.B) {
		// Latin is mostly 1, 2, 3 byte runes.
		benchBytes(b, indexSizes, bmIndexRuneCaseUnicode(unicode.Latin, 'é'))
	})
	b.Run("Cyrillic", func(b *testing.B) {
		// Cyrillic is mostly 2 and 3 byte runes.
		benchBytes(b, indexSizes, bmIndexRuneCaseUnicode(unicode.Cyrillic, 'Ꙁ'))
	})
	b.Run("Han", func(b *testing.B) {
		// Han consists only of 3 and 4 byte runes.
		benchBytes(b, indexSizes, bmIndexRuneCaseUnicode(unicode.Han, '𠀿'))
	})
}

// Torture test IndexRune. This is useful for calculating the cutover
// for when we should switch to strings.Index in indexRuneCase.
func BenchmarkIndexRuneTorture_Bytes(b *testing.B) {
	b.Log("WARN: this only tests runes that are 4 bytes!")
	if *benchStdLib {
		benchBytesUnicode(b, indexSizes, bmIndexRune(strings.IndexRune))
	} else {
		benchBytesUnicode(b, indexSizes, bmIndexRune(IndexRune))
	}
}

func BenchmarkIndexByte(b *testing.B) {
	const ch = 'V'
	if got := IndexByte(benchmarkString, ch); got != 17 {
		b.Fatalf("wrong index: expected 17, got=%d", got)
	}
	b.SetBytes(int64(len(benchmarkString)))
	for i := 0; i < b.N; i++ {
		IndexByte(benchmarkString, ch)
	}
}

func BenchmarkIndexByteEmpty(b *testing.B) {
	const ch = 'V'
	for i := 0; i < b.N; i++ {
		IndexByte("", ch)
	}
}

// Benchmark the handling of [KkSs] which require a check for their
// equivalent Unicode folds.
func BenchmarkIndexByteLongSpecial(b *testing.B) {
	for i := range bmbuf {
		bmbuf[i] = 0
	}

	bmIndexByte := func(index func(string, byte) int) func(b *testing.B, n int) {
		return func(b *testing.B, n int) {
			buf := bmbuf[0:n]
			buf[n/2] = 's'
			copy(buf[n-2:], "ſ")
			s := string(buf)
			// We scan the first half of the string twice but the match occurs
			// in the first half so using that index here seems more fair than
			// using the full length of the string as number of bytes processed.
			b.SetBytes(int64(index(s, 's')))
			for i := 0; i < b.N; i++ {
				j := index(s, 's')
				if j != n/2 {
					b.Fatal("bad index", j)
				}
			}
			buf[n/2] = '\x00'
			buf[n-2] = '\x00'
			buf[n-1] = '\x00'
		}
	}

	benchBytes := func(b *testing.B, sizes []int, f func(b *testing.B, n int)) {
		for _, n := range sizes {
			b.Run(valName(n), func(b *testing.B) {
				if len(bmbuf) < n {
					bmbuf = make([]byte, n)
				}
				f(b, n)
			})
		}
	}

	benchBytes(b, indexSizes, bmIndexByte(IndexByte))
}

func BenchmarkLastIndexByte(b *testing.B) {
	if testing.Short() {
		b.Skip("short test")
	}
	s := "b" + strings.Repeat("a", 128)
	c := byte('B')
	if i := LastIndexByte(s, c); i != 0 {
		b.Fatal("invalid index:", i)
	}
	b.SetBytes(int64(len(s)))
	for i := 0; i < b.N; i++ {
		LastIndexByte(s, c)
	}
}

// WARN
var benchStdLib = flag.Bool("stdlib", false, "Use strings.Index in benchmarks (for comparison)")

// WARN: this is not really fair because of strings.ToLower
func benchmarkIndex(b *testing.B, s, substr string) {
	if *benchStdLib {
		n := strings.Index(strings.ToLower(s), strings.ToLower(substr))
		if o := Index(s, substr); n != o {
			b.Errorf("strings.Index(%q, %q) = %d; want: %d", s, substr, n, o)
		}
		if n >= 0 {
			b.SetBytes(int64(len(s) + len(substr)))
		} else {
			b.SetBytes(int64(len(s)))
		}
		for i := 0; i < b.N; i++ {
			strings.Index(strings.ToLower(s), strings.ToLower(substr))
		}
	} else {
		if n := Index(s, substr); n >= 0 {
			b.SetBytes(int64(len(s) + len(substr)))
		} else {
			b.SetBytes(int64(len(s)))
		}
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
	if got := LastIndex(benchmarkString, "v"); got != 17 {
		b.Fatalf("wrong index: expected 17, got=%d", got)
	}
	for i := 0; i < b.N; i++ {
		LastIndex(benchmarkString, "v")
	}
}

// Thanks to variable length encoding it's possible the needle
// to be larger than the haystack.
func BenchmarkLastIndexNeedleExceedsHaystack(b *testing.B) {
	s := strings.Repeat("ab", 1024)
	substr := "z" + s
	i1 := strings.LastIndex(s, substr)
	i2 := LastIndex(s, substr)
	if i1 != i2 {
		b.Fatalf("wrong index: expected: %d, got: %d", i1, i2)
	}
	// Can't compare perf to the stdlib because we have to scan
	// the whole string and not just bail at the length mismatch.
	b.SetBytes(int64(len(s)))
	for i := 0; i < b.N; i++ {
		LastIndex(s, substr)
	}
}

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

const russianText = `Владимир Маяковский родился в селе Багдади[10] Кутаисской
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

var (
	russianUpper = strings.ToUpper(russianText)
	russianLower = strings.ToLower(russianText)
)

func BenchmarkIndexRussian(b *testing.B) {
	benchmarkIndex(b, russianText, "младенчестве")
}

// Pathological worst-case.
func BenchmarkIndexLateMatchLargeNeedle(b *testing.B) {
	bench := func(b *testing.B, s1, s2, s3 string) {
		m := strings.Repeat(s1, 100/len(s1))
		haystack := strings.Repeat(m+s2, 300) + m + s3
		needle := m + s3
		benchmarkIndex(b, haystack, needle)
	}
	b.Run("Latin", func(b *testing.B) {
		bench(b, "AB", "C", "D")
	})
	b.Run("Cyrillic", func(b *testing.B) {
		bench(b, "А̀ВЄ", "Ж", "Њ")
	})
	b.Run("Han", func(b *testing.B) {
		bench(b, "遠方", "來", "矣")
	})
}

// Pathological worst-case. Consistency here is a good thing.
func BenchmarkIndexLateMatchSmallNeedle(b *testing.B) {
	bench := func(b *testing.B, s1, s2 string) {
		s := strings.Repeat(s1, 1_000/len(s1)) + s2
		rs := []rune(s)
		for i := 2; i <= 64; i *= 2 {
			b.Run(strconv.Itoa(i), func(b *testing.B) {
				benchmarkIndex(b, s, string(rs[len(rs)-i:]))
			})
		}
	}
	b.Run("Numeric", func(b *testing.B) {
		bench(b, "123", "4")
	})
	b.Run("Latin", func(b *testing.B) {
		bench(b, "abc", "d")
	})
	b.Run("Cyrillic", func(b *testing.B) {
		bench(b, "А̀ВЄ", "Њ")
	})
	b.Run("Han", func(b *testing.B) {
		bench(b, "遠方", "來")
	})
}

// Pathological worst-case. Consistency here is a good thing.
func BenchmarkIndexEarlyMatchSmallNeedle(b *testing.B) {
	bench := func(b *testing.B, s1, s2 string) {
		for i := 2; i <= 32; i += 2 {
			s := strings.Repeat(s1, i) + s2
			substr := s1 + s2
			b.Run(strconv.Itoa(i), func(b *testing.B) {
				benchmarkIndex(b, s, substr)
			})
		}
	}
	b.Run("Latin", func(b *testing.B) {
		bench(b, "AB", "C")
	})
	b.Run("Cyrillic", func(b *testing.B) {
		bench(b, "А̀В", "Њ")
	})
	b.Run("Han", func(b *testing.B) {
		bench(b, "遠方", "來")
	})
}

// Thanks to variable length encoding it's possible the needle
// to be larger than the haystack.
func BenchmarkIndexNeedleExceedsHaystack(b *testing.B) {
	s := strings.Repeat("А̀В", 32*1024)
	substr := s + s[:len(s)/2] + "z"
	i1 := strings.Index(s, substr)
	i2 := Index(s, substr)
	if i1 != i2 {
		b.Fatalf("wrong index: expected: %d, got: %d", i1, i2)
	}
	// Can't compare perf to the stdlib because we have to scan
	// the whole string and not just bail at the length mismatch.
	b.SetBytes(int64(len(s)))
	for i := 0; i < b.N; i++ {
		Index(s, substr)
	}
}

// Pathological worst-case. Consistency here is a good thing.
func BenchmarkLastIndexLateMatchSmallNeedle(b *testing.B) {
	bench := func(b *testing.B, s1, s2 string) {
		s := s2 + strings.Repeat(s1, 1_000/len(s1))
		rs := []rune(s)
		for _, i := range []int{2, 16, 32} {
			b.Run(strconv.Itoa(i), func(b *testing.B) {
				b.SetBytes(int64(len(s)))
				substr := string(rs[:i])
				for i := 0; i < b.N; i++ {
					if j := LastIndex(s, substr); j != 0 {
						b.Fatalf("LastIndex(%q, %q) = %d; want: %d", s, substr, j, 0)
					}
				}
			})
		}
	}
	b.Run("Cyrillic", func(b *testing.B) {
		bench(b, "А̀ВЄ", "Њ")
	})
	b.Run("Han", func(b *testing.B) {
		bench(b, "遠方", "來")
	})
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
	i := LastIndex(benchInputHard, sep)
	if i < 0 {
		b.SetBytes(int64(len(benchInputHard)))
	} else {
		b.SetBytes(int64(i + len(sep)))
	}
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

// TODO: these benchmarks are not very useful
func BenchmarkLastIndexHard1(b *testing.B) { benchmarkLastIndexHard(b, "<>") }
func BenchmarkLastIndexHard2(b *testing.B) { benchmarkLastIndexHard(b, "</pre>") }
func BenchmarkLastIndexHard3(b *testing.B) { benchmarkLastIndexHard(b, "<b>hello world</b>") }

func BenchmarkLastIndexRuneUnicode(b *testing.B) {
	bench := func(b *testing.B, name string, rt *unicode.RangeTable) {
		b.Run(name, func(b *testing.B) {
			var rs []rune
			visitTable(rt, func(r rune) {
				if len(rs) < 1024 {
					rs = append(rs, r)
				}
			})
			s := string(rs)
			r := rs[0]
			b.SetBytes(int64(len(s)))
			for i := 0; i < b.N; i++ {
				lastIndexRune(s, r)
			}
		})
	}
	bench(b, "Han", unicode.Han)           // no folds
	bench(b, "Cyrillic", unicode.Cyrillic) // folds
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
			benchmarkIndex(b, s, key)
		})
	}
}

func BenchmarkIndexNonASCII(b *testing.B) {
	for _, size := range indexSizes {
		b.Run(valName(size), func(b *testing.B) {
			s := strings.Repeat("a", size-1) + string(rune(utf8.RuneSelf))
			if i := IndexNonASCII(s); i < 0 {
				b.Fatalf("IndexNonASCII(%q) = -1", s)
				return
			}
			b.SetBytes(int64(len(s)))
			for i := 0; i < b.N; i++ {
				IndexNonASCII(s)
			}
		})
	}
}

func BenchmarkHasPrefixASCII(b *testing.B) {
	s := strings.Repeat("a", 64)
	if !HasPrefix(s, s) {
		b.Fatalf("HasPrefix(%[1]q, %[1]q) = false; want: true", s)
	}
	b.SetBytes(int64(len(s)))
	for i := 0; i < b.N; i++ {
		HasPrefix(s, s)
	}
}

func BenchmarkHasPrefix(b *testing.B) {
	if !HasPrefix(benchmarkString, benchmarkString) {
		b.Fatalf("HasPrefix(%[1]q, %[1]q) = false; want: true", benchmarkString)
	}
	b.SetBytes(int64(len(benchmarkString)))
	for i := 0; i < b.N; i++ {
		HasPrefix(benchmarkString, benchmarkString)
	}
}

func BenchmarkHasPrefixUnicode(b *testing.B) {
	const prefix = "Владимир Маяковский родился"
	b.SetBytes(int64(len(prefix)))
	for i := 0; i < b.N; i++ {
		HasPrefix(prefix, "Владимир МАЯКОВСКИЙ родился")
	}
}

func BenchmarkHasPrefixHard(b *testing.B) {
	if !HasPrefix(benchInputHard, benchInputHard) {
		b.Fatalf("HasPrefix(%[1]q, %[1]q) = false; want: true", benchInputHard)
	}
	b.SetBytes(int64(len(benchInputHard)))
	for i := 0; i < b.N; i++ {
		HasPrefix(benchInputHard, benchInputHard)
	}
}

func BenchmarkHasPrefixRussian(b *testing.B) {
	if !HasPrefix(russianLower, russianUpper) {
		b.Fatalf("HasPrefix(%[1]q, %[1]q) = false; want: true", russianText)
	}
	b.SetBytes(int64(len(russianLower)))
	for i := 0; i < b.N; i++ {
		HasPrefix(russianLower, russianUpper)
	}
}

func BenchmarkHasPrefixLonger(b *testing.B) {
	prefix := strings.Repeat("\u212a", 32)
	s := strings.Repeat("k", 32)
	if !HasPrefix(s, prefix) {
		b.Fatalf("HasPrefix(%q, %q) = false; want: true", s, prefix)
	}

	b.Run("Equal", func(b *testing.B) {
		b.SetBytes(int64(len(prefix)))
		for i := 0; i < b.N; i++ {
			HasPrefix(s, prefix)
		}
	})

	b.Run("ShortCircuitSize", func(b *testing.B) {
		kprefix := prefix + "\u212a"
		b.SetBytes(int64(len(kprefix)))
		for i := 0; i < b.N; i++ {
			HasPrefix(s, kprefix)
		}
	})

	// Benchmark the overhead of checking for Kelvin
	b.Run("KelvinCheck", func(b *testing.B) {
		ks := s + "\u212a"
		b.SetBytes(int64(len(ks)))
		for i := 0; i < b.N; i++ {
			containsKelvin(ks)
		}
	})
}

// TODO: need to compare against the stdlib
func BenchmarkHasSuffix(b *testing.B) {
	if !HasSuffix(benchmarkString, benchmarkString) {
		b.Fatalf("HasSuffix(%[1]q, %[1]q) = false; want: true", benchmarkString)
	}
	for i := 0; i < b.N; i++ {
		HasSuffix(benchmarkString, benchmarkString)
	}
}

// TODO: match the logic of HasPrefix
// TODO: need to compare against the stdlib
func BenchmarkHasSuffixRussian(b *testing.B) {
	if !HasSuffix(russianLower, russianUpper) {
		b.Fatalf("HasSuffix(%[1]q, %[1]q) = false; want: true", russianText)
	}
	b.SetBytes(int64(len(russianLower)))
	for i := 0; i < b.N; i++ {
		HasSuffix(russianLower, russianUpper)
	}
}

func benchmarkIndexAny(b *testing.B, s, chars string) {
	i1 := strings.IndexAny(s, chars)
	i2 := IndexAny(s, chars)
	if i1 != i2 {
		b.Fatalf("strings.IndexAny != IndexAny: %d != %d", i1, i2)
	}
	min := len(s)
	for i, r := range chars {
		o := strings.IndexRune(s, r)
		if 0 <= o && o < min {
			min = i + utf8.RuneLen(r) // Include the length of the matched rune
		}
	}
	bytes := int64(min)
	if *benchStdLib {
		b.SetBytes(bytes)
		for i := 0; i < b.N; i++ {
			strings.IndexAny(s, chars)
		}
	} else {
		b.SetBytes(bytes)
		for i := 0; i < b.N; i++ {
			IndexAny(s, chars)
		}
	}
}

func BenchmarkIndexAnyASCII(b *testing.B) {
	x := strings.Repeat("#", 2048) // Never matches set
	cs := "0123456789abcdefghijklmnopqrstuvwxyz0123456789abcdefghijklmnopqrstuvwxyz"
	for k := 1; k <= 2048; k <<= 4 {
		for j := 1; j <= 64; j <<= 1 {
			b.Run(fmt.Sprintf("%d:%d", k, j), func(b *testing.B) {
				benchmarkIndexAny(b, x[:k], cs[:j])
			})
		}
	}
}

func BenchmarkIndexAnyUTF8(b *testing.B) {
	x := strings.Repeat("#", 2048) // Never matches set
	// TODO: use a more diverse string (diff languages)
	cs := "你好世界, hello world. 你好世界, hello world. 你好世界, hello world."
	for k := 1; k <= 2048; k <<= 4 {
		for j := 1; j <= 64; j <<= 1 {
			b.Run(fmt.Sprintf("%d:%d", k, j), func(b *testing.B) {
				var chars string
				n := j
				for i, r := range cs {
					n--
					if n <= 0 {
						chars = cs[:i+utf8.RuneLen(r)]
						break
					}
				}
				benchmarkIndexAny(b, x[:k], chars)
			})
		}
	}
}

func benchmarkLastIndexAny(b *testing.B, s, chars string) {
	i1 := strings.LastIndexAny(s, chars)
	i2 := LastIndexAny(s, chars)
	if i1 != i2 {
		b.Fatalf("strings.LastIndexAny != LastIndexAny: %d != %d", i1, i2)
	}
	// TODO: make sure the logic here is correct
	i := strings.LastIndexAny(s, chars)
	if i < 0 {
		i = 0
	}
	bytes := int64(len(s) - i)
	if *benchStdLib {
		b.SetBytes(bytes)
		for i := 0; i < b.N; i++ {
			strings.LastIndexAny(s, chars)
		}
	} else {
		b.SetBytes(bytes)
		for i := 0; i < b.N; i++ {
			LastIndexAny(s, chars)
		}
	}
}

func BenchmarkLastIndexAnyASCII(b *testing.B) {
	x := strings.Repeat("#", 2048) // Never matches set
	cs := "0123456789abcdefghijklmnopqrstuvwxyz0123456789abcdefghijklmnopqrstuvwxyz"
	for k := 1; k <= 2048; k <<= 4 {
		for j := 1; j <= 64; j <<= 1 {
			b.Run(fmt.Sprintf("%d:%d", k, j), func(b *testing.B) {
				benchmarkLastIndexAny(b, x[:k], cs[:j])
			})
		}
	}
}

func BenchmarkLastIndexAnyUTF8(b *testing.B) {
	x := strings.Repeat("#", 2048) // Never matches set
	cs := "你好世界, hello world. 你好世界, hello world. 你好世界, hello world."
	for k := 1; k <= 2048; k <<= 4 {
		for j := 1; j <= 64; j <<= 1 {
			b.Run(fmt.Sprintf("%d:%d", k, j), func(b *testing.B) {
				benchmarkLastIndexAny(b, x[:k], cs[:j])
			})
		}
	}
}

func BenchmarkCount(b *testing.B) {
	bench := func(name, s, sep string) {
		b.Run(name, func(b *testing.B) {
			i := strings.Count(s, sep)
			j := Count(s, sep)
			if i != j {
				b.Fatalf("Count(%q, %q) = %d; want: %d", s, sep, j, i)
			}
			b.SetBytes(int64(len(s)))
			for i := 0; i < b.N; i++ {
				Count(s, sep)
			}
		})
	}
	bench("ASCII", strings.Repeat("    ab", 64), "ab")
	bench("Unicode", strings.Repeat("你好世界", 128), "你好世界")
	// Make sure we lazily process substr.
	bench("NoMatch", strings.Repeat("你", 8), strings.Repeat("好", 256))
}

// Micro-benchmarks for caseFold

var caseFoldBenchmarkRunes = [16]rune{
	0xA7C9,
	0xA696,
	0x03A7,
	0x021E,
	0x03A3,
	0x01B5,
	0x01A6,
	0xABBC,
	0xA72C,
	0x1F8E,
	0x0056,
	0x016E,
	0x1E86,
	0x1C92,
	0x0555,
	0x0544,
}

var caseFoldBenchmarkAll []rune

func loadCaseFoldBenchmarkAll() {
	if caseFoldBenchmarkAll != nil {
		return
	}
	a := make([]rune, 0, len(foldableRunes))
	for _, r := range foldableRunes {
		if tables.CaseFold(r) != r {
			a = append(a, r)
		}
	}
	// Make sure the slice is consistently sorted before
	// randomizing order. This is relevant because the
	// order of slice elements may change.
	if !sort.IsSorted(byRune(a)) {
		sort.Sort(byRune(a))
	}
	rr := rand.New(rand.NewSource(12345))
	rr.Shuffle(len(a), func(i, j int) {
		a[i], a[j] = a[j], a[i]
	})
	caseFoldBenchmarkAll = a
}

func BenchmarkCaseFold(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = tables.CaseFold(caseFoldBenchmarkRunes[i%len(caseFoldBenchmarkRunes)])
	}
}

func BenchmarkCaseFoldAll(b *testing.B) {
	loadCaseFoldBenchmarkAll()
	for i := 0; i < b.N; i++ {
		for j := i; j < len(caseFoldBenchmarkAll) && j < b.N; j++ {
			_ = tables.CaseFold(caseFoldBenchmarkAll[j])
		}
	}
}

// Micro-benchmarks for toUpperLower

var toUpperLowerBenchmarkRunes = [16]rune{
	0xA68A,
	0x0204,
	0x04EC,
	0x00D0,
	0x0053,
	0xA698,
	0x1F1A,
	0x038E,
	0x1F1B,
	0x2126,
	0x16E47,
	0x01D1,
	0x13CC,
	0x01BC,
	0x048E,
	0x0386,
}

func BenchmarkToUpperLower(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _, _ = tables.ToUpperLower(toUpperLowerBenchmarkRunes[i%len(toUpperLowerBenchmarkRunes)])
	}
}

func BenchmarkNonLetterASCII(b *testing.B) {
	base := "!\"#$%&'()*+,-./0123456789:;<=>?@[\\]^_`{|}~"
	base += base + base + base
	for _, size := range []int{4, 8, 16, 24, 32, 64, 128} {
		b.Run(fmt.Sprint(size), func(b *testing.B) {
			s := base[:size]
			b.SetBytes(int64(len(s)))
			for i := 0; i < b.N; i++ {
				nonLetterASCII(s)
			}
		})
	}
}
