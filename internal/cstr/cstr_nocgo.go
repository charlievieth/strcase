//go:build !cgo
// +build !cgo

package cstr

func Strcasecmp(s, t string) int             { return 0 }
func Wcscasecmp(s, t string) int             { return 0 }
func Strcasestr(haystack, needle string) int { return 0 }
