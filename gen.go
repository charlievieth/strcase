//go:build gen
// +build gen

package main

import (
	"bytes"
	"fmt"
	"go/format"
	"log"
	"math"
	"os"
	"os/exec"
	"sort"
	"unicode"
	"unicode/utf8"

	"golang.org/x/text/unicode/rangetable"
)

func init() {
	log.SetPrefix("[gen] ")
	log.SetFlags(log.Lshortfile)
}

var categories = rangetable.Merge(mapToTable(
	unicode.Categories,
	unicode.Scripts,
	unicode.Properties,
	unicode.FoldCategory,
	unicode.FoldScript,
))

func mapToTable(maps ...map[string]*unicode.RangeTable) *unicode.RangeTable {
	n := 0
	for _, m := range maps {
		n += len(m)
	}
	tabs := make([]*unicode.RangeTable, 0, n)
	for _, m := range maps {
		for _, t := range m {
			tabs = append(tabs, t)
		}
	}
	return rangetable.Merge(tabs...)
}

func printRangeTable(w *bytes.Buffer, name string, rt *unicode.RangeTable) {
	rt = rangetable.Merge(rt) // Optimize

	fmt.Fprintf(w, "var %s = &unicode.RangeTable{\n", name)
	if len(rt.R16) == 0 {
		fmt.Fprintln(w, "\tR16: []unicode.Range16{},")
	} else {
		fmt.Fprintln(w, "\tR16: []unicode.Range16{")
		for _, r := range rt.R16 {
			fmt.Fprintf(w, "\t\t{%#04X, %#04X, %d}, // %q - %q\n", r.Lo, r.Hi, r.Stride, r.Lo, r.Hi)
		}
		fmt.Fprintln(w, "\t},")
	}
	if len(rt.R32) == 0 {
		fmt.Fprintln(w, "\tR32: []unicode.Range32{},")
	} else {
		fmt.Fprintln(w, "\tR32: []unicode.Range32{")
		for _, r := range rt.R32 {
			fmt.Fprintf(w, "\t\t{%#06X, %#06X, %d}, // %q - %q\n", r.Lo, r.Hi, r.Stride, r.Lo, r.Hi)
		}
		fmt.Fprintln(w, "\t},")
	}
	fmt.Fprintln(w, "}")
	fmt.Fprint(w, "\n\n")
}

func folds(sr rune) []rune {
	r := unicode.SimpleFold(sr)
	runes := make([]rune, 1, 2)
	runes[0] = sr
	for r != sr {
		runes = append(runes, r)
		r = unicode.SimpleFold(r)
	}
	return runes
}

func genMustLower(w *bytes.Buffer) {
	var runes []rune
	rangetable.Visit(categories, func(r rune) {
		if ff := folds(r); len(ff) > 2 {
			runes = append(runes, ff...)
			return
		}
		switch lr := unicode.ToLower(r); {
		case r >= utf8.RuneSelf && lr < utf8.RuneSelf:
			runes = append(runes, r, lr, unicode.ToUpper(lr))
		case unicode.ToUpper(r) != unicode.ToUpper(lr):
			runes = append(runes, r, lr, unicode.ToUpper(lr))
		}
	})

	if len(runes) == 0 {
		log.Panic("Failed to generate any runes!")
	}

	table := rangetable.New(runes...)
	printRangeTable(w, "_MustLower", table)
}

type byRune []rune

func (r byRune) Len() int           { return len(r) }
func (r byRune) Swap(i, j int)      { r[i], r[j] = r[j], r[i] }
func (r byRune) Less(i, j int) bool { return r[i] < r[j] }

func dedupe(r []rune) []rune {
	sort.Sort(byRune(r))
	k := 1
	for i := 1; i < len(r); i++ {
		if r[k-1] != r[i] {
			r[k] = r[i]
			k++
		}
	}
	return r[:k]
}

func printRangeMap(w *bytes.Buffer, name string, runes map[rune][]rune) {
	keys := make([]rune, 0, len(runes))
	for k, rs := range runes {
		runes[k] = dedupe(rs)
		keys = append(keys, k)
	}
	sort.Sort(byRune(keys))

	fmt.Fprint(w, "\n\n")
	fmt.Fprintf(w, "var %s = map[rune][]rune{\n", name)
	for _, k := range keys {
		if k <= math.MaxUint16 {
			fmt.Fprintf(w, "\t%#04X: {", k)
		} else {
			fmt.Fprintf(w, "\t%#06X: {", k)
		}
		for i, r := range runes[k] {
			if i > 0 {
				fmt.Fprint(w, ", ")
			}
			if r <= math.MaxUint16 {
				fmt.Fprintf(w, "%#04X", r)
			} else {
				fmt.Fprintf(w, "%#06X", r)
			}
		}
		fmt.Fprintf(w, "}, // %q: %q\n", k, runes[k])
	}
	fmt.Fprintln(w, "}")
}

