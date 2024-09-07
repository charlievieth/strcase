// Copyright 2024 Charlie Vieth. All rights reserved.
// Use of this source code is governed by the MIT license.

/*
Package [bytcase] is a case-insensitive and Unicode aware implementation of the
Go standard library's [bytes] package that is fast, accurate, and never
allocates memory.

Simple Unicode case-folding is used for all comparisons. This matches the
behavior of [bytes.EqualFold].

Package bytcase also provides two functions for identifying non-ASCII characters
that are not available in the bytes package: [IndexNonASCII] and
[ContainsNonASCII].
On amd64 and arm64 these functions are implemented in assembly and their
performance is mostly governed by memory bandwidth.

[bytes]: https://pkg.go.dev/bytes
[bytcase]: https://pkg.go.dev/github.com/charlievieth/strcase/bytcase
*/
package bytcase

// BUG(cvieth): There is no mechanism for full case folding, that is, for
// characters that involve multiple runes in the input or output
// (see: https://pkg.go.dev/unicode#pkg-note-BUG).
//
// This is a limitation of Go's [unicode] package.
//
// [unicode]: https://pkg.go.dev/unicode#pkg-note-BUG
