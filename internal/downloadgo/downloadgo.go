package main

import (
	"archive/tar"
	"bufio"
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	urlpkg "net/url"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
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
	n := 4096
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

func copyHeader(tr *tar.Reader, hdr *tar.Header, root string) error {
	path := filepath.Join(root, hdr.Name)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY, hdr.FileInfo().Mode())
	if err != nil {
		return err
	}
	if _, err := io.Copy(f, tr); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}

func ExtractTarball(tarfile, root string) error {
	if err := os.MkdirAll(root, 0755); err != nil {
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
		path := filepath.Join(root, hdr.Name)
		switch mode := hdr.FileInfo().Mode(); {
		case mode&os.ModeSymlink != 0:
			return errors.New("symlinks are not supported: " + path)
		case mode.IsDir():
			if err := os.MkdirAll(path, hdr.FileInfo().Mode()); err != nil {
				return err
			}
		case mode.IsRegular():
			if err := copyHeader(tr, hdr, root); err != nil {
				return err
			}
		}
	}
	return nil
}

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

func tablesFilename() (string, error) {
	v := runtime.Version()
	for _, s := range []string{"go1.18", "go1.19", "go1.20"} {
		if strings.HasPrefix(v, s) {
			return "tables_go120.go", nil
		}
	}
	for _, s := range []string{"go1.21"} {
		if strings.HasPrefix(v, s) {
			return "tables_go121.go", nil
		}
	}
	return "", fmt.Errorf("unsupported go version: %q", v)
}

func main() {
	{
		fmt.Println(tablesFilename())
		return
	}
	ctx := context.Background()
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
	if err := ExtractTarball("tmp/go1.21.0.darwin-arm64.tar.gz", "tmp/go1.21.0"); err != nil {
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
