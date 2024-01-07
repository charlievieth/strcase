package main

import (
	"archive/tar"
	"bufio"
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	urlpkg "net/url"
	"os"
	"os/signal"
	"path"
	"path/filepath"
	"sync"
)

func init() {
	log.SetFlags(log.Lshortfile)
	log.SetOutput(os.Stderr)
}

var copyBuf = sync.Pool{}

func DownloadFilename(url string) (string, error) {
	u, err := urlpkg.Parse(url)
	if err != nil {
		return "", err
	}
	return path.Base(u.Path), nil
}

func readResponse(res *http.Response) ([]byte, error) {
	n := 32 * 1024
	if res.ContentLength > 0 && int64(int(res.ContentLength)) == res.ContentLength {
		n = int(res.ContentLength)
	}
	var buf bytes.Buffer
	buf.Grow(n)
	if _, err := buf.ReadFrom(res.Body); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func closeResponse(res *http.Response) {
	if res != nil && res.Body != nil {
		io.Copy(io.Discard, res.Body)
		res.Body.Close()
	}
}

// TODO: cache created directories (this speeds things up when dealing with
// slow filesystems)
//
// var knownDirs map[string]struct{}
//
// func MkdirAll(path string, perm os.FileMode) error {
// 	if _, ok := knownDirs[path]; ok {
// 		return nil
// 	}
// 	if err := os.MkdirAll(path, perm); err != nil {
// 		return err
// 	}
// 	if knownDirs == nil {
// 		knownDirs = make(map[string]struct{})
// 	}
// 	knownDirs[path] = struct{}{}
// 	return nil
// }

func Download(ctx context.Context, url, dirname string) error {
	if err := os.MkdirAll(dirname, 0755); err != nil {
		return err
	}
	filename, err := DownloadFilename(url)
	if err != nil {
		return err
	}
	filename = filepath.Join(dirname, filename)
	out, err := os.OpenFile(filename, os.O_EXCL|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer out.Close()

	exit := func(err error) error {
		out.Close()
		os.Remove(filename)
		return err
	}

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return exit(err)
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return exit(err)
	}
	defer closeResponse(res)

	if res.StatusCode != 200 {
		return fmt.Errorf("GET: %s: returned status code: %d",
			res.Request.URL, res.StatusCode)
	}

	if _, err := io.Copy(out, res.Body); err != nil {
		return exit(err)
	}
	if err := res.Body.Close(); err != nil {
		return exit(err)
	}
	if err := out.Close(); err != nil {
		return exit(err)
	}
	return nil
}

// // TODO: use this
// func copyHeader2(tr *tar.Reader, hdr *tar.Header, root string) error {
// 	// TODO: use os.O_EXCL ???
// 	const flag = os.O_CREATE | os.O_TRUNC | os.O_WRONLY
// 	path := filepath.Join(root, hdr.Name)
// 	f, err := os.OpenFile(path, flag, hdr.FileInfo().Mode())
// 	if err != nil {
// 		if !os.IsNotExist(err) {
// 			return err
// 		}
// 		// Lazily attempt to create the directory
// 		if err2 := os.MkdirAll(filepath.Dir(path), 0755); err2 != nil {
// 			return err2
// 		}
// 		f, err = os.OpenFile(path, flag, hdr.FileInfo().Mode())
// 		if err != nil {
// 			return err
// 		}
// 	}
// 	if _, err := io.Copy(f, tr); err != nil {
// 		f.Close()
// 		return err
// 	}
// 	return f.Close()
// }

var knownDirs = make(map[string]bool)

func mkdirAll(path string, perm os.FileMode) error {
	if knownDirs[path] {
		return nil
	}
	if err := os.MkdirAll(path, perm); err != nil {
		return err
	}
	for {
		knownDirs[path] = true
		d := filepath.Dir(path)
		if d == path {
			break
		}
		path = d
	}
	return nil
}

func copyHeader(tr *tar.Reader, hdr *tar.Header, root string) error {
	path := filepath.Join(root, hdr.Name)
	if err := mkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, hdr.FileInfo().Mode())
	if err != nil {
		return err
	}
	if _, err := io.Copy(f, tr); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}

func ExtractTarball(ctx context.Context, tarfile, root string) error {
	tarfile = filepath.Clean(tarfile)
	root = filepath.Clean(root)

	switch ext := filepath.Ext(tarfile); ext {
	case ".tgz", ".gz":
	default:
		return fmt.Errorf("tar: unsupported file extension: %q", ext)
	}

	if err := mkdirAll(root, 0755); err != nil {
		return err
	}

	fi, err := os.Open(tarfile)
	if err != nil {
		return err
	}
	defer fi.Close()

	gr, err := gzip.NewReader(bufio.NewReaderSize(fi, 96*1024))
	if err != nil {
		return err
	}
	defer gr.Close()

	tr := tar.NewReader(gr)
	for {
		hdr, err := tr.Next()
		if err != nil {
			if err != io.EOF {
				return err
			}
			break
		}
		switch hdr.Typeflag {
		case tar.TypeReg, tar.TypeRegA:
			if err := copyHeader(tr, hdr, root); err != nil {
				return err
			}
		case tar.TypeDir:
			path := filepath.Join(root, hdr.Name)
			if err := mkdirAll(path, hdr.FileInfo().Mode()); err != nil {
				return err
			}
		default:
			return fmt.Errorf("tar: unsupported type %x and mode %s: %q",
				hdr.Typeflag, hdr.FileInfo().Mode(), hdr.Name)
		}
		// Check if the context is cancelled
		select {
		case <-ctx.Done():
			log.Println("tar: context canceled removing up root directory:", root)
			if err := os.RemoveAll(root); err != nil {
				log.Println("tar: error removing root directory:", err)
			}
			return ctx.Err()
		default:
		}
	}
	return nil
}

