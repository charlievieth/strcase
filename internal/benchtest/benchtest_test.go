package benchtest

import (
	"flag"
	"fmt"
	"math/rand"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/charlievieth/strcase"
)

var benchStdLib = flag.Bool("stdlib", false, "Use the stdlib's strings package instead of strcase (for comparison)")

func benchIndexRune(b *testing.B, s string, r rune) {
	if strings.IndexRune(s, r) != strcase.IndexRune(s, r) {
		b.Fatal("Invalid benchmark: strings/strcase results are not equal")
	}
	if *benchStdLib {
		for i := 0; i < b.N; i++ {
			strings.IndexRune(s, r)
		}
	} else {
		for i := 0; i < b.N; i++ {
			strcase.IndexRune(s, r)
		}
	}
}

func benchIndex(b *testing.B, s, substr string) {
	if strings.Index(s, substr) != strcase.Index(s, substr) {
		b.Fatal("Invalid benchmark: strings/strcase results are not equal")
	}
	if *benchStdLib {
		for i := 0; i < b.N; i++ {
			strings.Index(s, substr)
		}
	} else {
		for i := 0; i < b.N; i++ {
			strcase.Index(s, substr)
		}
	}
}

func benchIndexByte(b *testing.B, s string, c byte) {
	if strings.IndexByte(s, c) != strcase.IndexByte(s, c) {
		b.Fatal("Invalid benchmark: strings/strcase results are not equal")
	}
	if *benchStdLib {
		for i := 0; i < b.N; i++ {
			strings.IndexByte(s, c)
		}
	} else {
		for i := 0; i < b.N; i++ {
			strcase.IndexByte(s, c)
		}
	}
}

func benchLastIndex(b *testing.B, s, substr string) {
	if strings.LastIndex(s, substr) != strcase.LastIndex(s, substr) {
		b.Fatal("Invalid benchmark: strings/strcase results are not equal")
	}
	if *benchStdLib {
		for i := 0; i < b.N; i++ {
			strings.LastIndex(s, substr)
		}
	} else {
		for i := 0; i < b.N; i++ {
			strcase.LastIndex(s, substr)
		}
	}
}

func benchEqualFold(b *testing.B, s1, s2 string) {
	if strings.EqualFold(s1, s2) != (strcase.Compare(s1, s2) == 0) {
		b.Fatal("Invalid benchmark: strings/strcase results are not equal")
	}
	if *benchStdLib {
		for i := 0; i < b.N; i++ {
			strings.EqualFold(s1, s2)
		}
	} else {
		for i := 0; i < b.N; i++ {
			strcase.Compare(s1, s2)
		}
	}
}

func benchCount(b *testing.B, s, substr string) {
	if strings.Count(s, substr) != strcase.Count(s, substr) {
		b.Fatal("Invalid benchmark: strings/strcase results are not equal")
	}
	if *benchStdLib {
		for i := 0; i < b.N; i++ {
			strings.Count(s, substr)
		}
	} else {
		for i := 0; i < b.N; i++ {
			strcase.Count(s, substr)
		}
	}
}

func benchIndexAny(b *testing.B, s, cutset string) {
	if strings.IndexAny(s, cutset) != strcase.IndexAny(s, cutset) {
		b.Fatal("Invalid benchmark: strings/strcase results are not equal")
	}
	if *benchStdLib {
		for i := 0; i < b.N; i++ {
			strings.IndexAny(s, cutset)
		}
	} else {
		for i := 0; i < b.N; i++ {
			strcase.IndexAny(s, cutset)
		}
	}
}

func benchLastIndexAny(b *testing.B, s, cutset string) {
	if strings.LastIndexAny(s, cutset) != strcase.LastIndexAny(s, cutset) {
		b.Fatal("Invalid benchmark: strings/strcase results are not equal")
	}
	if *benchStdLib {
		for i := 0; i < b.N; i++ {
			strings.LastIndexAny(s, cutset)
		}
	} else {
		for i := 0; i < b.N; i++ {
			strcase.LastIndexAny(s, cutset)
		}
	}
}

// The below benchmarks are from src/strings/strings_test.go

const benchmarkString = "some_text=some☺value"

func BenchmarkIndexRune(b *testing.B) {
	if got := strings.IndexRune(benchmarkString, '☺'); got != 14 {
		b.Fatalf("wrong index: expected 14, got=%d", got)
	}
	benchIndexRune(b, benchmarkString, '☺')
}

var benchmarkLongString = strings.Repeat(" ", 100) + benchmarkString

