// Copyright 2023 Charlie Vieth. All rights reserved.
// Use of this source code is governed by the MIT license.

// gen generates the Unicode lookup tables used by strcase. The tables must
// be regenerated if this code is changed (`go generate`).
package main

import (
	"bytes"
	"cmp"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"go/format"
	"io/fs"
	"log"
	"math"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"runtime/pprof"
	"slices"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unicode"

	"github.com/schollz/progressbar/v3"
	"golang.org/x/term"
	"golang.org/x/text/unicode/rangetable"

	"github.com/charlievieth/strcase/internal/gen/gen"
	"github.com/charlievieth/strcase/internal/gen/ucd"
	"github.com/charlievieth/strcase/internal/gen/util"
)

func init() {
	initLogs()

	// If $GOMAXPROCS isn't set, use the full capacity of the machine.
	// For small machines, use at least 4 threads.
	if os.Getenv("GOMAXPROCS") == "" {
		n := runtime.NumCPU()
		if n < 4 {
			n = 4
		}
		runtime.GOMAXPROCS(n)
	}
}

func initLogs() {
	log.SetPrefix("")
	log.SetFlags(log.Lshortfile)
	log.SetOutput(os.Stdout) // use stdout instead of stderr
}

// WARN: we need to include 'İ' (0x0130) and 'ı' (0x0131) in _FoldMap because
// we don't want to fallback to using toUpperLower() since we don't accept the
// upper/lower-case variants of these runes (breaks simple folding semantics).
//
// We should remove these runes and any other runes in _FoldMap from _UpperLower
// and maybe remove 'İ' and 'ı' from _FoldMap as well.
//
// TODO: remove İ (0x0130) from _UpperLower and fix tests

// TODO: consider renaming the generated tables
const (
	MaxChar              = 0x10FFFF
	caseFoldShift        = 19
	caseFoldSize         = 8192
	foldMapShift         = 24
	foldMapSize          = 256
	upperLowerTableSize  = 8192
	upperLowerTableShift = 19
)

const (
	GenCookieKey   = "b3d53e54921be5a951563a7e77bfcb1a"
	GenCookieValue = "c1bbb5805a8c389e0dd50234d5c38cce"
)

type foldPair struct {
	From uint32
	To   uint32
}

var (
	categories *unicode.RangeTable
	caseFolds  []foldPair
	caseRanges []unicode.CaseRange // used by toLower and toUpper
	caseOrbit  []foldPair          // used by simpleFold
	asciiFold  [unicode.MaxASCII + 1]uint16
)

var (
	chars   = make([]Char, MaxChar+1)
	scripts = make(map[string][]rune)
	props   = make(map[string][]rune) // a property looks like a script; can share the format
)

// TODO: move
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
		chars[p1].foldCase = p2
	})
	slices.SortFunc(caseFolds, func(a, b foldPair) int {
		return cmp.Compare(a.From, b.From)
	})
}

var buildTags = map[string]struct{ version, buildTags, filename string }{
	"13.0.0": {"13.0.0", "go1.16,!go1.21", "tables_go116.go"},
	"15.0.0": {"15.0.0", "go1.21", "tables_go121.go"},
}

// tablesFileName is the names of the file to generate and is based off
// of the Go version this program is ran with.
//
// NB: this must be called after command line flags are parsed
func tablesFileName(dirname string) string {
	// gen.UnicodeVersion is set by the "-unicode" flag
	if name := buildTags[gen.UnicodeVersion()].filename; name != "" {
		return filepath.Join(dirname, name)
	}
	log.Panicf("unsupported Unicode version %q this script might need "+
		"to be updated", gen.UnicodeVersion())
	panic("unreachable")
}

type span struct {
	Start, End int64
}

func generateSpans(start, end, delta int64) []span {
	maxEnd := end
	var spans []span
	for i := start; i <= end; i += delta {
		end := i + delta
		if end >= maxEnd {
			end = maxEnd
		}
		spans = append(spans, span{Start: i, End: end})
	}

	// Reverse spans since larger values are more likely to be better seeds.
	// We previously randomly shuffled the spans, but led to non-deterministic
	// behavior when more than one seed was ideal.
	for i := len(spans)/2 - 1; i >= 0; i-- {
		opp := len(spans) - 1 - i
		spans[i], spans[opp] = spans[opp], spans[i]
	}

	return spans
}

func hash(seed, key, shift uint32) uint32 {
	m := seed * key
	return m >> shift
}

func shiftHash(seed, key, shift uint32) uint32 {
	key |= key << 24 // fill top bits not occupied by unicode.MaxRune
	m := seed * key
	return m >> shift
}

type HashConfig struct {
	TableName string
	TableSize int
	HashShift uint32 // TODO: this can be calculated from TableSize
	// TODO: name is confusing with HashShift
	ShiftHash bool // Use shiftHash instead of hash

	// Use the first valid hash value instead of performing an exhaustive
	// search to find the "best" hash seed. The seed is not deterministic
	// and this is only meant for testing since it is much much faster.
	FirstValidHash bool
}

var hashSeedCache = map[string]uint32{}

