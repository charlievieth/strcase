package bytcase_test

import (
	"fmt"
	"sort"
	"unicode/utf8"

	"github.com/charlievieth/strcase/bytcase"
)

// // Index returns the index of the first instance of substr in s,
// // or -1 if substr is not present in s.
// // If ignoreCase is true the match is case-insensitive.
// func Index(s, substr string, ignoreCase bool) int {
// 	if ignoreCase {
// 		return bytcase.Index(s, substr)
// 	}
// 	return strings.Index(s, substr)
// }

func ExampleCompare() {
	// ASCII
	fmt.Println(bytcase.Compare([]byte("A"), []byte("b")))
	fmt.Println(bytcase.Compare([]byte("A"), []byte("a")))
	fmt.Println(bytcase.Compare([]byte("B"), []byte("a")))

	// Unicode
	fmt.Println(bytcase.Compare([]byte("s"), []byte("ſ")))
	fmt.Println(bytcase.Compare([]byte("αβδ"), []byte("ΑΒΔ")))

	// All invalid UTF-8 sequences are considered equal
	fmt.Println(bytcase.Compare([]byte("\xff"), []byte(string(utf8.RuneError))))
	// Output:
	// -1
	// 0
	// 1
	// 0
	// 0
	// 0
}

// Case insensitive sort using [bytcase.Compare].
func ExampleCompare_sort() {
	a := [][]byte{
		[]byte("b"),
		[]byte("a"),
		[]byte("α"),
		[]byte("B"),
		[]byte("Α"), // U+0391
		[]byte("A"),
	}
	sort.SliceStable(a, func(i, j int) bool {
		return bytcase.Compare(a[i], a[j]) < 0
	})
	fmt.Printf("%q\n", a)
	// Output:
	// ["a" "A" "b" "B" "α" "Α"]
}

// Case insensitive search using [bytcase.Compare].
func ExampleCompare_search() {
	a := [][]byte{
		[]byte("a"),
		[]byte("b"),
		[]byte("α"),
	}
	s := []byte("B") // string being searched for
	i := sort.Search(len(a), func(i int) bool {
		return bytcase.Compare(a[i], s) >= 0
	})

	fmt.Printf("%d: %q\n", i, a[i])
	// Output:
	// 1: "b"
}

// Using [bytcase.Compare] and [sort.Find] to search a string slice.
func ExampleCompare_find() {
	a := []string{
		"a",
		"b",
		"α",
	}
	for _, s := range []string{"A", "B", "Z"} {
		i, found := sort.Find(len(a), func(i int) int {
			return bytcase.Compare([]byte(s), []byte(a[i]))
		})
		if found {
			fmt.Printf("%q found at index %d\n", s, i)
		} else {
			fmt.Printf("%q not found", s)
		}
	}
	// Output:
	// "A" found at index 0
	// "B" found at index 1
	// "Z" not found
}

func ExampleContains() {
	fmt.Println(bytcase.Contains([]byte("SeaFood"), []byte("foo")))
	fmt.Println(bytcase.Contains([]byte("SeaFood"), []byte("bar")))
	fmt.Println(bytcase.Contains([]byte("SeaFood"), []byte("")))
	fmt.Println(bytcase.Contains([]byte(""), []byte("")))
	fmt.Println(bytcase.Contains([]byte("ΑΔΕΛΦΟΣΎΝΗΣ"), []byte("αδελφοσύνης")))
	// All invalid UTF-8 sequences are considered equal
	fmt.Println(bytcase.Contains([]byte("\xed\xa0\x80\x80"), []byte("\xed\xbf\xbf\x80")))
	// Output:
	// true
	// false
	// true
	// true
	// true
	// true
}

func ExampleContainsAny() {
	fmt.Println(bytcase.ContainsAny([]byte("team"), []byte("I")))
	fmt.Println(bytcase.ContainsAny([]byte("fail"), []byte("UI")))
	fmt.Println(bytcase.ContainsAny([]byte("ure"), []byte("UI")))
	fmt.Println(bytcase.ContainsAny([]byte("failure"), []byte("UI")))
	fmt.Println(bytcase.ContainsAny([]byte("foo"), []byte("")))
	fmt.Println(bytcase.ContainsAny([]byte(""), []byte("")))
	fmt.Println(bytcase.ContainsAny([]byte("αβδ"), []byte("Α")))
	// Output:
	// false
	// true
	// true
	// true
	// false
	// false
	// true
}

