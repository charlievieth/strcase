// Copyright 2023 Charlie Vieth. All rights reserved.
// Use of this source code is governed by the MIT license.

package strcase

import (
	"flag"
	"fmt"
	"runtime"
	"sort"
	"strings"
	"testing"
)

var calibrate = flag.Bool("calibrate", false, "compute crossover for linear vs. binary search")

// TODO: use this
//
// TestCalibrate determines the cutoff where a brute-force-search is faster
// than the current Index algorithm.
func TestCalibrateBruteForce(t *testing.T) {
	if !*calibrate {
		return
	}

	// TODO: run this on amd64
	if runtime.GOARCH == "amd64" {
		fmt.Printf("warning: running calibration on %s\n", runtime.GOARCH)
	}

	bench := func(t *testing.T, name, prefix string) {
		t.Run(name, func(t *testing.T) {
			n := sort.Search(64, func(n int) bool {
				key := prefix + "a"
				s := strings.Repeat(prefix+strings.Repeat(" ", n-1), 1<<16/n)
				bruteForce := func(b *testing.B) {
					for i := 0; i < b.N; i++ {
						bruteForceIndexUnicode(s, key)
					}
				}
				shortIndex := func(b *testing.B) {
					for i := 0; i < b.N; i++ {
						shortIndexUnicode(s, key)
					}
				}
				bmbrute := testing.Benchmark(bruteForce)
				bmshort := testing.Benchmark(shortIndex)
				fmt.Printf("n=%d: brute=%d index=%d\n", n, bmbrute.NsPerOp(), bmshort.NsPerOp())
				return bmbrute.NsPerOp()*100 > bmshort.NsPerOp()*110
			})
			fmt.Printf("calibration: brute-force cutoff = %d\n", n)
		})
	}

	bench(t, "Unicode", "α")
	bench(t, "ASCII", "a")
}

// This test checks if the performance of strings.LastIndexByte has improved
// and beats the naive implementation (based on Go 1.19) that is used here.
func TestCalibrateLastIndexByte(t *testing.T) {
	if !*calibrate {
		return
	}

	s := "1" + strings.Repeat("a", 256)
	c := byte('1')

	benchStringsLastIndexByte := func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			strings.LastIndexByte(s, c)
		}
	}
	benchLastIndexByte := func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			LastIndexByte(s, c)
		}
	}
	nsStrings := testing.Benchmark(benchStringsLastIndexByte).NsPerOp()
	nsStrcase := testing.Benchmark(benchLastIndexByte).NsPerOp()

	t.Logf("strings=%d stracse=%d\n", nsStrings, nsStrcase)

	// If strings.LastIndexByte is 1.5x faster than the current naive
	// Go 1.19 implementation then we should take advantage of that.
	if nsStrings > nsStrcase+nsStrcase/2 {
		t.Fatalf("strings.LastIndexByte = %d ns/op LastIndexByte = %d ns/op Delta = %.2fx",
			nsStrings, nsStrcase, float64(nsStrcase)/float64(nsStrings))
	}
}
