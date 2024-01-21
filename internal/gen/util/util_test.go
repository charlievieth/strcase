package util

import (
	"os"
	"path/filepath"
	"testing"
)

func TestProjectRoot(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(wd, "../../../")
	fi1, err := os.Stat(want)
	if err != nil {
		t.Fatal(err)
	}
	root, err := ProjectRoot()
	if err != nil {
		t.Fatal(err)
	}
	fi2, err := os.Stat(root)
	if err != nil {
		t.Fatal(err)
	}
	if !os.SameFile(fi1, fi2) {
		t.Fatalf("projectRoot() = %q; want: %q", root, want)
	}
}

func TestGenTablesRoot(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(wd, "../gentables")
	fi1, err := os.Stat(want)
	if err != nil {
		t.Fatal(err)
	}
	root, err := GenTablesRoot()
	if err != nil {
		t.Fatal(err)
	}
	fi2, err := os.Stat(root)
	if err != nil {
		t.Fatal(err)
	}
	if want != root || !os.SameFile(fi1, fi2) {
		t.Fatalf("projectRoot() = %q; want: %q", root, want)
	}
}
