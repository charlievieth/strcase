package bytealg

import (
	"bytes"
	"fmt"
	"testing"
)

func testIndexByte(t *testing.T, base, name string, fn func([]byte, byte) int) {
	const maxErrors = 40
	for _, size := range []int{1, 7, 6, 15, 16, 17, 56, 66, 129, 256, 1024, 1027} {
		t.Run(fmt.Sprintf("%d", size), func(t *testing.T) {
			errCount := 0
			s := bytes.Repeat([]byte{'a'}, size)
			for i := 0; i < size; i++ {
				for j, c := range []byte{'x', 'X'} {
					if i < len(s)-1 {
						s[i] = c ^ ' ' // swap case
						s[i+1] = c
					} else {
						s[i] = c
					}
					if o := fn(s, c); o != i {
						// t.Errorf("%s(%q, %c) = %d; want: %d", name, s, c, o, i)
						t.Errorf("%d.%d got: %d; want: %d", i, j, o, i)
						errCount++
					}
					for i := 0; i < len(s); i++ {
						s[i] = 'a'
					}
					if errCount >= maxErrors {
						t.FailNow()
					}
				}
			}
		})
	}
}

const alpha = "abcdefghijklmnopqrstuvwyz" // no X

func TestIndexByte(t *testing.T) {
	testIndexByte(t, alpha, "IndexByte", IndexByte)
	testIndexByte(t, "a", "IndexByte", IndexByte)
}

func TestIndexByteString(t *testing.T) {
	fn := func(s []byte, c byte) int {
		return IndexByteString(string(s), c)
	}
	testIndexByte(t, alpha, "IndexByteString", fn)
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
	benchBytes(b, indexSizes, bmIndexByte(IndexByte))
}

func BenchmarkIndexBytePortable(b *testing.B) {
	benchBytes(b, indexSizes, bmIndexByte(indexBytePortable))
}

func bmIndexByte(index func([]byte, byte) int) func(b *testing.B, n int) {
	return func(b *testing.B, n int) {
		buf := bmbuf[0:n]
		buf[n-1] = 'x'
		for i := 0; i < b.N; i++ {
			j := index(buf, 'x')
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
