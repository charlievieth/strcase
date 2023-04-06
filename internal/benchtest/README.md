# benchtest

Package benchtest is used to benchmark the [github.com/charlievieth/strcase](https://pkg.go.dev/github.com/charlievieth/strcase)
package against the Go standard library's [strings](https://pkg.go.dev/strings) package.

The benchtest package is not part of the strcase package since some of the
benchmarks here are not entirely relevant to strcase (none require case
conversions). Instead they are a useful measure of the overhead / raw
performance of strcase compared to the stdlib's strings package.

## Usage

By default benchtest runs benchmarks using the
[strcase](https://pkg.go.dev/github.com/charlievieth/strcase)
package.

benchtest defines two flags that control how the benchmarks are run
(by default the strcase is used):
   * `-stdlib`: use the [`strings`](https://pkg.go.dev/strings) package
   * `-stdlib-case`: use the [`strings`](https://pkg.go.dev/strings) package
   and convert text with [`strings.ToUpper`](https://pkg.go.dev/strings#ToUpper)
   this mimics how most Go projects currently perform case-insensitive searches
   (note: `ToUpper` is used because our test strings are lower-case)
      * e.g. `strings.Index(strings.ToUpper(s), strings.ToUpper(sep))`

For convenience a [Makefile](./Makefile) is provided that has targets for:
`strcase`, `stdlib`, and `stdlib-case`. The `release` make target can be used
to generate a comprehensive comparison between strcase and strings (note:
it is very slow).

Run benchmarks using the [`strcase`](https://pkg.go.dev/github.com/charlievieth/strcase)
package:

```sh
$ go test -bench .               # or `make strcase`
```

Run benchmarks using the [`strings`](https://pkg.go.dev/strings) package:

```sh
$ go test -bench . -stdlib       # or `make stdlib`
```

Run benchmarks using the [`strings`](https://pkg.go.dev/strings) package and
convert the case of the strings being searched for with
[`strings.ToUpper`](https://pkg.go.dev/strings#ToUpper) on each invocation:

```sh
$ go test -bench . -stdlib-case  # or `make stdlib-case`
```

Comparing benchmarks with [`benchstat`](https://pkg.go.dev/golang.org/x/perf@v0.0.0-20230227161431-f7320a6d63e8/cmd/benchstat):

```sh
$ # NOTE: `make release` will also perform the following commands
$ go install golang.org/x/perf/cmd/benchstat@latest   # install benchstat
$ go test -bench . -count 5 -stdlib | tee stdlib.txt  # generate stdlib bench report
$ go test -bench . -count 5 | tee strcase.txt         # generate strcase bench report
$ benchstat stdlib.txt strcase.txt                    # compare strcase to stlib performance
```

**Note:** A count value of 5 is usually sufficient for benchstat. The timing
of these benchmarks are pretty consistent (low variance) so using a count of 1
is fine for a quick comparison, but if using a count of 1 the deprecated
[benchcmp](https://pkg.go.dev/golang.org/x/tools@v0.7.0/cmd/benchcmp) command
should be used since benchstat will complain that it needs more runs to
generate a statistically significant report.

Additionally, these benchmarks can take a long time to run so the `-timeout`
flag may be needed (`-timeout=30m` is usually sufficient) to prevent `go test`
from timing out (the Makefile sets this by default).

## Generating a benchmark report

Use the make `release` target to generate a comprehensive comparison between
`strcase` and `strings` (note: it is very slow). It should be used when cutting
new releases.
