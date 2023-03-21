// Copyright 2023 Charlie Vieth. All rights reserved.
// Use of this source code is governed by the MIT license.

package bytealg

// Empirical data shows that using Index can get better
// performance when len(s) <= 16.
const MaxBruteForce = 16

// func init() {
// 	// Optimize cases where the length of the substring is less than 32 bytes
// 	MaxLen = 32
// }

// Cutover reports the number of failures of IndexByte we should tolerate
// before switching over to Index.
// n is the number of bytes processed so far.
// See the bytes.Index implementation for details.
func Cutover(n int) int {
	// 1 error per 16 characters, plus a few slop to start.
	return 4 + n>>4
}

func IndexByte(s []byte, c byte) int
func IndexByteString(s string, c byte) int
