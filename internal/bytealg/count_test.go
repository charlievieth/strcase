package bytealg

import (
	"strings"
	"testing"
)

var CountTests = []struct {
	s   string
	sep byte
	num int
}{
	{"12345678987654321", '6', 2},
	{"611161116", '6', 3},
	{"11111", '1', 5},
	{"aBaB", 'a', 2},
	{"ABAB", 'a', 2},
	{strings.Repeat("AB", 256), 'a', 256},
	{strings.Repeat("ab", 256), 'A', 256},
}

func TestCount(t *testing.T) {
	for _, tt := range CountTests {
		if num := CountString(tt.s, tt.sep); num != tt.num {
			t.Errorf("Count(%q, %q) = %d, want %d", tt.s, tt.sep, num, tt.num)
		}
	}
}

func TestCountHard(t *testing.T) {
	s := strings.Repeat("AB", 32*1024)
	lower := strings.Repeat("ab", 32*1024)
	n := 0
	for i := 0; i < len(s); i++ {
		want := strings.Count(lower[:i], "a")
		got := CountString(s[:i], 'a')
		if got != want {
			t.Errorf("%d: want: %d got: %d", i, want, got)
			n++
		}
		if n >= 30 {
			t.Fatal("too many errors:", n)
		}
		if i%64*1024 == 0 {
			t.Log(i)
		}
	}
}
