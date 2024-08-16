package tables

// NB: this table is not generated but probably should be

import (
	"testing"
	"unicode"
	"unicode/utf8"
)

func TestCaseFold(t *testing.T) {
	t.Run("Limits", func(t *testing.T) {
		for r := unicode.MaxRune; r < unicode.MaxRune+10; r++ {
			x := CaseFold(r)
			if x != r {
				t.Errorf("caseFold(0x%04X) = 0x%04X; want: 0x%04X", r, x, r)
			}
		}
		for r := rune(0); r < ' '; r++ {
			x := CaseFold(r)
			if x != r {
				t.Errorf("caseFold(0x%04X) = 0x%04X; want: 0x%04X", r, x, r)
			}
		}
		if r := CaseFold(utf8.RuneError); r != utf8.RuneError {
			t.Errorf("caseFold(0x%04X) = 0x%04X; want: 0x%04X", utf8.RuneError, r, utf8.RuneError)
		}
	})
	t.Run("ValidFolds", func(t *testing.T) {
		for _, p := range _CaseFolds {
			if r := CaseFold(rune(p.From)); r != rune(p.To) {
				t.Errorf("caseFold(0x%04X) = 0x%04X; want: 0x%04X", rune(p.From), r, rune(p.To))
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
			Visit(rt, func(r rune) {
				if rr, ok := folds[r]; ok {
					r = rr
				}
				if got := CaseFold(r); got != r {
					t.Errorf("caseFold(0x%04X) = 0x%04X; want: 0x%04X", r, got, r)
				}
			})
		}
	})
}

// Visit visits all runes in the given RangeTable in order, calling fn for each.
func Visit(rt *unicode.RangeTable, fn func(rune)) {
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