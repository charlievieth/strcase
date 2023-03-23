# strcase

[![GoDoc](https://img.shields.io/badge/godoc-reference-blue.svg)](https://pkg.go.dev/github.com/charlievieth/strcase)
[![Tests](https://github.com/charlievieth/strcase/actions/workflows/test.yml/badge.svg)](https://github.com/charlievieth/strcase/actions/workflows/test.yml)

Package strcase is a case-insensitive and Unicode aware implementation of the
Go standard library's [`strings`](https://pkg.go.dev/strings) package.

strcase uses Unicode simple case folding to determine equality, none of it's
functions allocate memory, and it is optimized for GOARCH `amd64` and `arm64`.

## Features

- Accurate: Unicode simple folding is used to determine equality.
   - Any matched text would also match with [`strings.EqualFold`](https://pkg.go.dev/strings#EqualFold).
- Fast: strcase is optimized for `amd64` and `arm64`.
- Zero allocation: none of the strcase functions allocated memory.
- Thoroughly tested and fuzzed.
