package strcase_test

import (
	"fmt"

	"github.com/charlievieth/strcase"
)

func ExampleCompare() {
	// ASCII
	fmt.Println(strcase.Compare("A", "b"))
	fmt.Println(strcase.Compare("A", "a"))
	fmt.Println(strcase.Compare("B", "a"))

	// Unicode
	fmt.Println(strcase.Compare("s", "ſ"))
	fmt.Println(strcase.Compare("αβδ", "ΑΒΔ"))
	// Output:
	// -1
	// 0
	// 1
	// 0
	// 0
}

func ExampleContains() {
	fmt.Println(strcase.Contains("SeaFood", "foo"))
	fmt.Println(strcase.Contains("SeaFood", "bar"))
	fmt.Println(strcase.Contains("SeaFood", ""))
	fmt.Println(strcase.Contains("", ""))
	fmt.Println(strcase.Contains("ΑΔΕΛΦΟΣΎΝΗΣ", "αδελφοσύνης"))
	// Output:
	// true
	// false
	// true
	// true
	// true
}

func ExampleContainsAny() {
	fmt.Println(strcase.ContainsAny("team", "I"))
	fmt.Println(strcase.ContainsAny("fail", "UI"))
	fmt.Println(strcase.ContainsAny("ure", "UI"))
	fmt.Println(strcase.ContainsAny("failure", "UI"))
	fmt.Println(strcase.ContainsAny("foo", ""))
	fmt.Println(strcase.ContainsAny("", ""))
	fmt.Println(strcase.ContainsAny("αβδ", "Α"))
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
	fmt.Println(strcase.ContainsRune("aardvark", 'A'))
	fmt.Println(strcase.ContainsRune("timeout", 'A'))
	// Output:
	// true
	// false
}

func ExampleCount() {
	fmt.Println(strcase.Count("cheese", "e"))
	fmt.Println(strcase.Count("five", ""))
	fmt.Println(strcase.Count("ΑΒΔ", "α"))
	fmt.Println(strcase.Count("ΑΒΔ", ""))
	// Output:
	// 3
	// 5
	// 1
	// 4
}

func ExampleHasPrefix() {
	fmt.Println(strcase.HasPrefix("Gopher", "go"))
	fmt.Println(strcase.HasPrefix("Gopher", "c"))
	fmt.Println(strcase.HasPrefix("Gopher", ""))
	// Moonlight Night (Mayakovsky) - 1916
	fmt.Println(strcase.HasPrefix("А вот и полная повисла в воздухе.", "А ВОТ"))
	// Output:
	// true
	// false
	// true
	// true
}

func ExampleHasSuffix() {
	fmt.Println(strcase.HasSuffix("Amigo", "GO"))
	fmt.Println(strcase.HasSuffix("Amigo", "AMI"))
	fmt.Println(strcase.HasSuffix("Amigo", ""))
	// Moonlight Night (Mayakovsky) - 1916
	fmt.Println(strcase.HasSuffix("А вот и полная повисла в воздухе.", "В Воздухе."))
	// Output:
	// true
	// false
	// true
	// true
}

func ExampleIndex() {
	fmt.Println(strcase.Index("chicken", "KEN"))
	fmt.Println(strcase.Index("chicken", "DMR"))
	fmt.Println(strcase.Index("日a本b語ç日ð本ê語", "Ç日Ð本Ê"))
	// Output:
	// 4
	// -1
	// 11
}

func ExampleIndexAny() {
	fmt.Println(strcase.IndexAny("chicken", "AEIOUY"))
	fmt.Println(strcase.IndexAny("crwth", "AEIOUY"))
	// Kelvin K (U+212A) matches ASCII 'K' and 'k'
	fmt.Println(strcase.IndexAny("45K", "k"))
	// Latin small letter long S 'ſ' matches ASCII 'S' and 's'
	fmt.Println(strcase.IndexAny("salsa", "ſ"))
	// Output:
	// 2
	// -1
	// 2
	// 0
}

func ExampleIndexByte() {
	fmt.Println(strcase.IndexByte("golang", 'G'))
	fmt.Println(strcase.IndexByte("gophers", 'H'))
	fmt.Println(strcase.IndexByte("golang", 'X'))
	// Latin small letter long S 'ſ' matches ASCII 'S' and 's'
	fmt.Println(strcase.IndexByte("ſinfulneſs", 's'))
	// K
	// Output:
	// 0
	// 3
	// -1
	// 0
}

func ExampleIndexRune() {
	fmt.Println(strcase.IndexRune("chicken", 'K'))
	// U+212A is the code point for Kelvin K
	fmt.Println(strcase.IndexRune("chicken", '\u212A'))
	fmt.Println(strcase.IndexRune("chicken", 'D'))
	fmt.Println(strcase.IndexRune("日a本b語ç日", 'Ç'))
	// Output:
	// 4
	// 4
	// -1
	// 11
}

func ExampleLastIndex() {
	fmt.Println(strcase.Index("go gopher", "GO"))
	fmt.Println(strcase.LastIndex("go gopher", "GO"))
	fmt.Println(strcase.LastIndex("go gopher", "rodent"))
	// Moonlight Night (Mayakovsky) - 1916
	fmt.Println(strcase.LastIndex("А вот и полная повисла в воздухе.", "ПОЛНАЯ"))
	// Output:
	// 0
	// 3
	// -1
	// 13
}

func ExampleLastIndexAny() {
	fmt.Println(strcase.LastIndexAny("go gopher", "GO"))
	fmt.Println(strcase.LastIndexAny("go gopher", "RODENT"))
	fmt.Println(strcase.LastIndexAny("go gopher", "FAIL"))
	fmt.Println(strcase.LastIndexAny("Картѣ", "РТ" /* U+0420 & U+0422 */))
	// Output:
	// 4
	// 8
	// -1
	// 6
}

func ExampleLastIndexByte() {
	fmt.Println(strcase.LastIndexByte("Hello, world", 'L'))
	fmt.Println(strcase.LastIndexByte("Hello, world", 'O'))
	// Kelvin K (U+212A) matches ASCII 'K' and 'k'
	fmt.Println(strcase.LastIndexByte("Hello, \u212Aelvin", 'k'))
	// Output:
	// 10
	// 8
	// 7
}

// // Картѣ ꙟтѫѧ а сфѫнтꙋлꙋй апостоль
// 	// КАРТѢ ꙞТѪѦ А СФѪНТꙊЛꙊЙ АПОСТОЛЬ
// 	fmt.Println(strcase.LastIndexByte("Hello, Картѣ", 'Ѣ'))
