package test

import (
	"math/rand"
	"reflect"
	"testing"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/charlievieth/strcase/internal/tables/assigned"
)

func TestInvalidRune(t *testing.T) {
	rr := rand.New(rand.NewSource(time.Now().UnixNano()))
	for i := 0; i < 10_000; i++ {
		r := invalidRune(rr)
		if utf8.ValidRune(r) {
			t.Fatalf("utf8.ValidRune(%q) = %t; want: %t", r, true, false)
		}
	}
}

// Test that our generated tables contain all the desired Unicode points
// since we use a subset of Unicode categories to generate them for perf
// reasons.
func TestGeneratedRuneTables(t *testing.T) {
	if testing.Short() {
		t.Skip("short test")
	}
	all := assigned.Assigned(unicode.Version)
	if all == nil {
		t.Fatal("no assigned tables for Unicode version:", unicode.Version)
	}

	diff := func(got, want []rune) (extra, missing []rune) {
		gm := make(map[rune]bool, len(got))
		wm := make(map[rune]bool, len(want))
		for _, r := range got {
			gm[r] = true
		}
		for _, r := range want {
			if !gm[r] {
				missing = append(missing, r)
			}
			wm[r] = true
		}
		for _, r := range got {
			if !wm[r] {
				extra = append(extra, r)
			}
		}
		return extra, missing
	}
	compare := func(name string, got, want []rune) {
		t.Helper()
		if reflect.DeepEqual(got, want) {
			return
		}
		extra, missing := diff(got, want)
		t.Errorf("%s table is missing or contains extra runes\n"+
			"Extra:   0x%04X\n"+
			"Missing: 0x%04X\n", name, extra, missing)
	}

	multiwidth, foldable := generateRuneTables(all)
	compare("multiwidthRunes", multiwidth, multiwidthRunes)
	compare("foldableRunes", foldableRunes, foldable)
}
