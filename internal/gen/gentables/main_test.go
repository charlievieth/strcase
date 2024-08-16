package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"slices"
	"sort"
	"syscall"
	"testing"
	"unicode"

	"github.com/charlievieth/strcase/internal/gen/util"
	"github.com/stretchr/testify/assert"
	"golang.org/x/term"
	"golang.org/x/text/unicode/rangetable"
)

func TestMain(m *testing.M) {
	// Tests must run with the same CLDR and Unicode version of the Go
	// version running the tests. This is because we use the unicode
	// package to assert that our code is correct.
	for _, name := range []string{"cldr", "unicode"} {
		f := flag.Lookup(name)
		if f.Value.String() != f.DefValue {
			panic(`The "-` + name + `" flag may not be set for tests.`)
		}
	}
	root, err := util.ProjectRoot()
	if err != nil {
		panic(err)
	}

	initTables(root, filepath.Join("../../tables", tablesFileName(unicode.Version)))
	os.Exit(m.Run())
}

func sortedKeys[M ~map[string]V, V any](m M) []string {
	a := make([]string, 0, len(m))
	for k := range m {
		a = append(a, k)
	}
	slices.Sort(a)
	return a
}

func writeJSON(t testing.TB, filename string, v any) {
	t.Helper()
	data, err := json.MarshalIndent(v, "", "    ")
	if err != nil {
		t.Fatal(err)
	}
	data = append(data, '\n')
	if err := os.WriteFile(filename, data, 0644); err != nil {
		t.Fatal(err)
	}
}

func diff(t testing.TB, v1, v2 any) {
	if _, err := exec.LookPath("diff"); err != nil {
		assert.Equal(t, v1, v2)
		return
	}

	// Don't use t.TempDir() since we want to preserve the files so
	// that users can inspect them on test failure.
	tmp, err := os.MkdirTemp("", "strcase.*")
	if err != nil {
		t.Fatal(err)
	}
	writeJSON(t, tmp+"/want.json", v1)
	writeJSON(t, tmp+"/got.json", v2)

	args := []string{
		"-u",
		tmp + "/want.json",
		tmp + "/got.json",
	}
	if term.IsTerminal(syscall.Stdout) {
		args = append([]string{"--color=always"}, args...)
	}

	cmd := exec.Command("diff", args...)
	cmd.Dir = tmp

	out, err := cmd.CombinedOutput()
	out = bytes.TrimSpace(out)
	if err != nil {
		var ee *exec.ExitError
		if errors.As(err, &ee) {
			if ee.ExitCode() != 1 {
				t.Fatalf("error running: %q: %v\n%s", cmd.Args, err, out)
			}
		}
		// Ok, exit code is due to the diff
	}
	t.Errorf("\n%s\nTMPDIR: %s\nvimdiff %s %s",
		out, tmp, tmp+"/want.json", tmp+"/got.json")
}

func rangetableEqual(r1, r2 *unicode.RangeTable) bool {
	if (r1 == nil) != (r2 == nil) {
		return false
	}
	return r1 != nil && r1.LatinOffset == r2.LatinOffset &&
		slices.Equal(r1.R16, r2.R16) && slices.Equal(r1.R32, r2.R32)
}

func compareRangeTables(t *testing.T, want, got map[string]*unicode.RangeTable) {
	t.Run("Keys", func(t *testing.T) {
		k1 := sortedKeys(want)
		k2 := sortedKeys(got)
		if !slices.Equal(k1, k2) {
			diff(t, k1, k2)
		}
	})
	if t.Failed() {
		return
	}

	t.Run("Tables", func(t *testing.T) {
		failures := 0
		for key, rt1 := range want {
			t.Run(key, func(t *testing.T) {
				rt2 := got[key]
				if !rangetableEqual(rt1, rt2) {
					diff(t, rt1, rt2)
					t.Fail()
					failures++
				}
			})
			if failures >= 10 {
				t.Fatal("Too many errors:", failures)
			}
		}
	})
}

func TestCategories(t *testing.T) {
	compareRangeTables(t, unicode.Categories, loadCategoryTables())
}

func TestScript(t *testing.T) {
	compareRangeTables(t, unicode.Scripts, loadScriptOrProperty(false))
}

func TestProperty(t *testing.T) {
	compareRangeTables(t, unicode.Properties, loadScriptOrProperty(true))
}

func TestLoadCasefold(t *testing.T) {
	categories, scripts := loadCasefold()
	t.Run("FoldCategory", func(t *testing.T) {
		compareRangeTables(t, unicode.FoldCategory, categories)
	})
	t.Run("FoldScript", func(t *testing.T) {
		compareRangeTables(t, unicode.FoldScript, scripts)
	})
}

