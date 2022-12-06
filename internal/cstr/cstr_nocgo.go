//go:build !cgo || windows
// +build !cgo windows

package cstr

const Enabled = false

func Strcasecmp(s, t string) int             { return 0 }
func Strcasestr(haystack, needle string) int { return 0 }
func Wcscasecmp(s, t string) int             { return 0 }
