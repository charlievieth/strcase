// Copyright 2023 Charlie Vieth. All rights reserved.
// Use of this source code is governed by the MIT license.

// Package strcase is a fast case-insensitive implementation of the Go standard
// library's [strings] package.
//
// Except where noted, simple Unicode case-folding is used to determine equality.
//
// [strings]: https://pkg.go.dev/strings
package strcase

// TODO: make sure package doc is accurate.

// BUG(cvieth): There is no mechanism for full case folding, that is, for
// characters that involve multiple runes in the input or output
// (see: https://pkg.go.dev/unicode#pkg-note-BUG).