// TODO: move this to its own package

type GoFileResponse struct {
	Filename string `json:"filename"`
	OS       string `json:"os"`
	Arch     string `json:"arch"`
	Version  string `json:"version"`
	Sha256   string `json:"sha256"`
	Size     int    `json:"size"`
	Kind     string `json:"kind"`
}

type GoVersionResponse struct {
	Version string           `json:"version"`
	Stable  bool             `json:"stable"`
	Files   []GoFileResponse `json:"files"`
}

func FetchGoVersions(ctx context.Context) ([]GoVersionResponse, error) {
	res, err := http.Get("https://go.dev/dl/?mode=json")
	if err != nil {
		return nil, err
	}
	defer closeResponse(res)

	if res.StatusCode != 200 {
		return nil, fmt.Errorf("GET: %s: returned status code: %d",
			res.Request.URL, res.StatusCode)
	}

	data, err := readResponse(res)
	if err != nil {
		return nil, err
	}

	var versions []GoVersionResponse
	if err := json.Unmarshal(data, &versions); err != nil {
		return nil, err
	}
	if len(versions) == 0 {
		return nil, fmt.Errorf("GET: %s: empty response", res.Request.URL)
	}
	return versions, nil
}

// TODO: check if we can use the installed go version
func LatestGoVersion(ctx context.Context) (string, error) {
	res, err := http.Get("https://go.dev/dl/?mode=json")
	if err != nil {
		return "", err
	}
	defer closeResponse(res)

	if res.StatusCode != 200 {
		return "", fmt.Errorf("GET: %s: returned status code: %d",
			res.Request.URL, res.StatusCode)
	}

	data, err := readResponse(res)
	if err != nil {
		return "", err
	}

	// TODO: do we want to return the download URL as well?
	var versions []GoVersionResponse
	if err := json.Unmarshal(data, &versions); err != nil {
		return "", err
	}
	// if len(versions) != 1 {
	// 	return "", fmt.Errorf("expected 1 Go version result got: %d", len(versions))
	// }
	return versions[0].Version, nil
}

// https://go.dev/dl/go1.20.7.linux-amd64.tar.gz

// Go 1.20 version - used for Unicode version 13.0.0 tables
const Go120 = "go1.20.7"

// WARN: rename
var tags = []struct{ version, buildTags string }{
	{"9.0.0", "!go1.10"},
	{"10.0.0", "go1.10,!go1.13"},
	{"11.0.0", "go1.13,!go1.14"},
	{"12.0.0", "go1.14,!go1.16"},
	{"13.0.0", "go1.16,!go1.21"},
	{"15.0.0", "go1.21"},
}

// TODO: take Unicode version as the argument and download the Go version for that
func DownloadGo(unicodeVersion string) {
	panic("IMPLEMENT")
}

func main() {
	{
		dir := "/Users/cvieth/go/src/github.com/charlievieth/strcase/internal/downloadgo/downloadgo.go"
		for {
			fmt.Println(dir)
			d := filepath.Dir(dir)
			if d == dir {
				break
			}
			dir = d
		}
		return
	}

	// {
	// 	_, err := os.Create("/Users/cvieth/go/src/github.com/charlievieth/strcase/internal/downloadgo/x/y/x.go")
	// 	fmt.Println(err)
	// 	fmt.Println(os.IsNotExist(err))
	// 	return
	// }

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	go func() {
		<-ctx.Done()
		log.Println("signaled: cleaning up and exiting")
		stop()
	}()

	versions, err := FetchGoVersions(ctx)
	if err != nil {
		log.Fatal(err)
	}
	go120 := versions[1]
	a := go120.Files
	for _, f := range go120.Files {
		if f.Kind == "archive" {
			a = append(a, f)
		}
	}
	go120.Files = a
	data, err := json.MarshalIndent(go120, "", "\t")
	if err != nil {
		log.Fatal(err)
	}
	if err := os.WriteFile("goversions/go1.20.json", data, 0644); err != nil {
		log.Fatal(err)
	}
	PrintJSON(versions)
	return

	// Real download link:
	// https://dl.google.com/go/go1.21.0.darwin-arm64.tar.gz
	if err := Download(ctx, "https://go.dev/dl/go1.21.0.darwin-arm64.tar.gz", "tmp"); err != nil {
		log.Fatal(err)
	}
	if err := ExtractTarball(ctx, "tmp/go1.21.0.darwin-arm64.tar.gz", "tmp/go1.21.0"); err != nil {
		log.Fatal(err)
	}
	// // https://go.dev/dl/go1.21.0.darwin-arm64.pkg
	// u, err := urlpkg.Parse("https://go.dev/dl/go1.21.0.darwin-arm64.pkg")
	//
	//	if err != nil {
	//		log.Fatal(err)
	//	}
	//
	// fmt.Println(u.Path)
}

func PrintJSON(v interface{}) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "    ")
	return enc.Encode(v)
}