func cacheKey(inputs []uint32) string {
	if !slices.IsSorted(inputs) {
		slices.Sort(inputs)
	}
	b := make([]byte, len(inputs)*4)
	for i, u := range inputs {
		binary.LittleEndian.PutUint32(b[i*4:], u)
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

// TODO:
//   - Return the first working hash value.
//   - Find a better algorithm.
//
// GenerateHashValues performs a brute-force search for the best possible
// multiplicative hash seed for inputs. All uint32 values are checked.
func (conf *HashConfig) GenerateHashValues(inputs []uint32) (hashSeed uint32) {
	const delta = 512 * 1024

	if *useCachedSeeds {
		if seed, ok := tableInfo.TableHashes[conf.TableName]; ok {
			log.Printf("WARN: using cached seed %d for table: %s", seed, conf.TableName)
			return seed
		}
	}

	if seed, ok := hashSeedCache[cacheKey(inputs)]; ok {
		log.Printf("Using previously computed seed (0x%04X) for the %s table",
			seed, conf.TableName)
		tableInfo.TableHashes[conf.TableName] = seed
		return seed
	}

	if slices.IndexFunc(inputs, func(u uint32) bool { return u != 0 }) < 0 {
		log.Panicf("Input is all zeros for table: %s", conf.TableName)
	}

	log.Printf("Generating values for the %s table (this may take a long time)...\n",
		conf.TableName)

	// This can take awhile so use a progress bar.
	var bar *progressbar.ProgressBar
	if term.IsTerminal(int(os.Stdout.Fd())) {
		bar = progressbar.Default(math.MaxUint32)
	} else {
		bar = progressbar.DefaultSilent(math.MaxUint32)
	}
	start := time.Now()

	// TODO: set GOMAXPROCS to NumCPU ??
	numProcs := runtime.GOMAXPROCS(0)

	// When FirstValidHash we exit before consuming all spans.
	done := make(chan struct{})
	defer close(done)

	ch := make(chan *span, numProcs*2)
	go func() {
		spans := generateSpans(1, math.MaxUint32, delta)
		for i := range spans {
			select {
			case ch <- &spans[i]:
			case <-done:
				return
			}
		}
		close(ch)
	}()

	var (
		bestIndex = int64(math.MaxInt64)
		bestSeed  uint32
		foundSeed atomic.Bool
		mu        sync.Mutex
		wg        sync.WaitGroup
	)
	if seed := tableInfo.TableHashes[conf.TableName]; seed != 0 {
		bestSeed = seed
	}
	for i := 0; i < numProcs; i++ {
		wg.Add(1)
		go func(inputs []uint32) {
			defer wg.Done()
			seen := make([]bool, conf.TableSize)
			for sp := range ch {
				if foundSeed.Load() {
					return
				}
				best := atomic.LoadInt64(&bestIndex)
			Loop:
				for i := sp.Start; i <= sp.End && best >= int64(len(inputs)); i++ {
					for i := range seen {
						seen[i] = false // TODO: zero
					}
					// TODO: load more often
					if i%8192 == 0 {
						best = atomic.LoadInt64(&bestIndex)
					}
					var maxIdx int64
					shift := conf.HashShift
					seed := uint32(i)
					useShiftHash := conf.ShiftHash
					// TODO: is there anyway we can optimize this?
					for _, x := range inputs {
						var u int64
						if useShiftHash {
							u = int64(shiftHash(seed, x, shift))
						} else {
							u = int64(hash(seed, x, shift))
						}
						if u > best || seen[u] {
							continue Loop
						}
						seen[u] = true
						if u > maxIdx {
							maxIdx = u
						}
					}
					best = atomic.LoadInt64(&bestIndex)
					if maxIdx < best {
						// Use mutex to simplify updating both values
						mu.Lock()
						best = atomic.LoadInt64(&bestIndex) // re-check
						if maxIdx < best {
							atomic.StoreInt64(&bestIndex, maxIdx)
							atomic.StoreUint32(&bestSeed, seed)

							// Stop at the first valid has
							if conf.FirstValidHash && foundSeed.CompareAndSwap(false, true) {
								log.Println("WARN: -first-valid-hash flag provided: stopping early")
								mu.Unlock()
								return
							}
						}
						mu.Unlock()
					}
				}
				// TODO: figure out is we need to use `1 + sp.End - sp.Start`
				if err := bar.Add64(sp.End - sp.Start); err != nil {
					max := bar.GetMax64()
					log.Panicf("error updating progress bar: %v: max: %d delta: %d",
						err, max, 1+sp.End-sp.Start)
				}
			}
		}(inputs)
	}
	wg.Wait()
	bar.Close()

	// TODO: we can probably just check bestSeed
	if bestIndex == math.MaxInt64 || bestSeed == 0 {
		log.Panicf("Failed to generate hash values for %s table: max_index: %d seed: %d",
			conf.TableName, bestIndex, bestSeed)
	}
	if bestIndex <= int64(conf.TableSize/2) {
		// Error if the algorithm found a table size that is a smaller power of 2
		log.Panicf("The hash table size can be reduced to %d or less. The best index is: %d.",
			conf.TableSize/2, bestIndex)
	}

	log.Printf("Successfully generated %s values in: %s\n"+
		"    max_index: %d\n"+
		"    seed:      %d\n",
		conf.TableName, time.Since(start), bestIndex, bestSeed)
	// log.Printf("Successfully generated %s values in: %s", conf.TableName, time.Since(start))
	// log.Printf("    max_index: %d", bestIndex)
	// log.Printf("    seed:      %d", bestSeed)

	hashSeedCache[cacheKey(inputs)] = bestSeed // Cache result
	tableInfo.TableHashes[conf.TableName] = bestSeed
	return bestSeed
}

const (
	CaseUpper = 1 << iota
	CaseLower
	CaseTitle
	CaseNone    = 0  // must be zero
	CaseMissing = -1 // character not present; not a valid case state
)

type caseState struct {
	point        rune
	_case        int
	deltaToUpper rune
	deltaToLower rune
	deltaToTitle rune
}

// Is d a continuation of the state of c?
func (c *caseState) adjacent(d *caseState) bool {
	if d.point < c.point {
		c, d = d, c
	}
	switch {
	case d.point != c.point+1: // code points not adjacent (shouldn't happen)
		return false
	case d._case != c._case: // different cases
		return c.upperLowerAdjacent(d)
	case c._case == CaseNone:
		return false
	case c._case == CaseMissing:
		return false
	case d.deltaToUpper != c.deltaToUpper:
		return false
	case d.deltaToLower != c.deltaToLower:
		return false
	case d.deltaToTitle != c.deltaToTitle:
		return false
	}
	return true
}

// Is d the same as c, but opposite in upper/lower case? this would make it
// an element of an UpperLower sequence.
func (c *caseState) upperLowerAdjacent(d *caseState) bool {
	// check they're a matched case pair.  we know they have adjacent values
	switch {
	case c._case == CaseUpper && d._case != CaseLower:
		return false
	case c._case == CaseLower && d._case != CaseUpper:
		return false
	}
	// matched pair (at least in upper/lower).  make the order Upper Lower
	if c._case == CaseLower {
		c, d = d, c
	}
	// for an Upper Lower sequence the deltas have to be in order
	//	c: 0 1 0
	//	d: -1 0 -1
	switch {
	case c.deltaToUpper != 0:
		return false
	case c.deltaToLower != 1:
		return false
	case c.deltaToTitle != 0:
		return false
	case d.deltaToUpper != -1:
		return false
	case d.deltaToLower != 0:
		return false
	case d.deltaToTitle != -1:
		return false
	}
	return true
}

// Does this character start an UpperLower sequence?
func (c *caseState) isUpperLower() bool {
	// for an Upper Lower sequence the deltas have to be in order
	//	c: 0 1 0
	switch {
	case c.deltaToUpper != 0:
		return false
	case c.deltaToLower != 1:
		return false
	case c.deltaToTitle != 0:
		return false
	}
	return true
}

// Does this character start a LowerUpper sequence?
func (c *caseState) isLowerUpper() bool {
	// for an Upper Lower sequence the deltas have to be in order
	//	c: -1 0 -1
	switch {
	case c.deltaToUpper != -1:
		return false
	case c.deltaToLower != 0:
		return false
	case c.deltaToTitle != -1:
		return false
	}
	return true
}

func getCaseState(i rune) (c *caseState) {
	c = &caseState{point: i, _case: CaseNone}
	ch := &chars[i]
	switch ch.codePoint {
	case 0:
		c._case = CaseMissing // Will get NUL wrong but that doesn't matter
		return
	case ch.upperCase:
		c._case = CaseUpper
	case ch.lowerCase:
		c._case = CaseLower
	case ch.titleCase:
		c._case = CaseTitle
	}
	// Some things such as roman numeral U+2161 don't describe themselves
	// as upper case, but have a lower case. Second-guess them.
	if c._case == CaseNone && ch.lowerCase != 0 {
		c._case = CaseUpper
	}
	// Same in the other direction.
	if c._case == CaseNone && ch.upperCase != 0 {
		c._case = CaseLower
	}

	if ch.upperCase != 0 {
		c.deltaToUpper = ch.upperCase - i
	}
	if ch.lowerCase != 0 {
		c.deltaToLower = ch.lowerCase - i
	}
	if ch.titleCase != 0 {
		c.deltaToTitle = ch.titleCase - i
	}
	return
}

// TODO: we need to do this since we can't use the [unicode] package due to
// a version mismatch between the Unicode version we're generating for and
// Unicode version of Go version being used to generate this.
//
// TODO: fixup the above comment.
func generateCaseRanges() []unicode.CaseRange {
	if gen.UnicodeVersion() == unicode.Version {
		return unicode.CaseRanges
	}
	// The requested Unicode version does not match the stdlibs
	var (
		cases      []unicode.CaseRange
		startState *caseState     // the start of a run; nil for not active
		prevState  = &caseState{} // the state of the previous character
	)
	for i := range chars {
		state := getCaseState(rune(i))
		if state.adjacent(prevState) {
			prevState = state
			continue
		}
		// end of run (possibly)
		if c, ok := getCaseRange(startState, prevState); ok {
			cases = append(cases, c)
		}
		// printCaseRange(startState, prevState)
		startState = nil
		if state._case != CaseMissing && state._case != CaseNone {
			startState = state
		}
		prevState = state
	}

	return cases
}

// Modified version of golang.org/x/text/internal/export/unicode.printCaseRange
func getCaseRange(lo, hi *caseState) (unicode.CaseRange, bool) {
	if lo == nil {
		return unicode.CaseRange{}, false
	}
	if lo.deltaToUpper == 0 && lo.deltaToLower == 0 && lo.deltaToTitle == 0 {
		// character represents itself in all cases - no need to mention it
		return unicode.CaseRange{}, false
	}
	switch {
	case hi.point > lo.point && lo.isUpperLower():
		c := unicode.CaseRange{
			Lo: uint32(lo.point),
			Hi: uint32(hi.point),
			Delta: [unicode.MaxCase]rune{
				unicode.UpperLower,
				unicode.UpperLower,
				unicode.UpperLower,
			},
		}
		return c, true
	case hi.point > lo.point && lo.isLowerUpper():
		log.Panicf("LowerUpper sequence: should not happen: %U. "+
			"If it's real, need to fix To()", lo.point)
	default:
		c := unicode.CaseRange{
			Lo: uint32(lo.point),
			Hi: uint32(hi.point),
			Delta: [unicode.MaxCase]rune{
				lo.deltaToUpper,
				lo.deltaToLower,
				lo.deltaToTitle,
			},
		}
		return c, true
	}
	return unicode.CaseRange{}, false
}

// simpleFold is the same as unicode.SimpleFold but uses the version of Unicode
// we loaded.
func simpleFold(r rune) rune {
	if r < 0 || r > unicode.MaxRune {
		return r
	}

	if int(r) < len(asciiFold) {
		return rune(asciiFold[r])
	}

	// Consult caseOrbit table for special cases.
	lo := 0
	hi := len(caseOrbit)
	for lo < hi {
		m := lo + (hi-lo)/2
		if rune(caseOrbit[m].From) < r {
			lo = m + 1
		} else {
			hi = m
		}
	}
	if lo < len(caseOrbit) && rune(caseOrbit[lo].From) == r {
		return rune(caseOrbit[lo].To)
	}

	// No folding specified. This is a one- or two-element
	// equivalence class containing rune and ToLower(rune)
	// and ToUpper(rune) if they are different from rune.
	if l := toLower(r); l != r {
		return l
	}
	return toUpper(r)
}

// to maps the rune using the specified case mapping.
// It additionally reports whether caseRange contained a mapping for r.
func to(_case int, r rune, caseRange []unicode.CaseRange) (mappedRune rune) {
	if len(caseRange) == 0 {
		panic("empty caseRange")
	}
	if _case < 0 || unicode.MaxCase <= _case {
		return unicode.ReplacementChar // as reasonable an error as any
	}
	// binary search over ranges
	lo := 0
	hi := len(caseRange)
	for lo < hi {
		m := lo + (hi-lo)/2
		cr := caseRange[m]
		if rune(cr.Lo) <= r && r <= rune(cr.Hi) {
			delta := cr.Delta[_case]
			if delta > unicode.MaxRune {
				// In an Upper-Lower sequence, which always starts with
				// an UpperCase letter, the real deltas always look like:
				//	{0, 1, 0}    UpperCase (Lower is next)
				//	{-1, 0, -1}  LowerCase (Upper, Title are previous)
				// The characters at even offsets from the beginning of the
				// sequence are upper case; the ones at odd offsets are lower.
				// The correct mapping can be done by clearing or setting the low
				// bit in the sequence offset.
				// The constants UpperCase and TitleCase are even while LowerCase
				// is odd so we take the low bit from _case.
				return rune(cr.Lo) + ((r-rune(cr.Lo))&^1 | rune(_case&1))
			}
			return r + delta
		}
		if r < rune(cr.Lo) {
			hi = m
		} else {
			lo = m + 1
		}
	}
	return r
}

// toUpper is the same as unicode.ToUpper but uses the Unicode table we loaded.
func toUpper(r rune) rune {
	if r <= unicode.MaxASCII {
		if 'a' <= r && r <= 'z' {
			r -= 'a' - 'A'
		}
		return r
	}
	return to(unicode.UpperCase, r, caseRanges)
}

// toLower is the same as unicode.ToLower but uses the Unicode table we loaded.
func toLower(r rune) rune {
	if r <= unicode.MaxASCII {
		if 'A' <= r && r <= 'Z' {
			r += 'a' - 'A'
		}
		return r
	}
	return to(unicode.LowerCase, r, caseRanges)
}

func folds(sr rune) []rune {
	r := simpleFold(sr)
	runes := make([]rune, 1, 2)
	runes[0] = sr
	for r != sr {
		runes = append(runes, r)
		r = simpleFold(r)
	}
	return runes
}

func genCaseFolds(w *bytes.Buffer, firstValidHash bool) {
	folds := caseFolds
	inputs := make([]uint32, len(folds))
	for i, p := range folds {
		inputs[i] = p.From
	}

	conf := HashConfig{
		TableName:      "CaseFolds",
		TableSize:      caseFoldSize,
		HashShift:      caseFoldShift,
		FirstValidHash: firstValidHash,
	}
	seed := conf.GenerateHashValues(inputs)

	// TODO: probably don't need this
	pairs := make([]foldPair, len(folds))
	copy(pairs, folds)
	slices.SortFunc(pairs, func(a, b foldPair) int {
		return cmp.Compare(a.From, b.From)
	})

	hashes := make([]foldPair, 0, len(pairs))
	for i, p := range pairs {
		hashes = append(hashes, foldPair{
			From: hash(p.From, seed, caseFoldShift),
			To:   uint32(i),
		})
	}
	slices.SortFunc(hashes, func(a, b foldPair) int {
		return cmp.Compare(a.From, b.From)
	})

	fmt.Fprint(w, "\n")
	fmt.Fprintf(w, "const _CaseFoldsSeed = 0x%04X\n", seed)
	fmt.Fprintf(w, "const _CaseFoldsShift = %d\n", caseFoldShift)
	fmt.Fprint(w, "\n")
	fmt.Fprintln(w, "// _CaseFolds stores all Unicode simple case-folds.")
	fmt.Fprintf(w, "var _CaseFolds = [%d]foldPair{\n", caseFoldSize)
	for _, h := range hashes {
		p := pairs[h.To]
		fmt.Fprintf(w, "\t%d: {0x%04X, 0x%04X}, // %q => %q\n", h.From, p.From, p.To, p.From, p.To)
	}
	fmt.Fprint(w, "}\n\n")
}

func dedupe(r []rune) []rune {
	if len(r) < 2 {
		return r
	}
	slices.Sort(r)
	return slices.Compact(r)
}

// TODO: consider renaming this table
func genFoldTable(w *bytes.Buffer, firstValidHash bool) {
	runes := make(map[rune][]rune)
	rangetable.Visit(categories, func(r rune) {
		ff := folds(r)
		if len(ff) > 2 {
			runes[r] = append(runes[r], ff...)
		}
		if len(ff) == 1 && toUpper(r) != toLower(r) {
			runes[r] = append(runes[r], ff...)
		}
	})
	// FIXME: fix the below since we have to work around it in the code
	// WARN: we should not need to add this manually
	runes['İ'] = append(runes['İ'], 'İ')
	runes['ß'] = append(runes['ß'], 'ẞ')

	keys := make([]uint32, 0, len(runes))
	for k, rs := range runes {
		// Make sure the key is included (was an issue with: 'ß')
		if !slices.Contains(rs, k) {
			rs = append(rs, k)
		}
		runes[k] = dedupe(rs)
		keys = append(keys, uint32(k))
	}

	conf := HashConfig{
		TableName:      "FoldMap",
		TableSize:      foldMapSize,
		HashShift:      foldMapShift,
		FirstValidHash: firstValidHash,
	}
	seed := conf.GenerateHashValues(keys)

	// Make key the first element of the rune slice
	folds := make([][]rune, 0, len(runes))
	for k, rs := range runes {
		if rs[0] != k {
			a := []rune{k}
			for _, r := range rs {
				if r != k {
					a = append(a, r)
				}
			}
			rs = a
		}
		folds = append(folds, rs)
	}
	slices.SortFunc(folds, func(f1, f2 []rune) int {
		return cmp.Compare(f1[0], f2[0])
	})

	fmt.Fprint(w, "\n")
	fmt.Fprintf(w, "const _FoldMapSeed = 0x%04X\n", seed)
	fmt.Fprintf(w, "const _FoldMapShift = %d\n", foldMapShift)
	fmt.Fprint(w, "\n")
	fmt.Fprintln(w, "// FoldMap stores the Unicode case-folds for characters "+
		"that have two or more folds.")
	fmt.Fprintf(w, "var _FoldMap = [%d][4]uint16{\n", foldMapSize)
	for _, ff := range folds {
		fmt.Fprintf(w, "\t%d: {0x%04X", hash(uint32(ff[0]), seed, foldMapShift), ff[0])
		for _, f := range ff[1:] {
			fmt.Fprintf(w, ", 0x%04X", f)
		}
		fmt.Fprintf(w, "}, // %q\n", ff)
	}
	fmt.Fprint(w, "}\n\n")

	type runeSet struct {
		r uint32
		a [2]rune
	}

	var noUpperLower []runeSet
	for k, rs := range runes {
		u := toUpper(k)
		l := toLower(k)
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
		slices.Sort(a)
		noUpperLower = append(noUpperLower, runeSet{uint32(k), [2]rune{a[0], a[1]}})
	}
	slices.SortFunc(noUpperLower, func(c1, c2 runeSet) int {
		return cmp.Compare(c1.r, c2.r)
	})

	const foldMapExcludingUpperLowerComment = `
// _FoldMapExcludingUpperLower stores the Unicode case-folds for charactecrs that
// have two or more folds, but excludes the uppercase and lowercase forms of the
// character.`

	fmt.Fprintln(w, "")
	fmt.Fprintf(w, "const _FoldMapExcludingUpperLowerSeed = 0x%04X\n", seed)
	fmt.Fprintf(w, "const _FoldMapExcludingUpperLowerShift = %d\n", foldMapShift)
	fmt.Fprintln(w, "")

	fmt.Fprintln(w, foldMapExcludingUpperLowerComment)
	fmt.Fprintf(w, "var _FoldMapExcludingUpperLower = [%d]struct {\n", foldMapSize)
	fmt.Fprintln(w, "\tr uint16")
	fmt.Fprintln(w, "\ta [2]uint16")
	fmt.Fprintln(w, "}{")
	for _, c := range noUpperLower {
		h := hash(c.r, seed, foldMapShift)
		if c.a[0] > math.MaxUint16 {
			log.Fatalf("rune 0x%04X is larger than MaxUint16 0x%04X", c.a[0], math.MaxUint16)
		}
		if c.a[1] > math.MaxUint16 {
			log.Fatalf("rune 0x%04X is larger than MaxUint16 0x%04X", c.a[1], math.MaxUint16)
		}
		fmt.Fprintf(w, "\t%d: {0x%04X, [2]uint16{0x%04X, 0x%04X}}, // %q: [%q, %q]\n",
			h, c.r, c.a[0], c.a[1], c.r, c.a[0], c.a[1])
	}
	fmt.Fprint(w, "}\n\n")
}

func genUpperLowerTable(w *bytes.Buffer, firstValidHash bool) {
	// WARN: attempt to use caseOrbit so that we don't have to handle special
	// cases with toUpperLowerSpecial.

	const docComment = `
// _UpperLower stores upper/lower case pairs of Unicode code points.
// This takes up more space than the stdlib's "unicode" package, but
// is roughly ~4x faster.`

	type Case struct {
		Rune  rune `json:"rune"`
		Upper rune `json:"upper"`
		Lower rune `json:"lower"`
	}
	var cases []Case

	// special cases where Rune != Upper or Lower
	var special []Case

	for r := rune('A'); r <= unicode.MaxRune; r++ {
		if r <= unicode.MaxASCII {
			continue
		}
		l := toLower(r)
		u := toUpper(r)
		if r != l || r != u {
			if r == l || r == u {
				cases = append(cases, Case{Rune: r, Upper: u, Lower: l})
			} else {
				special = append(special, Case{Rune: r, Upper: u, Lower: l})
			}
		}
	}

	keys := make([]uint32, len(cases))
	for i, c := range cases {
		keys[i] = uint32(c.Rune)
	}
	// TODO: this is probably not necessary
	slices.Sort(keys)
	keys = slices.Compact(keys)

	conf := HashConfig{
		TableName:      "UpperLower",
		TableSize:      upperLowerTableSize,
		HashShift:      upperLowerTableShift,
		ShiftHash:      true,
		FirstValidHash: firstValidHash,
	}
	seed := conf.GenerateHashValues(keys)

	fmt.Fprint(w, "\n")
	fmt.Fprintf(w, "const _UpperLowerSeed = 0x%04X\n", seed)
	fmt.Fprintf(w, "const _UpperLowerShift = %d\n", upperLowerTableShift)
	fmt.Fprint(w, "\n")
	fmt.Fprintln(w, strings.TrimSpace(docComment))
	fmt.Fprintf(w, "var _UpperLower = [%d][2]uint32{\n", upperLowerTableSize)
	for _, c := range cases {
		fmt.Fprintf(w, "\t%d: {0x%04X, 0x%04X}, // %q => %q\n",
			shiftHash(seed, uint32(c.Rune), upperLowerTableShift), c.Upper, c.Lower, c.Upper, c.Lower)
	}
	fmt.Fprint(w, "}\n\n")

	slices.SortFunc(special, func(c1, c2 Case) int {
		return cmp.Compare(c1.Rune, c2.Rune)
	})

	fmt.Fprintln(w, `
// toUpperLowerSpecial returns the uppercase and lowercase form of r,
// which is a character that is not equal to either its uppercase or
// lowercase form and thus cannot be mapped into the _UpperLower table.
func toUpperLowerSpecial(r rune) (rune, rune, bool) {
	switch r {`)
	for _, c := range special {
		fmt.Fprintf(w, "\tcase %q:\n", c.Rune)
		fmt.Fprintf(w, "\t\treturn %q, %q, %t\n", c.Upper, c.Lower, true)
	}
	fmt.Fprintln(w, "\t}")
	fmt.Fprintln(w, "\treturn r, r, false")
	fmt.Fprintln(w, "}")
}

func writeInitGuard(w *bytes.Buffer) {
	const s = `

func init() {
	// This is essentially a compile time assertion that can only fail if a
	// future Go release updates the version of Unicode it supports. This
	// check is elided if the Unicode versions match.
	//
	// TLDR if you see this panic file an issue:
	// https://github.com/charlievieth/strcase/issues
	//
	if UnicodeVersion != unicode.Version {
		panic("strcase.UnicodeVersion \"" + UnicodeVersion +
			"\" != unicode.Version \"" + unicode.Version + "\"")
	}
}

`
	w.WriteString(s)
}

func writeTypes(w *bytes.Buffer) {
	const s = `

// A foldPair stores Unicode case folding pairs
type foldPair struct {
	From uint32
	To   uint32
}

`
	w.WriteString(s)
}

func writeFunctions(w *bytes.Buffer) {
	const s = `

// TODO: rename to "foldCase"
//
// CaseFold returns the Unicode simple case-fold for r, if one exists, or r
// unmodified, if one does not exist.
func CaseFold(r rune) rune {
	// TODO: check if r is ASCII here?
	u := uint32(r)
	h := (u * _CaseFoldsSeed) >> _CaseFoldsShift
	p := _CaseFolds[h]
	if p.From == u {
		r = rune(p.To)
	}
	return r
}

// TODO: rename
func FoldMap(r rune) *[4]uint16 {
	u := uint32(r)
	h := (u * _FoldMapSeed) >> _FoldMapShift
	p := &_FoldMap[h]
	if uint32(p[0]) == u {
		return p
	}
	return nil
}

func FoldMapExcludingUpperLower(r rune) [2]rune {
	u := uint32(r)
	h := (u * _FoldMapSeed) >> _FoldMapShift
	p := &_FoldMapExcludingUpperLower[h]
	if uint32(p.r) == u {
		return [2]rune{rune(p.a[0]), rune(p.a[1])}
	}
	return [2]rune{}
}

// ToUpperLower combines unicode.ToUpper and unicode.ToLower in one function.
func ToUpperLower(r rune) (upper, lower rune, foundMapping bool) {
	if r <= 0x80 {
		if 'A' <= r && r <= 'Z' {
			return r, r + ('a' - 'A'), true
		}
		if 'a' <= r && r <= 'z' {
			return r - ('a' - 'A'), r, true
		}
		return r, r, false
	}
	// Hash rune r and see if it's in the _UpperLower table.
	u := uint32(r)
	h := (u | u<<24) * _UpperLowerSeed
	p := &_UpperLower[h>>_UpperLowerShift]
	if p[0] == u || p[1] == u {
		return rune(p[0]), rune(p[1]), true
	}
	// Handle Unicode characters that do not equal
	// their upper and lower case forms.
	return toUpperLowerSpecial(r)
}

`
	w.WriteString(s)
}

func runCommand(dir string, args ...string) {
	cmd := exec.Command("go", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("Error:   %v", err)
		log.Printf("Command: %s", strings.Join(cmd.Args, " "))
		log.Printf("Output:  %s", bytes.TrimSpace(out))
		log.Panicf("Failed to build generated file: %v\n", err)
	}
}

func testBuild(tablesFile string, data []byte, skipTests bool) {
	dir, err := os.MkdirTemp("", "strcase.*")
	if err != nil {
		log.Panic(err)
	}

	tables := filepath.Join(dir, filepath.Base(tablesFile))
	overlay := filepath.Join(dir, "overlay.json")

	type overlayJSON struct {
		Replace map[string]string
	}

	overlayData, err := json.Marshal(overlayJSON{
		Replace: map[string]string{
			// filepath.Base(tablesFile): tables,
			tablesFile: tables,
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

	cmdDir := filepath.Dir(tablesFile)
	runCommand(cmdDir, "build", "-overlay="+overlay, "./...")
	if !skipTests {
		runCommand(cmdDir, "test", "-overlay="+overlay, "./...")
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

	f, err := os.CreateTemp(filepath.Dir(name), filepath.Base(name)+".tmp.*")
	if err != nil {
		log.Fatal(err)
	}
	tmp := f.Name()
	exit := func(err error) {
		os.Remove(tmp)
		log.Panic(err)
	}
	if _, err := f.Write(data); err != nil {
		exit(err)
	}
	if err := f.Chmod(0644); err != nil {
		exit(err)
	}
	if err := f.Close(); err != nil {
		exit(err)
	}
	if err := os.Rename(tmp, name); err != nil {
		exit(err)
	}
}

func writeTemp(name string, b []byte) {
	dir, err := os.MkdirTemp("", "strcase.gen.*")
	if err != nil {
		log.Panic(err)
	}
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, b, 0644); err != nil {
		log.Panic(err)
	}
	log.Println("TMPFILE:", path)
}

// TODO: use gen.CodeWriter
//
// writeGo formats the Go source in w and re-writes it back to w.
func writeGo(w *bytes.Buffer, tablesFile, buildTags string) {
	data := make([]byte, w.Len())
	copy(data, w.Bytes())
	src, err := format.Source(data)
	if err != nil {
		writeTemp(tablesFile, data)
		log.Panic(err)
	}
	w.Reset()
	if _, err := gen.WriteGo(w, "tables", buildTags, src); err != nil {
		log.Fatal(err)
	}
}

func hashCaseFolds() string {
	b := make([]byte, 0, 4096)
	for _, p := range caseFolds {
		b = binary.LittleEndian.AppendUint32(b, p.From)
		b = binary.LittleEndian.AppendUint32(b, p.To)
	}
	return fmt.Sprintf("%x", sha256.Sum256(b))
}

type TableInfo struct {
	Filename       string            `json:"filename"`
	UnicodeVersion string            `json:"unicode_version"`
	CLDRVersion    string            `json:"cldr_version"`
	CaseFoldHash   string            `json:"case_fold_hash"`
	GenGoHash      string            `json:"gen_go_hash"`
	TableHashes    map[string]uint32 `json:"table_hashes"`
}

var tableInfo = TableInfo{
	TableHashes: map[string]uint32{},
}

func readTableInfo(root, tablesFile string) (map[string]TableInfo, error) {
	m := make(map[string]TableInfo)
	// The ".tables.json" file is adjacent to the generated tables file.
	data, err := os.ReadFile(filepath.Join(root, ".tables.json"))
	if err != nil {
		return m, err
	}
	if err := json.Unmarshal(data, &m); err != nil {
		return m, err
	}
	return m, nil
}

// TODO: change this to use the Unicode version instead of the file name
//
// Note: loadTableInfo may be an absolute path.
func loadTableInfo(root, tablesFile string) {
	m, err := readTableInfo(root, tablesFile)
	if err != nil {
		if os.IsNotExist(err) {
			return
		}
		log.Panic(err)
	}
	base := filepath.Base(tablesFile)
	tableInfo = m[base]
	// TODO: do we need this intermediary step ??
	if tableInfo.Filename == "" {
		tableInfo.Filename = base
	}
	if tableInfo.TableHashes == nil {
		tableInfo.TableHashes = make(map[string]uint32)
	}
}

func updateTableInfoFile(root, tablesFile, fileHash, foldHash string) {
	base := filepath.Base(tablesFile)

	tableInfo.Filename = base
	tableInfo.UnicodeVersion = gen.UnicodeVersion()
	tableInfo.CLDRVersion = gen.CLDRVersion()
	tableInfo.CaseFoldHash = foldHash
	tableInfo.GenGoHash = fileHash

	m, err := readTableInfo(root, tablesFile)
	if err != nil && !os.IsNotExist(err) {
		log.Panic(err)
	}
	m[base] = tableInfo

	data, err := json.MarshalIndent(m, "", "    ")
	if err != nil {
		log.Panic(err)
	}
	writeFile(filepath.Join(root, ".tables.json"), data)
}

func fileExists(name string) bool {
	_, err := os.Lstat(name)
	return err == nil
}

var category = map[string]bool{
	// Nd Lu etc.
	// We use one-character names to identify merged categories
	"L": true, // Lu Ll Lt Lm Lo
	"P": true, // Pc Pd Ps Pe Pu Pf Po
	"M": true, // Mn Mc Me
	"N": true, // Nd Nl No
	"S": true, // Sm Sc Sk So
	"Z": true, // Zs Zl Zp
	"C": true, // Cc Cf Cs Co Cn
}

// This contains only the properties we're interested in.
type Char struct {
	codePoint rune // if zero, this index is not a valid code point.
	category  string
	upperCase rune
	lowerCase rune
	titleCase rune
	foldCase  rune // simple case folding
	caseOrbit rune // next in simple case folding orbit
}

func allCategories() []string {
	a := make([]string, 0, len(category))
	for k := range category {
		a = append(a, k)
	}
	sort.Strings(a)
	return a
}

func allCatFold(m map[string]map[rune]bool) []string {
	a := make([]string, 0, len(m))
	for k := range m {
		a = append(a, k)
	}
	slices.Sort(a)
	return a
}

// WARN WARN WARN: move this
func loadChars() {
	ucd.Parse(gen.OpenUCDFile("UnicodeData.txt"), func(p *ucd.Parser) {
		c := Char{codePoint: p.Rune(0)}

		getRune := func(field int) rune {
			if p.String(field) == "" {
				return 0
			}
			return p.Rune(field)
		}

		c.category = p.String(ucd.GeneralCategory)
		category[c.category] = true
		switch c.category {
		case "Nd":
			// Decimal digit
			p.Int(ucd.NumericValue)
		case "Lu":
			c.upperCase = getRune(ucd.CodePoint)
			c.lowerCase = getRune(ucd.SimpleLowercaseMapping)
			c.titleCase = getRune(ucd.SimpleTitlecaseMapping)
		case "Ll":
			c.upperCase = getRune(ucd.SimpleUppercaseMapping)
			c.lowerCase = getRune(ucd.CodePoint)
			c.titleCase = getRune(ucd.SimpleTitlecaseMapping)
		case "Lt":
			c.upperCase = getRune(ucd.SimpleUppercaseMapping)
			c.lowerCase = getRune(ucd.SimpleLowercaseMapping)
			c.titleCase = getRune(ucd.CodePoint)
		default:
			c.upperCase = getRune(ucd.SimpleUppercaseMapping)
			c.lowerCase = getRune(ucd.SimpleLowercaseMapping)
			c.titleCase = getRune(ucd.SimpleTitlecaseMapping)
		}

		chars[c.codePoint] = c
	})
}

// WARN: rename and fix other loadCategories()
func loadCategoryTables() map[string]*unicode.RangeTable {
	categoryOp := func(code rune, class uint8) bool {
		category := chars[code].category
		return len(category) > 0 && category[0] == class
	}

	list := allCategories()
	cats := make(map[string]*unicode.RangeTable, len(list))

	// Avoid calculating these tables if possible
	if gen.UnicodeVersion() == unicode.Version {
		for _, name := range list {
			if _, ok := unicode.Categories[name]; !ok {
				log.Fatal("unknown category", name)
			}
			cats[name] = unicode.Categories[name]
		}
		return cats
	}

	for _, name := range list {
		if _, ok := category[name]; !ok {
			log.Fatal("unknown category", name)
		}
		var rt *unicode.RangeTable
		if len(name) == 1 { // unified categories
			rt = dumpRange(func(code rune) bool { return categoryOp(code, name[0]) })
		} else {
			rt = dumpRange(func(code rune) bool { return chars[code].category == name })
		}
		cats[name] = rt
	}
	return cats
}

type Op func(code rune) bool

// TODO: rename
func dumpRange(inCategory Op) *unicode.RangeTable {
	runes := []rune{}
	for i := range chars {
		r := rune(i)
		if inCategory(r) {
			runes = append(runes, r)
		}
	}
	return rangetable.New(runes...)
}

// PropList.txt has the same format as Scripts.txt so we can share its parser.
func loadScriptOrProperty(doProps bool) map[string]*unicode.RangeTable {
	if gen.UnicodeVersion() == unicode.Version {
		if doProps {
			return unicode.Properties
		}
		return unicode.Scripts
	}

	file := "Scripts.txt"
	table := scripts
	if doProps {
		file = "PropList.txt"
		table = props
	}
	ucd.Parse(gen.OpenUCDFile(file), func(p *ucd.Parser) {
		name := p.String(1)
		table[name] = append(table[name], p.Rune(0))
	})

	// Handle deprecated "STerm" alias (this is only needed for tests)
	if rt, ok := table["Sentence_Terminal"]; ok {
		table["STerm"] = rt
	}
	tab := make(map[string]*unicode.RangeTable, len(table))
	for name, runes := range table {
		tab[name] = rangetable.New(runes...)
	}
	return tab
}

func loadCasefold() (foldCategory, foldScript map[string]*unicode.RangeTable) {
	// Build list of case-folding groups attached to each canonical folded char (typically lower case).
	var caseOrbit = make([][]rune, MaxChar+1)
	for j := range chars {
		i := rune(j)
		c := &chars[i]
		if c.foldCase == 0 {
			continue
		}
		orb := caseOrbit[c.foldCase]
		if orb == nil {
			orb = append(orb, c.foldCase)
		}
		caseOrbit[c.foldCase] = append(orb, i)
	}

	// Insert explicit 1-element groups when assuming [lower, upper] would be wrong.
	for j := range chars {
		i := rune(j)
		c := &chars[i]
		f := c.foldCase
		if f == 0 {
			f = i
		}
		orb := caseOrbit[f]
		if orb == nil && (c.upperCase != 0 && c.upperCase != i || c.lowerCase != 0 && c.lowerCase != i) {
			// Default assumption of [upper, lower] is wrong.
			caseOrbit[i] = []rune{i}
		}
	}

	// Delete the groups for which assuming [lower, upper] or [upper, lower] is right.
	for i, orb := range caseOrbit {
		if len(orb) == 2 && chars[orb[0]].upperCase == orb[1] && chars[orb[1]].lowerCase == orb[0] {
			caseOrbit[i] = nil
		}
		if len(orb) == 2 && chars[orb[1]].upperCase == orb[0] && chars[orb[0]].lowerCase == orb[1] {
			caseOrbit[i] = nil
		}
	}

	// Record orbit information in chars.
	for _, orb := range caseOrbit {
		if orb == nil {
			continue
		}
		sort.Slice(orb, func(i, j int) bool {
			return orb[i] < orb[j]
		})
		c := orb[len(orb)-1]
		for _, d := range orb {
			chars[c].caseOrbit = d
			c = d
		}
	}

	loadAsciiFold()
	loadCaseOrbit()

	// Fast exit if we don't need to recalculate these tables.
	if gen.UnicodeVersion() == unicode.Version {
		return unicode.FoldCategory, unicode.FoldScript
	}

	// Tables of category and script folding exceptions: code points
	// that must be added when interpreting a particular category/script
	// in a case-folding context.
	cat := make(map[string]map[rune]bool)
	for name := range category {
		if x := foldExceptions(inCategory(name)); len(x) > 0 {
			cat[name] = x
		}
	}

	scr := make(map[string]map[rune]bool)
	for name := range scripts {
		if x := foldExceptions(scripts[name]); len(x) > 0 {
			scr[name] = x
		}
	}

	return loadCatFold(cat), loadCatFold(scr) // TODO: this takes ~60ms
}

func loadAsciiFold() {
	for i := rune(0); i <= unicode.MaxASCII; i++ {
		c := chars[i]
		f := c.caseOrbit
		if f == 0 {
			switch {
			case c.lowerCase != i && c.lowerCase != 0:
				f = c.lowerCase
			case c.upperCase != i && c.upperCase != 0:
				f = c.upperCase
			default:
				f = i
			}
		}
		asciiFold[i] = uint16(f)
	}
}

// TODO: rename
func loadCaseOrbit() {
	for i := range chars {
		c := &chars[i]
		if c.caseOrbit != 0 {
			caseOrbit = append(caseOrbit, foldPair{uint32(i), uint32(c.caseOrbit)})
		}
	}
}

// inCategory returns a list of all the runes in the category.
func inCategory(name string) []rune {
	var x []rune
	for j := range chars {
		i := rune(j)
		c := &chars[i]
		if c.category == name || len(name) == 1 && len(c.category) > 1 && c.category[0] == name[0] {
			x = append(x, i)
		}
	}
	// fmt.Printf("%s: %d\n", name, len(x))
	return x
}

// foldExceptions returns a list of all the runes fold-equivalent
// to runes in class but not in class themselves.
func foldExceptions(class []rune) map[rune]bool {
	// Create map containing class and all fold-equivalent chars.
	m := make(map[rune]bool)
	for _, r := range class {
		c := &chars[r]
		if c.caseOrbit == 0 {
			// Just upper and lower.
			if u := c.upperCase; u != 0 {
				m[u] = true
			}
			if l := c.lowerCase; l != 0 {
				m[l] = true
			}
			m[r] = true
			continue
		}
		// Otherwise walk orbit.
		r0 := r
		for {
			m[r] = true
			r = chars[r].caseOrbit
			if r == r0 {
				break
			}
		}
	}

	// Remove class itself.
	for _, r := range class {
		delete(m, r)
	}

	// What's left is the exceptions.
	return m
}

func loadCatFold(m map[string]map[rune]bool) map[string]*unicode.RangeTable {
	folds := allCatFold(m)
	tabs := make(map[string]*unicode.RangeTable, len(folds))
	for _, name := range folds {
		class := m[name]
		tabs[name] = dumpRange(func(code rune) bool { return class[code] })
	}
	return tabs
}

// TODO: move to the top
func initTables(root, tablesFile string) {
	loadTableInfo(root, tablesFile)
	loadChars()
	loadCaseFolds()                               // download Unicode tables
	foldCategories, foldScripts := loadCasefold() // TODO: this is slow (~200ms)

	cats := []map[string]*unicode.RangeTable{
		loadCategoryTables(),
		loadScriptOrProperty(false),
		loadScriptOrProperty(true),
		foldCategories,
		foldScripts,
	}

	tabs := make([]*unicode.RangeTable, 0, len(cats))
	for _, m := range cats {
		for _, rt := range m {
			tabs = append(tabs, rt)
		}
	}
	categories = rangetable.Merge(tabs...)

	caseRanges = generateCaseRanges()
}

func requireFlags(names ...string) {
	var missing []string
	for _, name := range names {
		if flag.Lookup(name) == nil {
			missing = append(missing, name)
		}
	}
	if len(missing) > 0 {
		log.Panicf("The following flags were not registered: %q", missing)
	}
}

func listSourceFiles(dir string) []string {
	match := func(name string) bool {
		if name == "go.mod" || name == "go.sum" {
			return true
		}
		if strings.HasSuffix(name, "_test.go") {
			return false
		}
		switch filepath.Ext(name) {
		case ".go", ".c", ".cc", ".cpp", ".cxx", ".f", ".F", ".f90", ".for",
			".h", ".hh", ".hpp", ".hxx", ".m", ".s", ".S", ".swig", ".swigcxx",
			".sx", ".syso":
			return true
		}
		return false
	}
	var a []string
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		name := d.Name()
		if d.IsDir() && (name == "vendor" || name[0] == '.') {
			return fs.SkipDir
		}
		if match(name) {
			a = append(a, path)
		}
		return nil
	})
	if err != nil {
		log.Fatal(err)
	}
	slices.Sort(a)
	return a
}

func hashGenFiles() string {
	root, err := util.GenTablesRoot()
	if err != nil {
		log.Fatal(err)
	}

	files := listSourceFiles(root)
	if len(files) == 0 {
		log.Fatal("no go source files in:", root)
	}

	h := sha256.New()
	for _, name := range files {
		fmt.Fprintf(h, "%s\x00", filepath.Base(name))
		data, err := os.ReadFile(name)
		if err != nil {
			log.Fatal(err)
		}
		_, _ = h.Write(data)
		_, _ = h.Write([]byte{'\x00', '\x00'})
	}

	return fmt.Sprintf("%x", h.Sum(nil))
}

var useCachedSeeds = flag.Bool("cache", false,
	"used cached seeds instead of regenerating (for testing only)")

func realMain() int {
	initLogs() // Other packages configure logs on init so do it again here

	root, err := util.ProjectRoot()
	if err != nil {
		log.Panic(err)
	}

	flag.Usage = func() {
		flag.CommandLine.SetOutput(os.Stdout) // Write help to STDOUT
		fmt.Fprintf(flag.CommandLine.Output(), "Usage: %s [OPTION]...\n",
			filepath.Base(os.Args[0]))
		flag.PrintDefaults()
	}

	// NB(charlie): the "-url", "-iana", "-unicode", and "-cldr" flags are
	// registered by the github.com/charlievieth/strcase/internal/gen/gen package.
	requireFlags("url", "iana", "unicode", "cldr")

	skipTests := flag.Bool("skip-tests", false, "skip running tests")
	skipBuild := flag.Bool("skip-build", false, "skip building the strcase package (testing only)")
	dryRun := flag.Bool("dry-run", false,
		"report if generate would change the generated tables file and exit non-zero")
	// TODO: remove `-update-gen-hash` since it's dangerous and bypasses
	// our checks.
	updateGenHash := flag.Bool("update-gen-hash", false,
		`only update the hash of the gen.go file stored in ".tables.go" `+
			`(WARN: this is only for development)`)
	cpuprofile := flag.String("cpuprofile", "",
		"write cpu profile to `file`\n"+
			"NOTE: this traps SIGINT.\n"+
			"  First SIGINT the cpu profile is written to 'file'.\n"+
			"  Second SIGINT the program aborts.")

	firstValidHash := flag.Bool("first-valid-hash", false,
		"use the first valid hash instead of performing an exhaustive search (testing only, much faster)")

	outputDir := flag.String("dir", filepath.Join(root, "internal", "tables"),
		"write generated tables to this directory")
	// outputDir := flag.String("dir", filepath.Join(root),
	// 	"write generated tables to this directory")

	flag.Parse()

	if flag.NArg() != 0 {
		flag.Usage()
		return 1
	}

	// Validate Unicode version flag

	var supportedVersions []string
	for v := range buildTags {
		supportedVersions = append(supportedVersions, v)
	}
	if _, ok := buildTags[gen.UnicodeVersion()]; !ok {
		slices.Sort(supportedVersions)
		log.Printf("The selected Unicode version %q is unsupported. Either the version\n"+
			"is incorrect or this code needs to updated to handle a new version of Go.\n"+
			"The supported Unicode versions are: %q.", gen.UnicodeVersion(), supportedVersions)
		return 1
	}

	tablesFile := tablesFileName(*outputDir)
	// Use the Unicode version as the log prefix since we invoke this program
	// multiple times with different versions.
	log.SetPrefix("(" + gen.UnicodeVersion() + ") ")

	log.Println("Tables file:", tablesFile)

	if *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)
		if err != nil {
			log.Fatal("could not create CPU profile: ", err)
		}
		if err := pprof.StartCPUProfile(f); err != nil {
			log.Fatal("could not start CPU profile: ", err)
		}
		defer pprof.StopCPUProfile()
		ch := make(chan os.Signal, 1)
		signal.Notify(ch, os.Interrupt)
		go func() {
			<-ch
			log.Println("writing CPU profile: next interrupt will stop the program")
			pprof.StopCPUProfile()
			if err := f.Close(); err != nil {
				log.Printf("error closing CPU profile: %v", err)
			}
			signal.Reset(os.Interrupt)
		}()
	}

	if !*updateGenHash && *outputDir != "." {
		if err := os.MkdirAll(*outputDir, 0755); err != nil {
			log.Println("error:", err)
			return 1
		}
	}

	buildTags := buildTags[gen.UnicodeVersion()].buildTags
	if buildTags == "" {
		log.Printf("missing build tags for unicode version: %q", gen.UnicodeVersion())
		return 1
	}

	initTables(root, tablesFile)
	fileHash := hashGenFiles() // hash gentables source files
	foldHash := hashCaseFolds()

	chop := func(s string, n int) string {
		if len(s) >= n {
			return s[:n]
		}
		return s
	}

	if fileExists(tablesFile) &&
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
			chop(tableInfo.CaseFoldHash, 8), chop(tableInfo.GenGoHash, 8))
		return 0
	}

	isTerm := term.IsTerminal(1)
	ansi := func(color int, s string) string {
		if !isTerm {
			return s
		}
		return fmt.Sprintf("\x1b[%d;m%s\x1b[0;m", color, s)
	}
	colorize := func(args ...string) []any {
		if len(args)&1 != 0 {
			log.Panicf("number of args (%d) must be even!", len(args))
		}
		// Quote args since we can't use '%q' in log.Printf
		for i, s := range args {
			args[i] = strconv.Quote(s)
		}
		// Colorize args that have changed if output is a terminal
		if isTerm {
			for i := 0; i < len(args); i += 2 {
				if args[i] != args[i+1] {
					args[i] = ansi(32, args[i])     // green
					args[i+1] = ansi(31, args[i+1]) // red
				}
			}
		}
		a := make([]any, len(args))
		for i, s := range args {
			a[i] = s
		}
		return a
	}

	// TODO: print a relative path
	log.Printf("gen: would update "+tablesFile+" due to the following changes:\n"+
		"    unicode_version: %s   => %s\n"+
		"    cldr_version:    %s       => %s\n"+
		"    case_fold_hash:  %s => %s\n"+
		"    gen_go_hash:     %s => %s\n\n",
		colorize(tableInfo.UnicodeVersion, gen.UnicodeVersion(),
			tableInfo.CLDRVersion, gen.CLDRVersion(),
			chop(tableInfo.CaseFoldHash, 8), chop(foldHash, 8),
			chop(tableInfo.GenGoHash, 8), chop(fileHash, 8))...)
	if *dryRun {
		log.Printf("%s gen: would change %s "+
			"(remove -dry-run flag to update the generated files)\n",
			tablesFile, ansi(33, "WARN:"))
		log.Printf("%s gen: exiting now\n", ansi(33, "WARN:"))

		return 1
	}

	// NB: can't test or build if the Unicode version does not match
	// the version used by the Go binary running this.
	if !*updateGenHash {
		var w bytes.Buffer

		w.WriteString("\n\nimport \"unicode\"\n\n")
		gen.WriteUnicodeVersion(&w)

		writeInitGuard(&w)
		writeTypes(&w)
		writeFunctions(&w)

		genCaseFolds(&w, *firstValidHash)
		genUpperLowerTable(&w, *firstValidHash)
		genFoldTable(&w, *firstValidHash)

		writeGo(&w, tablesFile, buildTags)
		if *skipBuild {
			log.Println("gen: skipping go build")
		} else {
			if gen.UnicodeVersion() != unicode.Version {
				log.Printf("gen: \"go build\" is ineffective because the generated file "+
					"will be excluded due to Unicode version: %q != %q",
					tableInfo.UnicodeVersion, unicode.Version)
			}
			testBuild(tablesFile, w.Bytes(), *skipTests)
		}

		// For dry runs only report if tables.go would be changed and
		// exit with an error if so.
		if *dryRun {
			// TODO: this might be unreachable
			if !dataEqual(tablesFile, w.Bytes()) {
				fmt.Printf("gen: would change %s", tablesFile)
				return 1
			}
			return 0
		}

		writeFile(tablesFile, w.Bytes())
	}

	updateTableInfoFile(root, tablesFile, fileHash, foldHash)
	log.Printf("Successfully generated tables:\n"+
		"    unicode_version: %q\n"+
		"    cldr_version:    %q\n"+
		"    case_fold_hash:  %q\n"+
		"    gen_go_hash:     %q\n",
		tableInfo.UnicodeVersion, tableInfo.CLDRVersion,
		chop(tableInfo.CaseFoldHash, 8), chop(tableInfo.GenGoHash, 8))

	// Exit 1 if we only update the hash of the generate files since this
	// is a development only flag.
	if *updateGenHash {
		return 1
	}
	return 0
}

func main() {
	if code := realMain(); code != 0 {
		os.Exit(code)
	}
}
