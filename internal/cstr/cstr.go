//go:build cgo
// +build cgo

package cstr

/*
#include <stdlib.h>
#include <stddef.h>
#include <string.h>
#include <strings.h> // strcasecmp
#include <locale.h>
#include <wchar.h>
#include <wctype.h>
#include <assert.h>

static void cstr_init_locale(void) {
	setlocale(LC_ALL, "en_US.UTF-8");
}

#if !defined(__MINGW32__) && !defined(_WIN32) && !defined(__CYGWIN__)
#  define HAVE_STRCASESTR
#  define HAVE_WCSCASECMP
#endif

#ifndef HAVE_STRCASESTR
static int cstr_strncasecmp(const char *l, const char *r, size_t n) {
	if (!n--) {
		return 0;
	}
	for (; *l && *r && n && (*l == *r || tolower(*l) == tolower(*r)); l++, r++, n--)
		;
	return tolower(*l) - tolower(*r);
}
#endif // HAVE_STRCASESTR

#ifndef HAVE_WCSCASECMP
static int cstr_wcsncasecmp(const wchar_t *l, const wchar_t *r, size_t n) {
	if (!n--)
		return 0;
	for (; *l && *r && n && (*l == *r || towlower(*l) == towlower(*r)); l++, r++, n--)
		;
	return towlower(*l) - towlower(*r);
}
#endif // HAVE_WCSCASECMP

static ptrdiff_t cstr_strcasestr(const char *haystack, const char *needle) {
	assert(haystack);
	assert(needle);

#ifdef HAVE_STRCASESTR
	char *res = strcasestr(haystack, needle);
	return res != NULL ? (ptrdiff_t)(res - haystack) : -1;
#else
	const char *h = haystack;
	size_t nlen = strlen(needle);
	for (; *h; h++) {
		if (!cstr_strncasecmp(h, needle, nlen)) {
			return (ptrdiff_t)(h - haystack);
		}
	}
	return -1;
#endif
}

int cstr_towc(const char *s, wchar_t **out, ssize_t *out_len) {
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

int cstr_wcscasecmp(const char *s1, const char *s2) {
	wchar_t *w1, *w2 = NULL;
	if (cstr_towc(s1, &w1, NULL) != 0) {
		goto exit_error;
	}
	if (cstr_towc(s2, &w2, NULL) != 0) {
		goto exit_error;
	}
#ifdef HAVE_WCSCASECMP
	int ret = wcscasecmp(w1, w2);
#else
	int ret = cstr_wcsncasecmp(w1, w2, -1);
#endif // HAVE_WCSCASECMP
	free(w1);
	free(w2);
	return ret;

exit_error:
	if (w1) {
		free(w1);
	}
	if (w2) {
		free(w2);
	}
	return -2;
}
*/
import "C"
import "unsafe"

const Enabled = true

// TODO: do we need this?
func init() {
	C.cstr_init_locale()
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
	cs := C.CString(s)
	ct := C.CString(t)
	ret := int(C.strcasecmp(cs, ct))
	C.free(unsafe.Pointer(cs))
	C.free(unsafe.Pointer(ct))
	return clamp(ret)
}

func Wcscasecmp(s, t string) int {
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
	hp := C.CString(haystack)
	np := C.CString(needle)
	n := int(C.cstr_strcasestr(hp, np))
	C.free(unsafe.Pointer(hp))
	C.free(unsafe.Pointer(np))
	return n
}
