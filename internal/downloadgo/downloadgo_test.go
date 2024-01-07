package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io/fs"
	"path/filepath"
	"sort"
	"strconv"
	"testing"
	"time"
)

// TODO: consider using this to record the directory state to detect changes
// WARN: rename
func MakeDirRecord(root string) (string, error) {
	type Info struct {
		Name    string
		Size    int64
		ModTime time.Time
	}
	var files []Info
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		name, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		if d.Type().IsRegular() {
			fi, err := d.Info()
			if err != nil {
				return err
			}
			files = append(files, Info{Name: name, Size: fi.Size(), ModTime: fi.ModTime()})
		}
		return nil
	})
	sort.Slice(files, func(i, j int) bool {
		return files[i].Name < files[j].Name
	})
	h := sha256.New()
	var b []byte
	for _, f := range files {
		b = b[:0]
		b = append(b, f.Name...)
		b = strconv.AppendInt(b, f.ModTime.UnixNano(), 10)
		b = strconv.AppendInt(b, f.Size, 10)
		h.Write(b)
	}
	return hex.EncodeToString(h.Sum(nil)), err
}

func BenchmarkMakeDirRecord(b *testing.B) {
	for i := 0; i < b.N; i++ {
		MakeDirRecord("tmp/go1.21.0")
	}
}

func BenchmarkExtractTarball(b *testing.B) {
	tmp := b.TempDir()
	ctx := context.Background()
	for i := 0; i < b.N; i++ {
		if err := ExtractTarball(ctx, "tmp/go1.21.0.darwin-arm64.tar.gz", tmp); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkMkdirAll(b *testing.B) {
	tmp := filepath.Join(b.TempDir(), "a/b/c/d/e/f")
	for i := 0; i < b.N; i++ {
		// if err := os.MkdirAll(tmp, 0755); err != nil {
		if err := mkdirAll(tmp, 0755); err != nil {
			b.Fatal(err)
		}
	}
}
