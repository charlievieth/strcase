// Package benchtest is used for benchmarking strcase against the Go stdlib's
// strings package.
//
// With few exceptions the benchmarks here were taken directly from Go's
// strings and bytes package.
//
// It is not part of the strcase package since some of the benchmarks here are
// not entirely relevant to strcase (none require case conversions).
// Instead they are a useful measure of the overhead of strcase compared to
// the stdlib's strings package.
package benchtest
