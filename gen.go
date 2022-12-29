//go:build gen
// +build gen

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"unicode"

	"github.com/charlievieth/strcase/internal/gen"
	"github.com/charlievieth/strcase/internal/ucd"
	"golang.org/x/exp/maps"
	"golang.org/x/exp/slices"
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
	var tabs []*unicode.RangeTable
	for _, m := range maps {
		for _, t := range m {
			tabs = append(tabs, t)
		}
	}
	return rangetable.Merge(tabs...)
}

type foldPair struct {
	From uint32
	To   uint32
}

var caseFolds []foldPair

type byFoldPair []foldPair

func (b byFoldPair) Len() int           { return len(b) }
func (b byFoldPair) Less(i, j int) bool { return b[i].From < b[j].From }
func (b byFoldPair) Swap(i, j int)      { b[i], b[j] = b[j], b[i] }

func loadCaseFolds() {
	ucd.Parse(gen.OpenUCDFile("CaseFolding.txt"), func(p *ucd.Parser) {
		kind := p.String(1)
		if kind != "C" && kind != "S" {
			// Only care about 'common' and 'simple' foldings.
			return
		}
		p1 := p.Rune(0)
		p2 := p.Rune(2)
		caseFolds = append(caseFolds, foldPair{uint32(p1), uint32(p2)})
	})
	sort.Sort(byFoldPair(caseFolds))
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

// func genMustLower(w *bytes.Buffer) {
// 	var runes []rune
// 	rangetable.Visit(categories, func(r rune) {
// 		if ff := folds(r); len(ff) > 2 {
// 			runes = append(runes, ff...)
// 			return
// 		}
// 		switch lr := unicode.ToLower(r); {
// 		case r >= utf8.RuneSelf && lr < utf8.RuneSelf:
// 			runes = append(runes, r, lr, unicode.ToUpper(lr))
// 		case unicode.ToUpper(r) != unicode.ToUpper(lr):
// 			runes = append(runes, r, lr, unicode.ToUpper(lr))
// 		}
// 	})
//
// 	if len(runes) == 0 {
// 		log.Panic("Failed to generate any runes!")
// 	}
//
// 	table := rangetable.New(runes...)
// 	printRangeTable(w, "_MustLower", table)
// }

func dedupe(r []rune) []rune {
	if len(r) <= 1 {
		return r
	}
	slices.Sort(r)
	// slices.Compact(r)
	// return r
	k := 1
	for i := 1; i < len(r); i++ {
		if r[k-1] != r[i] {
			r[k] = r[i]
			k++
		}
	}
	return r[:k]
}

func printRangeMap(w *bytes.Buffer, name, typ string, runes map[rune][]rune) {
	// keys := make([]rune, 0, len(runes))
	// for k := range runes {
	// 	// WARN
	// 	// runes[k] = dedupe(rs)
	// 	keys = append(keys, k)
	// }
	// sort.Sort(byRune(keys))

	keys := maps.Keys(runes)
	slices.Sort(keys)

	fmt.Fprint(w, "\n\n")
	fmt.Fprintf(w, "var %s = map[rune]%s{\n", name, typ)
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

func printIndexMap(w *bytes.Buffer, name string, runes map[rune]rune) {
	// keys := make([]rune, 0, len(runes))
	// for k := range runes {
	// 	keys = append(keys, k)
	// }
	// sort.Sort(byRune(keys))
	keys := maps.Keys(runes)
	slices.Sort(keys)

	fmt.Fprint(w, "\n\n")
	fmt.Fprintf(w, "var %s = map[rune]rune{\n", name)
	for _, k := range keys {
		if k <= math.MaxUint16 {
			fmt.Fprintf(w, "\t%#04X: ", k)
		} else {
			fmt.Fprintf(w, "\t%#06X: ", k)
		}
		r := runes[k]
		if r <= math.MaxUint16 {
			fmt.Fprintf(w, "%#04X, // %q\n", r, k)
		} else {
			fmt.Fprintf(w, "%#06X, // %q\n", r, k)
		}
	}
	fmt.Fprintln(w, "}")
}

// TODO: update other gen func to match this one
//
// WARN: use caseFolds
func genFoldMap(w *bytes.Buffer) {
	runes := make(map[rune][]rune)
	rangetable.Visit(categories, func(r rune) {
		ff := folds(r)
		if len(ff) > 2 {
			runes[r] = append(runes[r], ff...)
		}
		// WARN
		if len(ff) == 1 && unicode.ToUpper(r) != unicode.ToLower(r) {
			runes[r] = append(runes[r], ff...)
		}
	})
	// WARN WARN WARN: we should not need to add this manually
	runes['İ'] = append(runes['İ'], 'İ')
	runes['ß'] = append(runes['ß'], 'ẞ')

	keys := make([]rune, 0, len(runes))
	for k, rs := range runes {
		keys = append(keys, k)

		// Make sure the key is included (was an issue with: 'ß')
		if !slices.Contains(rs, k) {
			rs = append(rs, k)
		}
		runes[k] = dedupe(rs)
	}
	keys = dedupe(keys)

	if len(runes) == 0 {
		log.Panic("Failed to generate any runes!")
	}

	// TODO: use `[4]rune`
	printRangeMap(w, "_FoldMap", "[]rune", runes)

	noUpperLower := make(map[rune][]rune)
	for k, rs := range runes {
		u := unicode.ToUpper(k)
		l := unicode.ToLower(k)
		a := make([]rune, 0, 2)
		for _, r := range rs {
			if r != u && r != l {
				a = append(a, r)
			}
		}
		if len(a) > 2 {
			log.Fatalf("fold set excluding upper/lower %q "+
				"must have 2 or less elements got: %d", a, len(a))
		}
		switch len(a) {
		case 0:
			a = append(a, k, k)
		case 1:
			a = append(a, a[0])
		}
		noUpperLower[k] = a
	}

	printRangeMap(w, "_FoldMapExcludingUpperLower", "[2]rune", noUpperLower)
}

func fmtUint(u uint32) string {
	if u <= math.MaxUint16 {
		return fmt.Sprintf("%#04X", u)
	}
	return fmt.Sprintf("%#06X", u)
}

func printFoldPairsMap(w *bytes.Buffer, name string) {
	pairs := caseFolds
	if !sort.IsSorted(byFoldPair(pairs)) {
		log.Panic("pairs are not sorted")
	}

	fmt.Fprintf(w, "\nvar %s = map[rune]rune{\n", name)
	for _, p := range pairs {
		fmt.Fprintf(w, "\t%s: %s, // %q\t=>\t%q\n", fmtUint(p.From), fmtUint(p.To), p.From, p.To)
	}
	fmt.Fprint(w, "}\n\n")
}

// func printMultiLengthFolds(w *bytes.Buffer, name string) {
// 	pairs := caseFolds
// 	if !sort.IsSorted(byFoldPair(pairs)) {
// 		log.Panic("pairs are not sorted")
// 	}
//
// 	var runes []rune
// 	for _, p := range pairs {
// 		if utf8.RuneLen(rune(p.From)) != utf8.RuneLen(rune(p.To)) {
// 			runes = append(runes, rune(p.From), rune(p.To))
// 		}
// 	}
// 	runes = dedupe(runes)
//
// 	fmt.Fprintf(w, "\nvar %s = [...]rune{\n", name)
// 	for _, r := range runes {
// 		if r <= math.MaxUint16 {
// 			fmt.Fprintf(w, "\t%#04X, // %q\n", r, r)
// 		} else {
// 			fmt.Fprintf(w, "\t%#06X, // %q\n", r, r)
// 		}
// 	}
// 	fmt.Fprint(w, "}\n\n")
// }

func writeHeader(w *bytes.Buffer) {
	const hdr = `// Code generated by running "go generate" in github.com/charlievieth/strcase. DO NOT EDIT.
	// Unicode version: %s

package strcase

`
	fmt.Fprintf(w, hdr, unicode.Version)
}

func dataEqual(filename string, data []byte) bool {
	got, err := os.ReadFile(filename)
	return err == nil && bytes.Equal(got, data)
}

func testBuild(data []byte) {
	// abs, err := filepath.Abs("tables.go")
	// if err != nil {
	// 	log.Fatal(err)
	// }

	f, err := os.CreateTemp(filepath.Dir("."), "strcase-overlay.*")
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		f.Close()
		os.Remove(f.Name())
	}()

	type OverlayJSON struct {
		Replace map[string]string
	}
	enc := json.NewEncoder(f)
	err = enc.Encode(OverlayJSON{
		Replace: map[string]string{
			"tables.go": string(data),
		},
	})
	if err != nil {
		log.Fatal(err)
	}
	cmd := exec.Command("go", "build", "-overlay", f.Name())
	out, err := cmd.CombinedOutput()
	if err != nil {
		log.Println("Error:", err)
		log.Println(string(out))
	}
	log.Println("OK")
}

func writeFile(name string, data []byte) {
	testBuild(data)
	var buf bytes.Buffer
	if _, err := gen.WriteGo(&buf, "strcase", "", data); err != nil {
		log.Fatal(err)
	}
	// TODO: use `go build -overlay` to test the change
	if dataEqual(name, buf.Bytes()) {
		return
	}

	tmp := name + ".tmp"
	f, err := os.Create(tmp)
	if err != nil {
		log.Fatal(err)
	}
	fatal := func(err error) {
		f.Close()
		os.Remove(tmp)
		log.Panic(err)
	}
	if _, err := buf.WriteTo(f); err != nil {
		fatal(err)
	}
	if err := os.Rename(name+".tmp", name); err != nil {
		fatal(err)
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
	// rangetable.Visit(unicode.L, func(r rune) {
	// rangetable.Visit(categories, func(r rune) {
	// 	if ff := folds(r); len(ff) == 1 && unicode.ToUpper(r) != unicode.ToLower(r) {
	// 		fmt.Printf("%q\n", r)
	// 	}
	// })
	// return

	var runes []rune
	for _, p := range caseFolds {
		runes = append(runes, rune(p.From), rune(p.To))
	}
	// rangetable.Visit(categories, func(r rune) {
	// 	if ff := folds(r); len(ff) > 2 {
	// 		runes = append(runes, ff...)
	// 		return
	// 	}
	// })
	if len(runes) == 0 {
		log.Panic("Failed to generate any runes!")
	}

	// table := rangetable.New(runes...)

	fmt.Fprintln(w, "// WARN: do we need this ???")
	// printRangeTable(w, "_Foldable", table)
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
	// if _, err := exec.LookPath("gofmt"); err != nil {
	// 	log.Fatal(err)
	// }
	loadCaseFolds()

	var w bytes.Buffer
	gen.WriteUnicodeVersion(&w)
	gen.WriteCLDRVersion(&w)

	genFoldMap(&w)

	// WARN: use caseOrbit instead
	// printFoldPairsMap(&w, "caseFolds")

	// printMultiLengthFolds(&w, "_MultiLengthFolds")

	writeFile("tables.go", w.Bytes())
	// gen.WriteGoFile("tables.go", "strcase", w.Bytes())

	// src, err := format.Source(w.Bytes())
	// if err != nil {
	// 	log.Println("Error:", err)
	// 	log.Println("##### Source:")
	// 	log.Println(w.String())
	// 	log.Println("#####")
	// 	log.Panic(err)
	// }
	// src := formatSource(w.Bytes())
	// writeFile("tables.go", src)
}

// func printSwitch(w *bytes.Buffer, name string, runes []rune) {
// 	// if !sort.IsSorted(byRune(runes)) {
// 	// 	sort.Sort(byRune(runes))
// 	// }
// 	runes = dedupe(runes)
//
// 	fmt.Fprintf(w, "\nfunc %s(r rune) bool {\n", name)
// 	fmt.Fprintln(w, "\tswitch r {")
// 	fmt.Fprintf(w, "\tcase ")
//
// 	for i := 0; i < 8 && len(runes) > 0; i++ {
// 		r := runes[0]
// 		if r <= math.MaxUint16 {
// 			fmt.Fprintf(w, "%#04X, ", r)
// 		} else {
// 			fmt.Fprintf(w, "%#06X, ", r)
// 		}
// 		// fmt.Fprintf(w, "%#04X, ", runes[i])
// 		runes = runes[1:]
// 	}
// 	fmt.Fprintf(w, "\n")
// 	// fmt.Fprintln(w, ":")
//
// 	for len(runes) > 0 {
// 		for i := 0; i < 8 && len(runes) > 0; i++ {
// 			if i != 0 {
// 				w.WriteString(", ")
// 			}
// 			r := runes[0]
// 			if r <= math.MaxUint16 {
// 				fmt.Fprintf(w, "%#04X", r)
// 			} else {
// 				fmt.Fprintf(w, "%#06X", r)
// 			}
// 			// fmt.Fprintf(w, "%#04X", runes[0])
// 			runes = runes[1:]
// 		}
// 		if len(runes) > 0 {
// 			w.WriteString(",\n\t\t")
// 		} else {
// 			w.WriteString(":\n")
// 		}
// 	}
// 	fmt.Fprintln(w, "\t\treturn true")
// 	fmt.Fprintln(w, "\t}")
// 	fmt.Fprintln(w, "\treturn false")
// 	fmt.Fprintln(w, "}")
// 	fmt.Fprint(w, "\n")
// }