func TestUnicodeCategories(t *testing.T) {
	rangeTable := func(a []map[string]*unicode.RangeTable) *unicode.RangeTable {
		tabs := make([]*unicode.RangeTable, 0, len(a))
		for _, m := range a {
			for _, rt := range m {
				tabs = append(tabs, rt)
			}
		}
		return rangetable.Merge(tabs...)
	}
	want := []map[string]*unicode.RangeTable{
		unicode.Categories,
		unicode.Scripts,
		unicode.Properties,
		unicode.FoldCategory,
		unicode.FoldScript,
	}

	foldCategories, foldScripts := loadCasefold()
	got := []map[string]*unicode.RangeTable{
		loadCategoryTables(),
		loadScriptOrProperty(false),
		loadScriptOrProperty(true),
		foldCategories,
		foldScripts,
	}
	for i := range want {
		k1 := sortedKeys(want[i])
		k2 := sortedKeys(got[i])
		assert.Equal(t, k1, k2)
	}

	for i, name := range []string{"Categories", "Scripts", "Properties", "FoldCategory", "FoldScript"} {
		if i >= len(want) {
			continue
		}
		t.Run(name, func(t *testing.T) {
			compareRangeTables(t, want[i], got[i])
		})
	}

	compareRangeTables(
		t,
		map[string]*unicode.RangeTable{"All": rangeTable(want)},
		map[string]*unicode.RangeTable{"All": rangeTable(got)},
	)

	rangetableEqual(rangeTable(want), categories)
}

func TestGenerateSpans(t *testing.T) {
	t.Run("RandomShuffle", func(t *testing.T) {
		var spans []span
		for i := 0; i < 10; i++ {
			spans = generateSpans(0, 100, 7)
			sorted := sort.SliceIsSorted(spans, func(i, j int) bool {
				return spans[i].Start < spans[j].Start
			})
			if !sorted {
				return
			}
		}
		t.Errorf("spans are not randomly shuffled: %+v", spans)
	})
	t.Run("Complete", func(t *testing.T) {
		want := []span{
			{Start: 1, End: 8},
			{Start: 8, End: 15},
			{Start: 15, End: 22},
			{Start: 22, End: 29},
			{Start: 29, End: 36},
			{Start: 36, End: 43},
			{Start: 43, End: 50},
			{Start: 50, End: 50},
		}
		spans := generateSpans(1, 50, 7)
		sort.Slice(spans, func(i, j int) bool {
			return spans[i].Start < spans[j].Start
		})
		if !reflect.DeepEqual(want, spans) {
			diff(t, want, spans)
		}
		// assert.Equal(t, want, spans)
	})
}

func TestGenerateCaseRanges(t *testing.T) {
	got := caseRanges
	want := unicode.CaseRanges
	if !reflect.DeepEqual(got, want) {
		diff(t, want, got)
	}
}

func TestToUpperLower(t *testing.T) {
	for r := rune(0); r <= unicode.MaxRune; r++ {
		if toLower(r) != unicode.ToLower(r) {
			t.Errorf("toLower(%U) = %U; want: %U", r, toLower(r), unicode.ToLower(r))
		}
		if toUpper(r) != unicode.ToUpper(r) {
			t.Errorf("toUpper(%U) = %U; want: %U", r, toUpper(r), unicode.ToUpper(r))
		}
	}
}

func TestSimpleFold(t *testing.T) {
	for r := rune(0); r <= unicode.MaxRune; r++ {
		if simpleFold(r) != unicode.SimpleFold(r) {
			t.Errorf("simpleFold(%[1]q/%[1]U) = %[2]q/%[2]U; want: %[3]q/%[3]U", r,
				simpleFold(r), unicode.SimpleFold(r))
		}
	}
}

// WARN: DELETE ME
// func TestFindModfile(t *testing.T) {
// 	wd, err := os.Getwd()
// 	if err != nil {
// 		t.Fatal(err)
// 	}
// 	t.Fatal(findModfile(wd, "", "github.com/charlievieth/strcase"))
// }

/*
func diff(t testing.TB, v1, v2 any) {
	if _, err := exec.LookPath("git"); err != nil {
		assert.Equal(t, v1, v2)
		return
	}

	tmp, err := os.MkdirTemp("", "strcase.*")
	if err != nil {
		t.Fatal(err)
	}
	writeJSON(t, tmp+"/want.json", v1)
	writeJSON(t, tmp+"/got.json", v2)

	args := []string{
		"diff",
		"--no-index",
		"--exit-code",
		"--color=always",
		"--histogram",
		"--ignore-cr-at-eol",
		"--unified=6",
		tmp + "/want.json",
		tmp + "/got.json",
	}
	cmd := exec.Command("git", args...)
	cmd.Dir = tmp

	out, err := cmd.CombinedOutput()
	out = bytes.TrimSpace(out)
	if err != nil {
		var ee *exec.ExitError
		if errors.As(err, &ee) {
			if ee.ExitCode() != 1 {
				t.Fatalf("error running: %q: %v\n%s", cmd.Args, err, out)
			}
		}
		// Ok, exit code is due to the diff
	}
	t.Errorf("%s\nTMPDIR: %s\nvimdiff %s %s",
		out, tmp, tmp+"/want.json", tmp+"/got.json")
}
*/