func BenchmarkIndexRuneLongString(b *testing.B) {
	if got := strings.IndexRune(benchmarkLongString, '☺'); got != 114 {
		b.Fatalf("wrong index: expected 114, got=%d", got)
	}
	benchIndexRune(b, benchmarkLongString, '☺')
}

func BenchmarkIndexRuneFastPath(b *testing.B) {
	if got := strings.IndexRune(benchmarkString, 'v'); got != 17 {
		b.Fatalf("wrong index: expected 17, got=%d", got)
	}
	benchIndexRune(b, benchmarkString, 'v')
}

func BenchmarkIndex(b *testing.B) {
	if got := strings.Index(benchmarkString, "v"); got != 17 {
		b.Fatalf("wrong index: expected 17, got=%d", got)
	}
	benchIndex(b, benchmarkString, "v")
}

func BenchmarkLastIndex(b *testing.B) {
	if got := strings.Index(benchmarkString, "v"); got != 17 {
		b.Fatalf("wrong index: expected 17, got=%d", got)
	}
	benchLastIndex(b, benchmarkString, "v")
}

func BenchmarkIndexByte(b *testing.B) {
	if got := strings.IndexByte(benchmarkString, 'v'); got != 17 {
		b.Fatalf("wrong index: expected 17, got=%d", got)
	}
	benchIndexByte(b, benchmarkString, 'v')
}

