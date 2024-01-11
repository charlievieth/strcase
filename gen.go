//go:build gen
// +build gen

package main

import (
	"bytes"
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

func buildGen() string {
	gendir := filepath.Join(projectRoot(), "internal/gentables")
	if _, err := os.Stat(gendir); err != nil {
		log.Fatal(err)
	}

	exe := filepath.Join(projectRoot(), "bin", "gentables")
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
	return exe
}

func realMain(args []string) int {
	root := projectRoot()

	exe := buildGen()

	// TODO: supporting Unicode version 12.0.0 is annoying since arm64 support
	// is lacking on Go 1.15 and below.
	var exitcode int
	for _, version := range []string{"13.0.0", "15.0.0"} {
		cmd := exec.Command(exe, append([]string{"-unicode", version}, args...)...)
		cmd.Dir = root
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			log.Println("error running command %q: %v", cmd.Args, err)
			exitcode++
		}
	}
	return exitcode
}

func main() {
	log.SetPrefix("gen: ")
	log.SetFlags(log.Lshortfile)
	if code := realMain(os.Args[1:]); code != 0 {
		log.Fatal("exit:", code)
	}
}
