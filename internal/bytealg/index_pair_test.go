// Copyright 2026 Charlie Vieth. All rights reserved.
// Use of this source code is governed by the MIT license.

package bytealg

import (
	"math/rand"
	"strings"
	"testing"
)

// indexPairFoldRef is a reference implementation of IndexPairFold.
func indexPairFoldRef(s string, pair uint32) int {
	c0l := byte(pair)
	c0u := byte(pair >> 8)
	c1l := byte(pair >> 16)
	c1u := byte(pair >> 24)
	for i := 0; i+1 < len(s); i++ {
		if c := s[i]; c == c0l || c == c0u {
			if c := s[i+1]; c == c1l || c == c1u {
				return i
			}
		}
	}
	return -1
}

func TestIndexPairFold(t *testing.T) {
	pairs := [][2]byte{
		{'a', 'b'},
		{'A', 'B'},
		{'k', 'K'},
		{'1', '2'},
		{'a', 'a'},
		{' ', 'a'},
		{0x80, 0xFF},
		{0xC3, 0xA1},
		{'z', 0x00},
		{0x00, 0x00},
	}
	rr := rand.New(rand.NewSource(1234))
	alphabet := "aAbB12 \x00\x80\xFF\xC3\xA1zZ"
	for _, size := range []int{0, 1, 2, 3, 7, 8, 9, 15, 16, 17, 31, 32, 33,
		34, 35, 63, 64, 65, 100, 255, 256, 257, 1024} {
		for trial := 0; trial < 25; trial++ {
			b := make([]byte, size)
			for i := range b {
				b[i] = alphabet[rr.Intn(len(alphabet))]
			}
			s := string(b)
			for _, pc := range pairs {
				pair := PackPair(pc[0], pc[1])
				want := indexPairFoldRef(s, pair)
				got := IndexPairFold(s, pair)
				if got != want {
					t.Fatalf("IndexPairFold(%q, %q%q) = %d; want: %d",
						s, pc[0], pc[1], got, want)
				}
			}
		}
	}
	// Test all positions of a match in an otherwise non-matching string.
	for _, size := range []int{2, 3, 8, 16, 32, 33, 34, 35, 64, 100, 256} {
		for pos := 0; pos+2 <= size; pos++ {
			b := []byte(strings.Repeat("-", size))
			b[pos] = 'A'
			b[pos+1] = 'b'
			s := string(b)
			pair := PackPair('a', 'B')
			if got := IndexPairFold(s, pair); got != pos {
				t.Fatalf("IndexPairFold(%q, aB) = %d; want: %d", s, got, pos)
			}
		}
	}
	// Test unaligned starts (sub-slices).
	base := strings.Repeat("-", 128) + "aB" + strings.Repeat("-", 128)
	pair := PackPair('A', 'b')
	for off := 0; off < 64; off++ {
		s := base[off:]
		if got, want := IndexPairFold(s, pair), 128-off; got != want {
			t.Fatalf("IndexPairFold(base[%d:], Ab) = %d; want: %d", off, got, want)
		}
	}
}

// lastIndexPairFoldRef is a reference implementation of LastIndexPairFold.
func lastIndexPairFoldRef(s string, pair uint32) int {
	c0l := byte(pair)
	c0u := byte(pair >> 8)
	c1l := byte(pair >> 16)
	c1u := byte(pair >> 24)
	for i := len(s) - 2; i >= 0; i-- {
		if c := s[i]; c == c0l || c == c0u {
			if c := s[i+1]; c == c1l || c == c1u {
				return i
			}
		}
	}
	return -1
}

func TestLastIndexPairFold(t *testing.T) {
	pairs := [][2]byte{
		{'a', 'b'}, {'A', 'B'}, {'k', 'K'}, {'1', '2'}, {'a', 'a'},
		{' ', 'a'}, {0x80, 0xFF}, {0xC3, 0xA1}, {'z', 0x00}, {0x00, 0x00},
	}
	rr := rand.New(rand.NewSource(4321))
	alphabet := "aAbB12 \x00\x80\xFF\xC3\xA1zZ"
	for _, size := range []int{0, 1, 2, 3, 7, 8, 9, 15, 16, 17, 31, 32, 33,
		34, 35, 63, 64, 65, 100, 255, 256, 257, 1024} {
		for trial := 0; trial < 25; trial++ {
			b := make([]byte, size)
			for i := range b {
				b[i] = alphabet[rr.Intn(len(alphabet))]
			}
			s := string(b)
			for _, pc := range pairs {
				pair := PackPair(pc[0], pc[1])
				want := lastIndexPairFoldRef(s, pair)
				got := LastIndexPairFold(s, pair)
				if got != want {
					t.Fatalf("LastIndexPairFold(%q, %q%q) = %d; want: %d",
						s, pc[0], pc[1], got, want)
				}
			}
		}
	}
	// Test all positions of a match in an otherwise non-matching string.
	for _, size := range []int{2, 3, 8, 16, 32, 33, 34, 35, 64, 100, 256} {
		for pos := 0; pos+2 <= size; pos++ {
			b := []byte(strings.Repeat("-", size))
			b[pos] = 'A'
			b[pos+1] = 'b'
			s := string(b)
			pair := PackPair('a', 'B')
			if got := LastIndexPairFold(s, pair); got != pos {
				t.Fatalf("LastIndexPairFold(%q, aB) = %d; want: %d", s, got, pos)
			}
		}
	}
	// Test unaligned starts (sub-slices).
	base := strings.Repeat("-", 128) + "aB" + strings.Repeat("-", 128)
	pair := PackPair('A', 'b')
	for off := 0; off < 64; off++ {
		s := base[off:]
		if got, want := LastIndexPairFold(s, pair), 128-off; got != want {
			t.Fatalf("LastIndexPairFold(base[%d:], Ab) = %d; want: %d", off, got, want)
		}
	}
}
