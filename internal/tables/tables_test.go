// Copyright 2023 Charlie Vieth. All rights reserved.
// Use of this source code is governed by the MIT license.

package tables

import (
	"strings"
	"testing"
	"unicode"
	"unicode/utf8"

	"github.com/charlievieth/strcase/internal/tables/assigned"
)

func TestCaseFold(t *testing.T) {
	t.Run("Limits", func(t *testing.T) {
		for r := unicode.MaxRune; r < unicode.MaxRune+10; r++ {
			x := CaseFold(r)
			if x != r {
				t.Errorf("CaseFold(0x%04X) = 0x%04X; want: 0x%04X", r, x, r)
			}
		}
		for r := rune(0); r < ' '; r++ {
			x := CaseFold(r)
			if x != r {
				t.Errorf("CaseFold(0x%04X) = 0x%04X; want: 0x%04X", r, x, r)
			}
		}
		if r := CaseFold(utf8.RuneError); r != utf8.RuneError {
			t.Errorf("CaseFold(0x%04X) = 0x%04X; want: 0x%04X", utf8.RuneError, r, utf8.RuneError)
		}
	})
	t.Run("ValidFolds", func(t *testing.T) {
		for _, p := range _CaseFolds {
			if r := CaseFold(rune(p.From)); r != rune(p.To) {
				t.Errorf("CaseFold(0x%04X) = 0x%04X; want: 0x%04X", rune(p.From), r, rune(p.To))
			}
		}
	})
	t.Run("UnicodeCases", func(t *testing.T) {
		folds := make(map[rune]rune)
		for _, p := range _CaseFolds {
			if p.From != 0 {
				folds[rune(p.From)] = rune(p.To)
			}
		}
		for _, rt := range unicode.Categories {
			visit(rt, func(r rune) {
				if rr, ok := folds[r]; ok {
					r = rr
				}
				if got := CaseFold(r); got != r {
					t.Errorf("CaseFold(0x%04X) = 0x%04X; want: 0x%04X", r, got, r)
				}
			})
		}
	})
	// Test against all assigned Unicode code points.
	t.Run("Assigned", func(t *testing.T) {
		all := assigned.AssignedRunes(unicode.Version)
		if len(all) == 0 {
			t.Fatalf("missing assigned code points for Unicode version: %q", unicode.Version)
		}
		n := 0
		for _, r := range all {
			sr := CaseFold(r)
			if sr != r {
				n++
			}
			if !strings.EqualFold(string(sr), string(r)) {
				t.Errorf("CaseFold(%q) = %q is an invalid fold", r, sr)
			}
		}
		if n == 0 {
			t.Fatal("failed to fold any runes")
		}
	})
}

func TestUpperLower(t *testing.T) {
	// Test against all assigned Unicode code points.
	all := assigned.AssignedRunes(unicode.Version)
	if len(all) == 0 {
		t.Fatalf("missing assigned code points for Unicode version: %q", unicode.Version)
	}
	for _, r := range all {
		u0, l0, _ := ToUpperLower(r)
		u1 := unicode.ToUpper(r)
		l1 := unicode.ToLower(r)
		if u0 != u1 || l0 != l1 {
			t.Errorf("ToUpperLower(0x%04X) = 0x%04X, 0x%04X want: 0x%04X, 0x%04X",
				r, u0, l0, u1, l1)
		}
	}
}

// visit visits all runes in the given RangeTable in order, calling fn for each.
func visit(rt *unicode.RangeTable, fn func(rune)) {
	for _, r16 := range rt.R16 {
		for r := rune(r16.Lo); r <= rune(r16.Hi); r += rune(r16.Stride) {
			fn(r)
		}
	}
	for _, r32 := range rt.R32 {
		for r := rune(r32.Lo); r <= rune(r32.Hi); r += rune(r32.Stride) {
			fn(r)
		}
	}
}