func ExampleContainsRune() {
	// Finds whether a string contains a particular Unicode code point.
	fmt.Println(bytcase.ContainsRune([]byte("aardvark"), 'A'))
	fmt.Println(bytcase.ContainsRune([]byte("timeout"), 'A'))
	// Output:
	// true
	// false
}

func ExampleCount() {
	fmt.Println(bytcase.Count([]byte("cheese"), []byte("e")))
	fmt.Println(bytcase.Count([]byte("five"), []byte("")))
	fmt.Println(bytcase.Count([]byte("ΑΒΔ"), []byte("α")))
	fmt.Println(bytcase.Count([]byte("ΑΒΔ"), []byte("")))
	// Output:
	// 3
	// 5
	// 1
	// 4
}

func ExampleHasPrefix() {
	fmt.Println(bytcase.HasPrefix([]byte("Gopher"), []byte("go")))
	fmt.Println(bytcase.HasPrefix([]byte("Gopher"), []byte("c")))
	fmt.Println(bytcase.HasPrefix([]byte("Gopher"), []byte("")))
	// Moonlight Night (Mayakovsky) - 1916
	fmt.Println(bytcase.HasPrefix([]byte("А вот и полная повисла в воздухе."), []byte("А ВОТ")))
	// Output:
	// true
	// false
	// true
	// true
}

func ExampleHasSuffix() {
	fmt.Println(bytcase.HasSuffix([]byte("Amigo"), []byte("GO")))
	fmt.Println(bytcase.HasSuffix([]byte("Amigo"), []byte("AMI")))
	fmt.Println(bytcase.HasSuffix([]byte("Amigo"), []byte("")))
	// Moonlight Night (Mayakovsky) - 1916
	fmt.Println(bytcase.HasSuffix([]byte("А вот и полная повисла в воздухе."), []byte("В Воздухе.")))
	// Output:
	// true
	// false
	// true
	// true
}

func ExampleIndex() {
	fmt.Println(bytcase.Index([]byte("chicken"), []byte("KEN")))
	fmt.Println(bytcase.Index([]byte("chicken"), []byte("DMR")))
	fmt.Println(bytcase.Index([]byte("日a本b語ç日ð本ê語"), []byte("Ç日Ð本Ê")))

	// All invalid UTF-8 sequences are considered equal
	fmt.Println(bytcase.Index([]byte("a\xff"), []byte(string(utf8.RuneError))))
	fmt.Println(bytcase.Index([]byte("abc\xed\xa0\x80\x80"), []byte("\xed\xbf\xbf\x80")))
	// Output:
	// 4
	// -1
	// 11
	// 1
	// 3
}

func ExampleIndexAny() {
	fmt.Println(bytcase.IndexAny([]byte("chicken"), []byte("AEIOUY")))
	fmt.Println(bytcase.IndexAny([]byte("crwth"), []byte("AEIOUY")))
	// Kelvin K (U+212A) matches ASCII 'K' and 'k'
	fmt.Println(bytcase.IndexAny([]byte("45K"), []byte("k")))
	// Latin small letter long S 'ſ' matches ASCII 'S' and 's'
	fmt.Println(bytcase.IndexAny([]byte("salsa"), []byte("ſ")))
	// Output:
	// 2
	// -1
	// 2
	// 0
}

func ExampleIndexByte() {
	fmt.Println(bytcase.IndexByte([]byte("golang"), 'G'))
	fmt.Println(bytcase.IndexByte([]byte("gophers"), 'H'))
	fmt.Println(bytcase.IndexByte([]byte("golang"), 'X'))
	// Latin small letter long S 'ſ' matches ASCII 'S' and 's'
	fmt.Println(bytcase.IndexByte([]byte("ſinfulneſs"), 's'))
	// K
	// Output:
	// 0
	// 3
	// -1
	// 0
}

func ExampleIndexRune() {
	fmt.Println(bytcase.IndexRune([]byte("chicken"), 'K'))
	// U+212A is the code point for Kelvin K
	fmt.Println(bytcase.IndexRune([]byte("chicken"), '\u212A'))
	fmt.Println(bytcase.IndexRune([]byte("chicken"), 'D'))
	fmt.Println(bytcase.IndexRune([]byte("日a本b語ç日"), 'Ç'))
	// Output:
	// 4
	// 4
	// -1
	// 11
}

