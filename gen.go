// Copyright 2023 Charlie Vieth. All rights reserved.
// Use of this source code is governed by the MIT license.

//go:build gen
// +build gen

package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"flag"
	"fmt"
	"go/format"
	"io"
	"log"
	"math"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
	"unicode"

	"github.com/schollz/progressbar/v3"
	"golang.org/x/exp/maps"
	"golang.org/x/exp/slices"
	"golang.org/x/term"
	"golang.org/x/text/unicode/rangetable"

	"github.com/charlievieth/strcase/internal/gen"
	"github.com/charlievieth/strcase/internal/ucd"
)

const (
	caseFoldShift = 19
	caseFoldSize  = 8192
)

// Unicode categories used to construct the fold maps
var unicodeCategories = []map[string]*unicode.RangeTable{
	unicode.Categories,
	unicode.Scripts,
	unicode.Properties,
	unicode.FoldCategory,
	unicode.FoldScript,
}

var categories *unicode.RangeTable

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

func loadCategories() {
	tabs := make([]*unicode.RangeTable, 0, len(unicodeCategories))
	for _, m := range unicodeCategories {
		for _, t := range m {
			tabs = append(tabs, t)
		}
	}
	categories = rangetable.Merge(tabs...)
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
// TODO: use caseFolds if more performant
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
	// WARN: we should not need to add this manually
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

func hash(x, c uint32) uint32 {
	m := x * c
	return m >> caseFoldShift
}

var emptySeen [caseFoldSize]bool

type span struct {
	start, end int64
}

func generateSpans(start, end, delta int64) []span {
	maxEnd := end
	var spans []span
	for i := start; i <= end; i += delta {
		start := i
		if start == 0 {
			start = 1
		}
		end := i + delta
		if end >= maxEnd {
			end = maxEnd
		}
		spans = append(spans, span{start: int64(start), end: int64(end)})
	}
	return spans
}

func shuffleSpans(spans []span) []span {
	rr := rand.New(rand.NewSource(time.Now().UnixNano()))
	rr.Shuffle(len(spans), func(i, j int) {
		spans[i], spans[j] = spans[j], spans[i]
	})
	return spans
}

func genCaseFoldHashValues(inputs []uint32) (tableSize int, hashSeed uint32) {
	const delta = 500_000

	// This can take awhile so use a progress bar.
	var bar *progressbar.ProgressBar
	if term.IsTerminal(syscall.Stdout) {
		bar = progressbar.Default(math.MaxUint32)
	} else {
		bar = progressbar.DefaultSilent(math.MaxUint32)
	}

	numCPU := runtime.NumCPU()
	if numCPU >= 8 {
		numCPU -= 2
	}

	ch := make(chan *span, numCPU*2)
	go func() {
		spans := shuffleSpans(generateSpans(1, math.MaxUint32, delta))
		for i := range spans {
			ch <- &spans[i]
		}
		close(ch)
	}()

	var (
		bestIdx  = int64(math.MaxInt64)
		bestSeed uint32
		mu       sync.Mutex
		wg       sync.WaitGroup
	)
	for i := 0; i < numCPU; i++ {
		wg.Add(1)
		go func(id int, inputs []uint32) {
			defer wg.Done()
			seen := new([caseFoldSize]bool)
			for sp := range ch {
				best := atomic.LoadInt64(&bestIdx)
			Loop:
				for i := sp.start; i <= sp.end && best >= int64(len(inputs)); i++ {
					maxIdx := int64(0)
					*seen = emptySeen
					seed := uint32(i)
					for _, x := range inputs {
						u := int64(hash(seed, x))
						if u > best {
							continue Loop
						}
						if seen[u] {
							continue Loop
						}
						seen[u] = true
						if u > maxIdx {
							maxIdx = u
						}
					}
					best = atomic.LoadInt64(&bestIdx)
					if maxIdx < best {
						// Use mutex to simplify updating both values
						mu.Lock()
						best = atomic.LoadInt64(&bestIdx) // re-check
						if maxIdx < best {
							atomic.StoreInt64(&bestIdx, maxIdx)
							atomic.StoreUint32(&bestSeed, seed)
						}
						mu.Unlock()
					}
				}
				bar.Add64(sp.end - sp.start)
			}
		}(i, inputs)
	}
	wg.Wait()

	if bestIdx == math.MaxInt64 || bestSeed == 0 {
		log.Panic("failed to generate hash values for case fold table")
	}
	return int(bestIdx), bestSeed
}

func genCaseFolds(w *bytes.Buffer) {
	log.Println("Generating values for _CaseFolds table (this may take a long time)...")
	inputs := make([]uint32, len(caseFolds))
	for i, p := range caseFolds {
		inputs[i] = p.From
	}
	start := time.Now()
	maxIdx, seed := genCaseFoldHashValues(inputs)
	if maxIdx < caseFoldSize/2 {
		// Error if the algorithm found a table size that is a smaller power of 2
		log.Panicf("Hash table size can be reduced to %d or less...", caseFoldSize/2)
	}
	log.Printf("Successfully generated _CaseFolds values in: %s", time.Since(start))
	log.Printf("    max_index: %d", maxIdx)
	log.Printf("    seed:      %d", seed)

	// TODO: probably don't need this
	pairs := make([]foldPair, len(caseFolds))
	copy(pairs, caseFolds)
	slices.SortFunc(pairs, func(a, b foldPair) bool {
		return a.From < b.From
	})

	hashes := make([]foldPair, 0, len(pairs))
	for i, p := range pairs {
		hashes = append(hashes, foldPair{
			From: hash(p.From, seed),
			To:   uint32(i),
		})
	}
	slices.SortFunc(hashes, func(a, b foldPair) bool {
		return a.From < b.From
	})

	// TODO: add a comment?
	fmt.Fprint(w, "\n")
	fmt.Fprintf(w, "const _CaseFoldsSeed = 0x%04X\n", seed)
	fmt.Fprintf(w, "const _CaseFoldsShift = 0x%04X\n", caseFoldShift)
	fmt.Fprint(w, "\n")
	fmt.Fprintf(w, "var _CaseFolds = [%d]foldPair{\n", caseFoldSize)
	for _, h := range hashes {
		p := pairs[h.To]
		fmt.Fprintf(w, "\t%d: {0x%04X, 0x%04X}, // %q => %q\n", h.From, p.From, p.To, p.From, p.To)
	}
	fmt.Fprint(w, "}\n\n")
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

func testBuild(data []byte, skipTests bool) {
	dir, err := os.MkdirTemp("", "strcase.*")
	if err != nil {
		log.Panic(err)
	}

	tables := filepath.Join(dir, "tables.go")
	overlay := filepath.Join(dir, "overlay.json")

	type overlayJSON struct {
		Replace map[string]string
	}

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
	if !skipTests {
		runCommand("test", "-overlay="+overlay)
	}

	os.RemoveAll(dir) // Only remove temp dir if successful
}

func dataEqual(filename string, data []byte) bool {
	got, err := os.ReadFile(filename)
	return err == nil && bytes.Equal(got, data)
}

func writeFile(name string, data []byte) {
	if dataEqual(name, data) {
		return
	}

	var tmp string
	for i := 0; ; i++ {
		tmp = fmt.Sprintf("temp.%s.%d", name, time.Now().UnixNano())
		if _, err := os.Lstat(tmp); os.IsNotExist(err) {
			break
		}
		if i >= 1_000 {
			log.Fatalf("failed to generate tempory file for: %q", name)
		}
	}
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		log.Fatal(err)
	}
	if err := os.Rename(tmp, name); err != nil {
		os.Remove(tmp)
		log.Fatal(err)
	}
}

func writeGo(w *bytes.Buffer) {
	data := make([]byte, w.Len())
	copy(data, w.Bytes())
	src, err := format.Source(data)
	if err != nil {
		log.Panic(err)
	}
	w.Reset()
	if _, err := gen.WriteGo(w, "strcase", "", src); err != nil {
		log.Fatal(err)
	}
}

func hashCaseFolds() string {
	h := sha256.New()
	b := make([]byte, 8)
	for _, p := range caseFolds {
		binary.LittleEndian.PutUint32(b[0:4], p.From)
		binary.LittleEndian.PutUint32(b[4:8], p.To)
		h.Write(b)
	}
	return fmt.Sprintf("%x", h.Sum(nil))
}

func hashGenGoFile() string {
	f, err := os.Open("gen.go")
	if err != nil {
		log.Panic(err)
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		log.Panic(err)
	}
	return fmt.Sprintf("%x", h.Sum(nil))
}

func writeUnicodeConstants() {
	var w bytes.Buffer
	gen.WriteUnicodeVersion(&w)
	gen.WriteCLDRVersion(&w)
	writeGo(&w)
}

var tableInfo struct {
	UnicodeVersion string `json:"unicode_version"`
	CLDRVersion    string `json:"cldr_version"`
	CaseFoldHash   string `json:"case_fold_hash"`
	GenGoHash      string `json:"gen_go_hash"`
}

func loadTableInfo() {
	data, err := os.ReadFile(".tables.json")
	if err != nil {
		return
	}
	if err := json.Unmarshal(data, &tableInfo); err != nil {
		log.Panic(err)
	}
}

func updateTableInfoFile(fileHash, foldHash string) {
	tableInfo.UnicodeVersion = gen.UnicodeVersion()
	tableInfo.CLDRVersion = gen.CLDRVersion()
	tableInfo.CaseFoldHash = foldHash
	tableInfo.GenGoHash = fileHash
	data, err := json.MarshalIndent(&tableInfo, "", "    ")
	if err != nil {
		log.Panic(err)
	}
	writeFile(".tables.json", data)
}

func fileExists(name string) bool {
	_, err := os.Lstat(name)
	return err == nil
}

func main() {
	skipTests := flag.Bool("skip-tests", false, "skip running tests")
	dryRun := flag.Bool("dry-run", false,
		"report if generate would change tables.go and exit non-zero")

	gen.Init()               // TODO: we don't really need this
	log.SetOutput(os.Stdout) // use stdout instead of stderr

	loadTableInfo()
	loadCaseFolds()
	loadCategories()
	fileHash := hashGenGoFile()
	foldHash := hashCaseFolds()

	if fileExists("tables.go") &&
		gen.UnicodeVersion() == tableInfo.UnicodeVersion &&
		gen.CLDRVersion() == tableInfo.CLDRVersion &&
		foldHash == tableInfo.CaseFoldHash &&
		fileHash == tableInfo.GenGoHash {

		log.Printf("gen: exiting - no changes:\n"+
			"    unicode_version: %q\n"+
			"    cldr_version:    %q\n"+
			"    case_fold_hash:  %q\n"+
			"    gen_go_hash:     %q\n",
			tableInfo.UnicodeVersion, tableInfo.CLDRVersion,
			tableInfo.CaseFoldHash, tableInfo.GenGoHash)
		return
	}

	var w bytes.Buffer
	gen.WriteUnicodeVersion(&w)
	gen.WriteCLDRVersion(&w)

	genFoldMap(&w)
	genCaseFolds(&w)

	writeGo(&w)
	testBuild(w.Bytes(), *skipTests)

	// For dry runs only report if tables.go would be changed and
	// exit with an error if so.
	if *dryRun {
		if !dataEqual("tables.go", w.Bytes()) {
			fmt.Println("gen: would change tables.go")
			os.Exit(1)
		}
		return
	}

	writeFile("tables.go", w.Bytes())
	updateTableInfoFile(fileHash, foldHash)
	log.Printf("Successfully generated tables:\n"+
		"    unicode_version: %q\n"+
		"    cldr_version:    %q\n"+
		"    case_fold_hash:  %q\n"+
		"    gen_go_hash:     %q\n",
		tableInfo.UnicodeVersion, tableInfo.CLDRVersion,
		tableInfo.CaseFoldHash, tableInfo.GenGoHash)
}