func BenchmarkEqualFold(b *testing.B) {
	const s1 = "abcdefghijKz"
	const s2 = "abcDefGhijKz"

	b.Run("ASCII", func(b *testing.B) {
		benchEqualFold(b, s1, s2)
	})

	b.Run("UnicodePrefix", func(b *testing.B) {
		benchEqualFold(b, "αβδ"+s1, "ΑΒΔ"+s2)
	})

	b.Run("UnicodeSuffix", func(b *testing.B) {
		benchEqualFold(b, s1+"αβδ", s2+"ΑΒΔ")
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
	benchIndex(b, benchInputHard, sep)
}

func benchmarkLastIndexHard(b *testing.B, sep string) {
	benchLastIndex(b, benchInputHard, sep)
}

func benchmarkCountHard(b *testing.B, sep string) {
	benchCount(b, benchInputHard, sep)
}

func BenchmarkIndexHard1(b *testing.B) { benchmarkIndexHard(b, "<>") }
func BenchmarkIndexHard2(b *testing.B) { benchmarkIndexHard(b, "</pre>") }
func BenchmarkIndexHard3(b *testing.B) { benchmarkIndexHard(b, "<b>hello world</b>") }
func BenchmarkIndexHard4(b *testing.B) {
	benchmarkIndexHard(b, "<pre><b>hello</b><strong>world</strong></pre>")
}

func BenchmarkLastIndexHard1(b *testing.B) { benchmarkLastIndexHard(b, "<>") }
func BenchmarkLastIndexHard2(b *testing.B) { benchmarkLastIndexHard(b, "</pre>") }
func BenchmarkLastIndexHard3(b *testing.B) { benchmarkLastIndexHard(b, "<b>hello world</b>") }

func BenchmarkCountHard1(b *testing.B) { benchmarkCountHard(b, "<>") }
func BenchmarkCountHard2(b *testing.B) { benchmarkCountHard(b, "</pre>") }
func BenchmarkCountHard3(b *testing.B) { benchmarkCountHard(b, "<b>hello world</b>") }

var benchInputTorture = strings.Repeat("ABC", 1<<10) + "123" + strings.Repeat("ABC", 1<<10)
var benchNeedleTorture = strings.Repeat("ABC", 1<<10+1)

func BenchmarkIndexTorture(b *testing.B) {
	benchIndex(b, benchInputTorture, benchNeedleTorture)
}

func BenchmarkCountTorture(b *testing.B) {
	benchCount(b, benchInputTorture, benchNeedleTorture)
}

func BenchmarkCountTortureOverlapping(b *testing.B) {
	A := strings.Repeat("ABC", 1<<20)
	B := strings.Repeat("ABC", 1<<10)
	benchCount(b, A, B)
}

func BenchmarkCountByte(b *testing.B) {
	indexSizes := []int{10, 32, 4 << 10, 4 << 20, 64 << 20}
	benchStr := strings.Repeat(benchmarkString,
		(indexSizes[len(indexSizes)-1]+len(benchmarkString)-1)/len(benchmarkString))
	benchFunc := func(b *testing.B, benchStr string) {
		b.SetBytes(int64(len(benchStr)))
		benchCount(b, benchStr, "=")
	}
	for _, size := range indexSizes {
		b.Run(fmt.Sprintf("%d", size), func(b *testing.B) {
			benchFunc(b, benchStr[:size])
		})
	}

}

func BenchmarkIndexAnyASCII(b *testing.B) {
	x := strings.Repeat("#", 2048) // Never matches set
	cs := "0123456789abcdefghijklmnopqrstuvwxyz0123456789abcdefghijklmnopqrstuvwxyz"
	for k := 1; k <= 2048; k <<= 4 {
		for j := 1; j <= 64; j <<= 1 {
			b.Run(fmt.Sprintf("%d:%d", k, j), func(b *testing.B) {
				benchIndexAny(b, x[:k], cs[:j])
			})
		}
	}
}

func BenchmarkIndexAnyUTF8(b *testing.B) {
	x := strings.Repeat("#", 2048) // Never matches set
	cs := "你好世界, hello world. 你好世界, hello world. 你好世界, hello world."
	for k := 1; k <= 2048; k <<= 4 {
		for j := 1; j <= 64; j <<= 1 {
			b.Run(fmt.Sprintf("%d:%d", k, j), func(b *testing.B) {
				benchIndexAny(b, x[:k], cs[:j])
			})
		}
	}
}

func BenchmarkLastIndexAnyASCII(b *testing.B) {
	x := strings.Repeat("#", 2048) // Never matches set
	cs := "0123456789abcdefghijklmnopqrstuvwxyz0123456789abcdefghijklmnopqrstuvwxyz"
	for k := 1; k <= 2048; k <<= 4 {
		for j := 1; j <= 64; j <<= 1 {
			b.Run(fmt.Sprintf("%d:%d", k, j), func(b *testing.B) {
				benchLastIndexAny(b, x[:k], cs[:j])
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
				benchLastIndexAny(b, x[:k], cs[:j])
			})
		}
	}
}

func BenchmarkIndexPeriodic(b *testing.B) {
	key := "aa"
	for _, skip := range [...]int{2, 4, 8, 16, 32, 64} {
		b.Run(fmt.Sprintf("IndexPeriodic%d", skip), func(b *testing.B) {
			s := strings.Repeat("a"+strings.Repeat(" ", skip-1), 1<<16/skip)
			benchIndex(b, s, key)
		})
	}
}

// The below benchmarks are from src/bytes/bytes_test.go

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

var indexSizes = []int{10, 32, 4 << 10, 4 << 20, 64 << 20}

func BenchmarkIndexByte_Bytes(b *testing.B) {
	if *benchStdLib {
		benchBytes(b, indexSizes, bmIndexByte(strings.IndexByte))
	} else {
		benchBytes(b, indexSizes, bmIndexByte(strcase.IndexByte))
	}
}

func bmIndexByte(index func(string, byte) int) func(b *testing.B, n int) {
	return func(b *testing.B, n int) {
		buf := bmbuf[0:n]
		buf[n-1] = 'x'
		s := string(buf)
		for i := 0; i < b.N; i++ {
			j := index(s, 'x')
			if j != n-1 {
				b.Fatal("bad index", j)
			}
		}
		buf[n-1] = '\x00'
	}
}

func BenchmarkIndexRune_Bytes(b *testing.B) {
	if *benchStdLib {
		benchBytes(b, indexSizes, bmIndexRune(strings.IndexRune))
	} else {
		benchBytes(b, indexSizes, bmIndexRune(strcase.IndexRune))
	}
}

func BenchmarkIndexRuneASCII_Bytes(b *testing.B) {
	if *benchStdLib {
		benchBytes(b, indexSizes, bmIndexRuneASCII(strings.IndexRune))
	} else {
		benchBytes(b, indexSizes, bmIndexRuneASCII(strcase.IndexRune))
	}
}

func bmIndexRuneASCII(index func(string, rune) int) func(b *testing.B, n int) {
	return func(b *testing.B, n int) {
		buf := bmbuf[0:n]
		buf[n-1] = 'x'
		s := string(buf)
		for i := 0; i < b.N; i++ {
			j := index(s, 'x')
			if j != n-1 {
				b.Fatal("bad index", j)
			}
		}
		buf[n-1] = '\x00'
	}
}

func bmIndexRune(index func(string, rune) int) func(b *testing.B, n int) {
	return func(b *testing.B, n int) {
		buf := bmbuf[0:n]
		utf8.EncodeRune(buf[n-3:], '世')
		s := string(buf)
		for i := 0; i < b.N; i++ {
			j := index(s, '世')
			if j != n-3 {
				b.Fatal("bad index", j)
			}
		}
		buf[n-3] = '\x00'
		buf[n-2] = '\x00'
		buf[n-1] = '\x00'
	}
}
