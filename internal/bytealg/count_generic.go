// Copyright 2023 Charlie Vieth. All rights reserved.
// Use of this source code is governed by the MIT license.

//go:build !s390x && !wasm && !ppc64 && !amd64 && !arm64
// +build !s390x,!wasm,!ppc64,!amd64,!arm64

package bytealg

import (
	"bytes"
	"strings"
)

func Count(b []byte, c byte) int {
	if !isAlpha(c) {
		return bytes.Count(b, []byte{c})
	}
	n := 0
	c |= ' '
	for _, cc := range b {
		if cc|' ' == c {
			n++
		}
	}
	return n
}

func CountString(s string, c byte) int {
	if !isAlpha(c) {
		return strings.Count(s, string(c))
	}
	n := 0
	c |= ' '
	for i := 0; i < len(s); i++ {
		if s[i]|' ' == c {
			n++
		}
	}
	return n
}
