package util

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/mod/modfile"
)

func modfilePath(name string) (string, error) {
	data, err := os.ReadFile(name)
	if err != nil {
		return "", err
	}
	file, err := modfile.Parse(name, data, nil)
	if err != nil {
		return "", err
	}
	if file == nil || file.Module == nil || file.Module.Mod.Path == "" {
		return "", errors.New("util: missing module path: " + name)
	}
	return file.Module.Mod.Path, nil
}

func findModfile(child, pkgPath string) (string, error) {
	if !filepath.IsAbs(child) {
		return child, errors.New("directory must be absolute: " + child)
	}
	var first error
	dir := filepath.Clean(child)
	for {
		if _, err := os.Stat(dir + "/go.mod"); err == nil {
			path := filepath.Join(dir, "go.mod")
			pkg, err := modfilePath(path)
			if err != nil {
				if first == nil {
					first = err
				}
				continue
			}
			if pkg == pkgPath {
				return dir, nil
			}
		}
		parent := filepath.Dir(dir)
		if len(parent) >= len(dir) {
			break
		}
		dir = parent
	}
	if first != nil {
		return child, fmt.Errorf("util: error finding go.mod for package %q "+
			"in directory: %q: %w", pkgPath, child, first)
	}
	return child, fmt.Errorf("util: failed to find go.mod for package %q "+
		"in directory: %q", pkgPath, child)
}

func ProjectRoot() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	dir, err := findModfile(wd, "github.com/charlievieth/strcase")
	if err != nil {
		return "", err
	}
	return dir, nil
}

func GenTablesRoot() (string, error) {
	root, err := ProjectRoot()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(root, "internal/gen")
	if _, err := os.Stat(dir); err != nil {
		return "", err
	}
	return dir, nil
}
