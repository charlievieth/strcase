package bytealg

import (
	"bytes"
	"fmt"
	"testing"
)

func testIndexByte(t *testing.T, name string, fn func([]byte, byte) int) {
	errCount := 0
	for _, size := range []int{1, 7, 6, 15, 16, 17, 56, 66, 129, 256, 1024, 1027} {
		t.Run(fmt.Sprintf("%d", size), func(t *testing.T) {
			s := bytes.Repeat([]byte{'a'}, size)
			for i := 0; i < size; i++ {
				for _, c := range []byte{'x', 'X'} {
					if i < len(s)-1 {
						s[i] = c | ' ' // swap case
						s[i+1] = c
					} else {
						s[i] = c
					}
					if j := fn(s, c); j != i {
						t.Errorf("%s(%q, %c) = %d; want: %d", name, s, c, j, i)
					}
					for i := 0; i < len(s); i++ {
						s[i] = 'a'
					}
					if t.Failed() {
						errCount++
						if errCount >= 20 {
							panic(fmt.Sprintf("Too many errors: %d", errCount))
						}
					}
				}
			}
		})
	}
}

func TestIndexByte(t *testing.T) {
	testIndexByte(t, "IndexByte", IndexByte)
}

func TestIndexByteString(t *testing.T) {
	testIndexByte(t, "IndexByteString", func(s []byte, c byte) int {
		return IndexByteString(string(s), c)
	})
}
