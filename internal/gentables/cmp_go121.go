//go:build go1.21
// +build go1.21

package main

import "cmp"

// cmpCompare is a copy of cmp.Compare.
func cmpCompare[T cmp.Ordered](x, y T) int {
	return cmp.Compare(x, y)
}
