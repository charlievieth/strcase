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

func TestCalibrate(t *testing.T) {
	if !*calibrate {
		return
	}

	if runtime.GOARCH == "amd64" {
		fmt.Printf("warning: running calibration on %s\n", runtime.GOARCH)
	}

	// // calibration: brute-force cutoff = 24 for ASCII
	// n := sort.Search(64, func(n int) bool {
	// 	const russianText = `Владимир Маяковский родился в селе Багдади[10] Кутаисской
	// губернии Российской империи, в обедневшей дворянской семье[11] Владимира
	// Константиновича Маяковского (1857—1906), служившего лесничим третьего
	// разряда в Эриванской губернии, а с 1889 г. — в Багдатском лесничестве.
	// Маяковский вёл род от запорожских казаков, прадед отца поэта Кирилл
	// Маяковский был полковым есаулом Черноморских войск, что дало ему право
	// получить звание дворянина[12]. Мать поэта, Александра Алексеевна Павленко
	// (1867−1954), из рода кубанских казаков, родилась на Кубани, в станице
	// Терновской. В поэме «Владикавказ — Тифлис» 1924 года Маяковский называет
	// себя «грузином». О себе Маяковский сказал в 1927 году: «Родился я в
	// 1894[13] году на Кавказе. Отец был казак, мать — украинка. Первый язык —
	// грузинский. Так сказать, между тремя культурами» (из интервью пражской
	// газете «Prager Presse»)[14]. Бабушка по отцовской линии, Ефросинья Осиповна
	// Данилевская, — двоюродная сестра автора исторических романов Г. П.
	// Данилевского, родом из запорожских казаков. У Маяковского было две сестры:
	// Людмила (1884—1972) и Ольга (1890—1949) и два брата: Константин (умер в
	// трёхлетнем возрасте от скарлатины) и Александр (умер во младенчестве).`
	// 	key := "МЛАДЕНЧЕСТВЕ"
	// 	bruteForce := func(b *testing.B) {
	// 		for i := 0; i < b.N; i++ {
	// 			bruteForceIndexUnicode(russianText, key)
	// 		}
	// 	}
	// 	shortIndex := func(b *testing.B) {
	// 		for i := 0; i < b.N; i++ {
	// 			shortIndexUnicode(russianText, key)
	// 		}
	// 	}
	// 	bmbrute := testing.Benchmark(bruteForce)
	// 	bmshort := testing.Benchmark(shortIndex)
	// 	fmt.Printf("n=%d: brute=%d index=%d\n", n, bmbrute.NsPerOp(), bmshort.NsPerOp())
	// 	return bmbrute.NsPerOp()*100 > bmshort.NsPerOp()*110
	// })

	// calibration: brute-force cutoff = 24 for ASCII
	n := sort.Search(64, func(n int) bool {
		// key := "αa"
		// s := strings.Repeat("α"+strings.Repeat(" ", n-1), 1<<16/n)
		key := "aa"
		s := strings.Repeat("a"+strings.Repeat(" ", n-1), 1<<16/n)
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
}

func TestCalibrateIndexByte(t *testing.T) {
	if !*calibrate {
		return
	}

	if runtime.GOARCH == "amd64" {
		fmt.Printf("warning: running calibration on %s\n", runtime.GOARCH)
	}

	// calibration: brute-force cutoff = 24 for ASCII
	n := sort.Search(128, func(n int) bool {
		key := "vV"
		s := strings.Repeat(" ", n) + key + strings.Repeat(" ", n) + key
		c0 := key[1] // search for last element first
		c1 := key[0] // search for last element first
		cutoff := func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				m := strings.IndexByte(s, c0)
				if m == 0 {
					continue
				}
				// if m != -1 && len(s)-m >= n {
				if m != -1 && m >= n {
					s = s[:m] // limit search space
				}
				if o := strings.IndexByte(s, c1); m == -1 || (o != -1 && o < m) {
					m = o
				}
			}
		}
		nocutoff := func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				m := strings.IndexByte(s, c0)
				if m == 0 {
					continue
				}
				if o := strings.IndexByte(s, c1); m == -1 || (o != -1 && o < m) {
					m = o
				}
			}
		}
		bmcutoff := testing.Benchmark(cutoff)
		bmnaive := testing.Benchmark(nocutoff)
		fmt.Printf("n=%d: cutoff=%d index=%d\n", n, bmcutoff.NsPerOp(), bmnaive.NsPerOp())

		return bmnaive.NsPerOp()*100 > bmcutoff.NsPerOp()*110
	})
	fmt.Printf("calibration: brute-force cutoff = %d\n", n)
}

/*
var benchInputHard = makeBenchInputHard()

func benchmarkIndexHard(b *testing.B, sep string) {
	benchmarkIndex(b, benchInputHard, sep)
}

func BenchmarkIndexHard1(b *testing.B) { benchmarkIndexHard(b, "<>") }
func BenchmarkIndexHard2(b *testing.B) { benchmarkIndexHard(b, "</pre>") }
func BenchmarkIndexHard3(b *testing.B) { benchmarkIndexHard(b, "<b>hello world</b>") }
func BenchmarkIndexHard4(b *testing.B) {
	benchmarkIndexHard(b, "<pre><b>hello</b><strong>world</strong></pre>")
}
*/