func ExampleLastIndex() {
	fmt.Println(bytcase.Index([]byte("go gopher"), []byte("GO")))
	fmt.Println(bytcase.LastIndex([]byte("go gopher"), []byte("GO")))
	fmt.Println(bytcase.LastIndex([]byte("go gopher"), []byte("rodent")))
	// Moonlight Night (Mayakovsky) - 1916
	fmt.Println(bytcase.LastIndex([]byte("А вот и полная повисла в воздухе."), []byte("ПОЛНАЯ")))
	// Output:
	// 0
	// 3
	// -1
	// 13
}

func ExampleLastIndexAny() {
	fmt.Println(bytcase.LastIndexAny([]byte("go gopher"), []byte("GO")))
	fmt.Println(bytcase.LastIndexAny([]byte("go gopher"), []byte("RODENT")))
	fmt.Println(bytcase.LastIndexAny([]byte("go gopher"), []byte("FAIL")))
	fmt.Println(bytcase.LastIndexAny([]byte("Картѣ"), []byte("РТ") /* U+0420 & U+0422 */))
	// Output:
	// 4
	// 8
	// -1
	// 6
}

func ExampleLastIndexByte() {
	fmt.Println(bytcase.LastIndexByte([]byte("Hello, world"), 'L'))
	fmt.Println(bytcase.LastIndexByte([]byte("Hello, world"), 'O'))
	// Kelvin K (U+212A) matches ASCII 'K' and 'k'
	fmt.Println(bytcase.LastIndexByte([]byte("Hello, \u212Aelvin"), 'k'))
	// Output:
	// 10
	// 8
	// 7
}

func ExampleEqualFold() {
	fmt.Println(bytcase.EqualFold([]byte("Go"), []byte("go")))
	// true because comparison uses simple case-folding
	fmt.Println(bytcase.EqualFold([]byte("AB"), []byte("ab")))
	// false because comparison does not use full case-folding
	fmt.Println(bytcase.EqualFold([]byte("ß"), []byte("ss")))
	// Output:
	// true
	// true
	// false
}

func ExampleCut() {
	show := func(s, sep string) {
		before, after, found := bytcase.Cut([]byte(s), []byte(sep))
		fmt.Printf("Cut(%q, %q) = %q, %q, %v\n", s, sep, before, after, found)
	}
	show("Gopher", "GO")
	show("Gopher", "Ph")
	show("Gopher", "Er")
	show("Gopher", "Badger")
	show("123 αβδ 456", "ΑΒΔ")
	// Output:
	// Cut("Gopher", "GO") = "", "pher", true
	// Cut("Gopher", "Ph") = "Go", "er", true
	// Cut("Gopher", "Er") = "Goph", "", true
	// Cut("Gopher", "Badger") = "Gopher", "", false
	// Cut("123 αβδ 456", "ΑΒΔ") = "123 ", " 456", true
}

func ExampleCutPrefix() {
	show := func(s, sep string) {
		after, found := bytcase.CutPrefix([]byte(s), []byte(sep))
		fmt.Printf("CutPrefix(%q, %q) = %q, %v\n", s, sep, after, found)
	}
	show("Gopher", "Go")
	show("Gopher", "Ph")
	// Output:
	// CutPrefix("Gopher", "Go") = "pher", true
	// CutPrefix("Gopher", "Ph") = "Gopher", false
}

func ExampleCutSuffix() {
	show := func(s, sep string) {
		before, found := bytcase.CutSuffix([]byte(s), []byte(sep))
		fmt.Printf("CutSuffix(%q, %q) = %q, %v\n", s, sep, before, found)
	}
	show("Gopher", "Go")
	show("Gopher", "Er")
	// Output:
	// CutSuffix("Gopher", "Go") = "Gopher", false
	// CutSuffix("Gopher", "Er") = "Goph", true
}

func ExampleIndexNonASCII() {
	fmt.Println(bytcase.IndexNonASCII([]byte("日a本b語ç日")))
	fmt.Println(bytcase.IndexNonASCII([]byte("abc語")))
	fmt.Println(bytcase.IndexNonASCII([]byte("abc")))
	// Output:
	// 0
	// 3
	// -1
}

func ExampleContainsNonASCII() {
	fmt.Println(bytcase.ContainsNonASCII([]byte("日a本b語ç日")))
	fmt.Println(bytcase.ContainsNonASCII([]byte("abc語")))
	fmt.Println(bytcase.ContainsNonASCII([]byte("abc")))
	// Output:
	// true
	// true
	// false
}
