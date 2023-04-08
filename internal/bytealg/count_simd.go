// Copyright 2023 Charlie Vieth. All rights reserved.
// Use of this source code is governed by the MIT license.

//go:build s390x || wasm || ppc64
// +build s390x wasm ppc64

package bytealg

import (
	"bytes"
	"strings"
)

func Count(b []byte, c byte) int {
	s := []byte{c}
	if !isAlpha(c) {
		return bytes.Count(b, s)
	}
	n := bytes.Count(b, s)
	s[0] ^= ' ' // swap case
	return n + bytes.Count(b, s)
}

func CountString(s string, c byte) int {
	if !isAlpha(c) {
		return strings.Count(s, string(c))
	}
	return strings.Count(s, string(c)) +
		strings.Count(s, string(c^' ')) // swap case
}
