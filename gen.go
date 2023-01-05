//go:build gen
// +build gen

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
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
	slices.SortFunc(caseFolds, func(a, b foldPair) bool {
		return a.From < b.From
	})
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

func dedupe(r []rune) []rune {
	if len(r) < 2 {
		return r
	}
	slices.Sort(r)
	return slices.Compact(r)
}

func printRangeMap(w *bytes.Buffer, name, typ string, runes map[rune][]rune) {
	keys := maps.Keys(runes)
	slices.Sort(keys)

	fmt.Fprint(w, "\n\n")
	fmt.Fprintf(w, "var %s = map[rune]%s{\n", name, typ)
	for _, k := range keys {
		fmt.Fprintf(w, "\t0x%04X: {", k)
		for i, r := range runes[k] {
			if i > 0 {
				fmt.Fprint(w, ", ")
			}
			fmt.Fprintf(w, "0x%04X", r)
		}
		fmt.Fprintf(w, "}, // %q: %q\n", k, runes[k])
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

	for k, rs := range runes {
		// Make sure the key is included (was an issue with: 'ß')
		if !slices.Contains(rs, k) {
			rs = append(rs, k)
		}
		runes[k] = dedupe(rs)
	}

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

func printFoldPairsMap(w *bytes.Buffer, name string) {
	fmt.Fprintf(w, "\nvar %s = map[rune]rune{\n", name)
	for _, p := range caseFolds {
		fmt.Fprintf(w, "\t0x%04X: 0x%04X, // %q => %q\n", p.From, p.To, p.From, p.To)
	}
	fmt.Fprint(w, "}\n\n")
}

func genCaseOrbit(w *bytes.Buffer) {
	printFoldPairsMap(w, "caseOrbit")
}

func dataEqual(filename string, data []byte) bool {
	got, err := os.ReadFile(filename)
	return err == nil && bytes.Equal(got, data)
}

type overlayJSON struct {
	Replace map[string]string
}

func runCommand(args ...string) {
	cmd := exec.Command("go", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("Error:   %v", err)
		log.Printf("Command: %s", strings.Join(cmd.Args, " "))
		log.Printf("Output:  %s", bytes.TrimSpace(out))
		log.Panicf("Failed to build generated file: %v\n", err)
	}
}

func testBuild(data []byte) {
	dir, err := os.MkdirTemp("", "strcase.*")
	if err != nil {
		log.Panic(err)
	}

	tables := filepath.Join(dir, "tables.go")
	overlay := filepath.Join(dir, "overlay.json")

	overlayData, err := json.Marshal(overlayJSON{
		Replace: map[string]string{
			"tables.go": tables,
		},
	})
	if err != nil {
		log.Panic(err)
	}

	if err := os.WriteFile(overlay, overlayData, 0644); err != nil {
		log.Panic(err)
	}
	if err := os.WriteFile(tables, data, 0644); err != nil {
		log.Panic(err)
	}

	runCommand("build", "-overlay="+overlay)
	runCommand("test", "-overlay="+overlay)

	os.RemoveAll(dir) // Only remove temp dir if successful
}

func writeFile(name string, data []byte) {
	if dataEqual(name, data) {
		return
	}

	tmp := fmt.Sprintf("temp.%s.%d", name, time.Now().UnixNano())
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		log.Fatal(err)
	}
	if err := os.Rename(tmp, name); err != nil {
		os.Remove(tmp)
		log.Fatal(err)
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

// func formatSource(src []byte) []byte {
// 	cmd := exec.Command("gofmt", "-s")
// 	cmd.Stdin = bytes.NewReader(src)
// 	out, err := cmd.CombinedOutput()
// 	if err != nil {
// 		log.Fatalf("cmd: %q exited with error: %v\n%s\n",
// 			cmd.Args, err, string(out))
// 	}
// 	return out
// }

func writeGo(w *bytes.Buffer) {
	data := make([]byte, w.Len())
	copy(data, w.Bytes())
	w.Reset()
	if _, err := gen.WriteGo(w, "strcase", "", data); err != nil {
		log.Fatal(err)
	}
}

func main() {
	loadCaseFolds()

	var w bytes.Buffer
	gen.WriteUnicodeVersion(&w)
	gen.WriteCLDRVersion(&w)

	genFoldMap(&w)
	genCaseOrbit(&w)

	// WARN: use caseOrbit instead
	// printFoldPairsMap(&w, "caseFolds")

	// printMultiLengthFolds(&w, "_MultiLengthFolds")

	writeGo(&w)
	testBuild(w.Bytes())
	writeFile("tables.go", w.Bytes())

	// log.Fatal("TODO: generate case_orbit.go !!!")
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