// WARN: this is wrong since it includes folds that don't match ToLower.
// Need to change this to only include runes that map with ToLower
//
// TODO: update other gen func to match this one
func genFoldMap(w *bytes.Buffer) {
	runes := make(map[rune][]rune)
	rangetable.Visit(categories, func(r rune) {
		if ff := folds(r); len(ff) > 2 {
			runes[r] = append(runes[r], ff...)
		}

		lr := unicode.ToLower(r)
		ur := unicode.ToUpper(r)
		if r >= utf8.RuneSelf && lr < utf8.RuneSelf {
			runes[r] = append(runes[r], r, lr, unicode.ToUpper(lr))
		}
		if unicode.ToUpper(lr) != ur {
			runes[r] = append(runes[r], r, lr, ur)
		}
		if unicode.ToLower(ur) != lr {
			runes[r] = append(runes[r], r, lr, ur)
		}
		// if unicode.IsUpper(r) && unicode.ToUpper(lr) != r {
		// 	runes[r] = append(runes[r], r, lr)
		// }
	})

	if len(runes) == 0 {
		log.Panic("Failed to generate any runes!")
	}

	// Remove runes that don't map with ToLower
	trim := func(r rune, rs []rune) []rune {
		a := rs[:0]
		for i := 0; i < len(rs); i++ {
			rr := rs[i]
			if rr == r || unicode.ToLower(rr) == unicode.ToLower(r) {
				a = append(a, rr)
			}
		}
		return a
	}
	_ = trim

	// for _, rs := range runes {
	// 	for _, r := range rs {
	// 		rs := trim(r, runes[r])
	// 		if len(rs) >= 2 {
	// 			runes[r] = rs
	// 		} else {
	// 			delete(runes, r)
	// 		}
	// 	}
	// }

	// WARN: this comment is wrong!!!
	//
	// WARN: don't add runes that don't map from upper/lower and vise versa
	// like: 'С' \u0421 and 'ᲃ' \u1c83
	//
	// Make sure all runes are mapped
	// for _, rs := range runes {
	// 	for _, r := range rs {
	// 		runes[r] = append(runes[r], rs...)
	// 	}
	// }

	for k, rs := range runes {
		runes[k] = dedupe(rs)
	}

	printRangeMap(w, "_FoldMap", runes)
}

func genFoldMap2(w *bytes.Buffer) {
	runes := make(map[rune][]rune)
	rangetable.Visit(categories, func(r rune) {
		if ff := folds(r); len(ff) > 2 {
			runes[r] = append(runes[r], ff...)
		}
	})

	if len(runes) == 0 {
		log.Panic("Failed to generate any runes!")
	}

	printRangeMap(w, "_FoldMap2", runes)
}

func writeHeader(w *bytes.Buffer) {
	const hdr = `// Code generated by running "go generate" in github.com/charlievieth/strcase. DO NOT EDIT.

package strcase

import "unicode"

`
	w.WriteString(hdr)
}

func sameData(filename string, data []byte) bool {
	got, _ := os.ReadFile(filename)
	return bytes.Equal(got, data)
}

func writeFile(name string, data []byte) {
	if got, _ := os.ReadFile(name); bytes.Equal(got, data) {
		return
	}
	if err := os.WriteFile(name+".tmp", data, 0644); err != nil {
		log.Panic(err)
	}
	if err := os.Rename(name+".tmp", name); err != nil {
		log.Panic(err)
	}
}

// func maxDelta() {
// 	rangetable.Visit(categories, func(r rune) {
// 		l := unicode.ToLower(r)
// 		n0 := utf8.RuneLen(r)
// 		n1 := utf8.RuneLen(l)
// 		d := n0 - n1
// 		if d < 0 {
// 			d = n1 - n0
// 		}
// 		if d >= 1 {
// 			fmt.Printf("%q: %d\n", r, d)
// 		}
// 	})
// }

// WARN WARN WARN WARN WARN WARN WARN
// NEW
// WARN WARN WARN WARN WARN WARN WARN
func genFoldableRunes(w *bytes.Buffer) {
	var runes []rune
	rangetable.Visit(categories, func(r rune) {
		if ff := folds(r); len(ff) > 2 {
			runes = append(runes, ff...)
			return
		}
	})

	if len(runes) == 0 {
		log.Panic("Failed to generate any runes!")
	}

	table := rangetable.New(runes...)

	fmt.Fprintln(w, "// WARN: do we need this ???")
	printRangeTable(w, "_Foldable", table)
}

func formatSource(src []byte) []byte {
	cmd := exec.Command("gofmt", "-s")
	cmd.Stdin = bytes.NewReader(src)
	out, err := cmd.CombinedOutput()
	if err != nil {
		log.Fatalf("cmd: %q exited with error: %v\n%s\n",
			cmd.Args, err, string(out))
	}
	return out
}

func main() {
	if _, err := exec.LookPath("gofmt"); err != nil {
		log.Fatal(err)
	}

	var w bytes.Buffer
	writeHeader(&w)
	genMustLower(&w)
	// WARN: new
	genFoldableRunes(&w)
	genFoldMap(&w)
	// WARN: dev only
	genFoldMap2(&w)

	src, err := format.Source(w.Bytes())
	if err != nil {
		log.Println("Error:", err)
		log.Println("##### Source:")
		log.Println(w.String())
		log.Println("#####")
		log.Panic(err)
	}
	// src := formatSource(w.Bytes())
	writeFile("tables.go", src)
}
