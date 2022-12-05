//go:build cgo && !windows
// +build cgo,!windows

package cstr

/*
#include <stdlib.h>
#include <stddef.h>
#include <string.h>
#include <strings.h> // strcasecmp
#include <wchar.h>
#include <locale.h>
#include <assert.h>

static int cstr_init_locale(void) {
	if (setlocale(LC_ALL, "en_US.UTF-8")) {
		return 0;
	}
	return 1;
}

static ptrdiff_t cstr_strcasestr(const char *haystack, const char *needle) {
	assert(haystack);
	assert(needle);
	char *res = strcasestr(haystack, needle);
	return res != NULL ? (ptrdiff_t)(res - haystack) : -1;
}

static int cstr_towc(const char *s, wchar_t **out, ssize_t *out_len) {
	assert(s);
	assert(out);

	ssize_t n = s != NULL ? strlen(s) : 0;
	wchar_t *w = calloc(n + 1, sizeof(wchar_t));
	assert(w);
	wchar_t *wp = w;

	const char *p = s;
	const char *end = s + n;
	if (n > 0) {
		int rc;
		mbstate_t state;
		memset(&state, 0, sizeof(state));
		while ((rc = mbrtowc(wp, p, end - p, &state)) > 0) {
			p += rc;
			wp++;
		}
	}

	*out = w;
	if (out_len) {
		*out_len = wp - w + 1;
	}
	return 0;
}

static int cstr_wcscasecmp(const char *s1, const char *s2) {
	int ret = -2;
	wchar_t *w1, *w2 = NULL;
	if (cstr_towc(s1, &w1, NULL) != 0) {
		goto exit;
	}
	if (cstr_towc(s2, &w2, NULL) != 0) {
		goto exit;
	}
	ret = wcscasecmp(w1, w2);

exit:
	if (w1) {
		free(w1);
	}
	if (w2) {
		free(w2);
	}
	return ret;
}
*/
import "C"

import (
	"sync"
	"unsafe"
)

const Enabled = true

var initLocaleOnce sync.Once
var initLocaleOk bool

func initLocale() {
	initLocaleOnce.Do(func() {
		initLocaleOk = C.cstr_init_locale() == 0
	})
	if !initLocaleOk {
		panic("cstr: failed to set locale: \"en_US.UTF-8\"")
	}
}

func clamp(i int) int {
	if i < 0 {
		return -1
	}
	if i > 0 {
		return 1
	}
	return 0
}

func Strcasecmp(s, t string) int {
	initLocale()
	cs := C.CString(s)
	ct := C.CString(t)
	ret := int(C.strcasecmp(cs, ct))
	C.free(unsafe.Pointer(cs))
	C.free(unsafe.Pointer(ct))
	return clamp(ret)
}

func Wcscasecmp(s, t string) int {
	initLocale()
	cs := C.CString(s)
	ct := C.CString(t)
	ret := int(C.cstr_wcscasecmp(cs, ct))
	C.free(unsafe.Pointer(cs))
	C.free(unsafe.Pointer(ct))
	if ret == -2 {
		panic("internal error")
	}
	return clamp(ret)
}

func Strcasestr(haystack, needle string) int {
	initLocale()
	hp := C.CString(haystack)
	np := C.CString(needle)
	n := int(C.cstr_strcasestr(hp, np))
	C.free(unsafe.Pointer(hp))
	C.free(unsafe.Pointer(np))
	return n
}
