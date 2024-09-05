package assigned

import (
	"reflect"
	"testing"
	"unicode"
)

func TestAssigned(t *testing.T) {
	if Assigned(unicode.Version) == nil {
		t.Fatal("missing assigned Unicode points for version:", unicode.Version)
	}
}

func TestAssignedRunes(t *testing.T) {
	want := func(version string) []rune {
		var a []rune
		visit(Assigned(version), func(r rune) {
			a = append(a, r)
		})
		if len(a) == 0 {
			t.Fatal("invalid Unicode version:", version)
		}
		return a
	}
	for _, version := range []string{"13.0.0", "15.0.0"} {
		version := version
		t.Run(version, func(t *testing.T) {
			t.Parallel()
			w := want(version)
			a1 := AssignedRunes(version)
			if !reflect.DeepEqual(a1, w) {
				t.Error("AssignedRunes: invalid result") // don't print the massive slices
			}
			a2 := AssignedRunes(version)
			if &a1[0] != &a2[0] {
				t.Fatalf("AssignedRunes: result was not cached: %p == %p",
					&a1[0], &a2[0])
			}
		})
	}
}
