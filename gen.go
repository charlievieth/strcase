//go:build gen
// +build gen

package main

import (
	"bytes"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"
)

var projectRoot = sync.OnceValue(func() string {
	cmd := exec.Command("go", "list", "-f", "{{.Root}}", "github.com/charlievieth/strcase")
	out, err := cmd.CombinedOutput()
	if err != nil {
		log.Fatalf("error running command %q: %v\n\n%s\n",
			cmd.Args, err, bytes.TrimSpace(out))
	}
	dir := string(bytes.TrimSpace(out))
	if _, err := os.Stat(dir); err != nil {
		log.Fatal(err)
	}
	return dir
})

func buildGen() (string, func()) {
	gendir := filepath.Join(projectRoot(), "internal/gentables")
	if _, err := os.Stat(gendir); err != nil {
		log.Fatal(err)
	}

	dir, err := os.MkdirTemp("", "strcase-gen-*")
	if err != nil {
		log.Fatal(err)
	}
	exe := filepath.Join(dir, "gentables")
	if runtime.GOOS == "windows" {
		exe += ".exe"
	}

	cmd := exec.Command("go", "build", "-o", exe)
	cmd.Dir = gendir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		log.Fatalf("error running command %q: %v", cmd.Args, err)
	}
	return exe, func() { os.RemoveAll(dir) }
}

func realMain() error {
	root := projectRoot()

	exe, fn := buildGen()
	defer fn()

	// TODO: supporting Unicode version 12.0.0 is annoying since arm64 support
	// is lacking on Go 1.15 and below.
	for _, version := range []string{"13.0.0", "15.0.0"} {
		cmd := exec.Command(exe, "-unicode", version)
		cmd.Dir = root
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("error running command %q: %v", cmd.Args, err)
		}
	}
	return nil
}

func main() {
	log.SetFlags(log.Lshortfile)
	flag.Usage = func() {
		const msg = "Usage: %s\nGenerate Unicode tables for strcase.\n"
		fmt.Fprintf(flag.CommandLine.Output(), msg, filepath.Base(os.Args[0]))
		flag.PrintDefaults()
	}
	flag.Parse()
	if err := realMain(); err != nil {
		log.Fatal(err)
	}
}
