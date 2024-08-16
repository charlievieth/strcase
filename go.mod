module github.com/charlievieth/strcase

go 1.19

// TODO: go1.22 uses v0.15.0 - maybe this is why we aren't using
// the optimized assembly (not detecting POPCNT)
require golang.org/x/sys v0.16.0
