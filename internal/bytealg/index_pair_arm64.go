// Copyright 2026 Charlie Vieth. All rights reserved.
// Use of this source code is governed by the MIT license.

package bytealg

// NativeIndexPair is true if we have a fast native (assembly)
// implementation of IndexPairFold.
const NativeIndexPair = true

// IndexPairFold returns the index of the first position i in s where s[i]
// matches c0 and s[i+1] matches c1, where the four bytes packed into pair
// are the lower and upper case forms of c0 and c1 (see PackPair), or -1 if
// there is no such position.
//
//go:noescape
func IndexPairFold(s string, pair uint32) int

// LastIndexPairFold returns the index of the last position i in s where s[i]
// matches c0 and s[i+1] matches c1, where the four bytes packed into pair
// are the lower and upper case forms of c0 and c1 (see PackPair), or -1 if
// there is no such position.
//
//go:noescape
func LastIndexPairFold(s string, pair uint32) int
