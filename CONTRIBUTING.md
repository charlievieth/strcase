# Contributing

## How to contribute

We definitely welcome patches and contribution to this project!  

In particular we would love feedback and contributions from anyone with real
world experience using this project, and would love help with adding assembly
implementations of IndexByte, CountByte, and IndexNonASCII for processor
architectures (GOARCH) that we don't already have asm implementations for (a
review of the current amd64 and arm64 implementations by someone who actually
knows these architectures would also be welcome).

## Discuss Significant Changes

Before you invest a significant amount of time on a change, please create a
discussion or issue describing your proposal. This will help to ensure your
proposed change has a reasonable chance of being merged.

## Avoid Dependencies

Adding a dependency is a big deal. While on occasion a new dependency may be
accepted, the default answer to any change that adds a dependency is no.

## Avoid Allocating

Allocating memory is never allowed unless the corresponding function in
the [strings](https://pkg.go.dev/strings) package also allocates memory
or if the change was previously discussed and provides a speedup that
justifies the allocation.

## Development Environment Setup

Run `make pre-commit` to install a git pre-commit hook that checks if either
[gen.go](./gen.go) or [.tables.json](./.tables.json) are out of date.
Any change to [gen.go](./gen.go) requires the code generation to be re-run (`go
generate`) and the resulting changes to [.tables.json](./.tables.json) to be
committed along side any changes to [gen.go](./gen.go).
The [.tables.json](./.tables.json) stores a hash of the [gen.go](./gen.go) file
so that we don't accidentally change hash table generation logic without also
re-running the generation code (this may change in a future release since
re-generating the hash tables is slow).

The `testgenerate` make target can be used to check if `go generate` needs to
be ran.

## Running Tests

Before submitting a PR run the `release` make target (`make release`).
This will run all the tests, perform a more exhaustive fuzz test, and
lint the code.

During development running `go test` is generally sufficient.

## Running Benchmarks

The [internal/benchtest](./internal/benchtest) directory contains the release
benchmarks that are used to compare strcase to the strings package see the
packages [README](./internal/benchtest/README.md) for more information. But
TLDR `make release` can be ran from that directory to generate a statistically
accurate comparison (note: it is very slow).

Any strcase specific benchmarks should be added to
[`strcase_test.go`](./strcase_test.go).
