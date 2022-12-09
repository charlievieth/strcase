package strcase

import (
	"testing"
)

// var _FoldMapASCII = map[rune][]rune{
// 	0x004B: {0x004B, 0x006B, 0x212A}, // 'K': ['K' 'k' 'K']
// 	0x0053: {0x0053, 0x0073, 0x017F}, // 'S': ['S' 's' 'ſ']
// 	0x006B: {0x004B, 0x006B, 0x212A}, // 'k': ['K' 'k' 'K']
// 	0x0073: {0x0053, 0x0073, 0x017F}, // 's': ['S' 's' 'ſ']
// }

var _asciiFolds = [...][]rune{
	{0x004B, 0x006B, 0x212A},
	{0x0053, 0x0073, 0x017F},
	// {0x004B, 0x006B, 0x212A},
	// {0x0053, 0x0073, 0x017F},
}

var _foldsK = []rune{0x004B, 0x006B, 0x212A}
var _foldsS = []rune{0x0053, 0x0073, 0x017F}

func foldASCII(r rune) []rune {
	switch r {
	case 'K', 'k':
		return _foldsK
	case 'S', 's':
		return _foldsS
	}
	return nil
}

// var _empty2 [2]rune

// func foldExcludingUpperLower(r rune) [2]rune {
// 	return _FoldMapExcludingUpperLower[r]
// 	// if 0x004B <= r && r <= 0xA64B {
// 	// 	return _FoldMapExcludingUpperLower[r]
// 	// }
// 	// return _empty2
// }

func BenchmarkFoldMapASCII(b *testing.B) {
	var runes = [8]rune{'K', 'S', 'k', 's', 'ꙋ', 'ᲀ', 0xA64B + 1, 0xA64B + 2}
	for i := 0; i < b.N; i++ {
		// _, _ = _FoldMapASCII[runes[i%len(runes)]]
		// _ = foldASCII(runes[i%len(runes)])

		// r := runes[i%len(runes)]
		_ = foldExcludingUpperLower(runes[i%len(runes)])

		// if 0x004B <= r && r <= 0xA64B {
		// 	_ = _FoldMapExcludingUpperLower[r]
		// 	// _, _ = _FoldMap[r]
		// }

		// xFold(runes[i%len(runes)])
	}
}
