package bytealg

import (
	"bytes"
	"flag"
	"fmt"
	"strings"
	"testing"
)

var quiet = flag.Bool("quiet", false, "quiet test output")

func testIndexByte(t *testing.T, base, name string, fn func([]byte, byte) int) {
	const maxErrors = 40
	if t.Failed() {
		t.FailNow()
		return
	}
	if strings.ContainsAny(base, "xX") {
		t.Fatalf("base string %q may not contain %q", base, "xX")
	}
	var results []string
	// TODO: test near page boundaries
	for _, size := range []int{1, 7, 6, 15, 16, 17, 56, 66, 129, 256, 1024, 1027} {
		t.Run(fmt.Sprintf("%d", size), func(t *testing.T) {
			errCount := 0
			orig := strings.Repeat(base, size)
			s1 := make([]byte, len(orig))
			s2 := make([]byte, 0, len(orig))
			copy(s1, orig)
			for i := 0; i < size; i++ {
				for j, c := range []byte{'x', 'X'} {
					if i < len(s1)-1 {
						s1[i] = c ^ ' ' // swap case
						s1[i+1] = c
					} else {
						s1[i] = c
					}
					s2 = append(s2[:0], s1...) // Make sure we don't modify the haystack
					if o := fn(s1, c); o != i {
						if errCount < maxErrors {
							if !*quiet {
								t.Errorf("%d.%d got: %d; want: %d", i, j, o, i)
							} else {
								t.Fail()
							}
						}
						errCount++
					}
					if !bytes.Equal(s1, s2) {
						t.Fatalf("haystack was modified\ns1: %q\ns2: %q", s1, s2)
					}
					copy(s1, orig)
					if errCount >= maxErrors {
						// t.FailNow()
					}
				}
			}
			if errCount > 0 {
				results = append(results, fmt.Sprintf("%d: failed %d/%d", size, errCount, size*2))
			}
		})
	}
	if t.Failed() {
		t.Logf("%s Summary:\n%s", name, strings.Join(results, "\n"))
	}
}

const alpha = "abcdefghijklmnopqrstuvwyz" // no X

func TestIndexByte(t *testing.T) {
	testIndexByte(t, alpha, "IndexByte", IndexByte)
	testIndexByte(t, strings.ToUpper(alpha), "IndexByte", IndexByte)
	testIndexByte(t, "a", "IndexByte", IndexByte)
}

func TestIndexByteString(t *testing.T) {
	fn := func(s []byte, c byte) int {
		return IndexByteString(string(s), c)
	}
	testIndexByte(t, alpha, "IndexByteString", fn)
	testIndexByte(t, strings.ToUpper(alpha), "IndexByte", IndexByte)
	testIndexByte(t, "a", "IndexByteString", fn)
}

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

func BenchmarkIndexByte(b *testing.B) {
	benchBytes(b, indexSizes, bmIndexByte(IndexByte, true))
}

func BenchmarkIndexBytePortable(b *testing.B) {
	benchBytes(b, indexSizes, bmIndexByte(indexBytePortable, true))
}

func BenchmarkIndexByteStdLib(b *testing.B) {
	benchBytes(b, indexSizes, bmIndexByte(bytes.IndexByte, false))
}

func bmIndexByte(index func([]byte, byte) int, caseless bool) func(b *testing.B, n int) {
	return func(b *testing.B, n int) {
		buf := bmbuf[0:n]
		buf[n-1] = 'x'
		ch := byte('x')
		if caseless {
			ch = 'X'
		}
		for i := 0; i < b.N; i++ {
			j := index(buf, ch) // Search for uppercase variant
			if j != n-1 {
				b.Fatal("bad index", j)
			}
		}
		buf[n-1] = '\x00'
	}
}

func isAlphaPortable(c byte) bool {
	return 'A' <= c && c <= 'Z' || 'a' <= c && c <= 'z'
}

func indexBytePortable(s []byte, c byte) int {
	n := bytes.IndexByte(s, c)
	if n == 0 || !isAlphaPortable(c) {
		return n
	}

	// TODO: calculate the optimal cutoff
	if n > 0 && len(s) >= 16 {
		s = s[:n] // limit search space
	}

	c ^= ' ' // swap case
	if o := bytes.IndexByte(s, c); n == -1 || (o != -1 && o < n) {
		n = o
	}
	return n
}
