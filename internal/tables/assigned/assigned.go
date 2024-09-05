package assigned

import (
	"sync"
	"unicode"
)

// Assigned returns a RangeTable with all assigned code points for a given
// Unicode version. This includes graphic, format, control, and private-use
// characters. It returns nil if the data for the given version is not
// available.
func Assigned(version string) *unicode.RangeTable {
	return assigned[version]
}

var runes sync.Map

// Assigned returns a slice of runes with all assigned code points for a given
// Unicode version. This includes graphic, format, control, and private-use
// characters. An empty slice is returned if the data for the given version is
// not available.
func AssignedRunes(version string) []rune {
	if v, ok := runes.Load(version); ok {
		return v.(func() []rune)()
	}
	rt := Assigned(version)
	if rt == nil {
		return nil
	}
	var all []rune
	var once sync.Once
	fn := func() []rune {
		once.Do(func() {
			n := 0
			visit(rt, func(_ rune) {
				n++
			})
			all = make([]rune, 0, n)
			visit(rt, func(r rune) {
				all = append(all, r)
			})
		})
		return all
	}
	if v, loaded := runes.LoadOrStore(version, fn); loaded {
		return v.(func() []rune)()
	}
	return fn()
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
