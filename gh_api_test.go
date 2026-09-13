package puff

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

// jsonServer returns a handler that serves the given JSON response with status OK.
func jsonServer(body string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, body)
	})
}

func TestGetLatestRelease(t *testing.T) {
	defer resetHTTPVars(t)

	srv := httptest.NewServer(jsonServer(`{
		"tag_name": "v1.2.3",
		"assets": [
			{"name": "repo-v1.2.3-linux_amd64.tar.gz", "browser_download_url": "https://example.com/repo.tar.gz"},
			{"name": "other-asset", "browser_download_url": "https://example.com/other"}
		]
	}`))
	defer srv.Close()

	ghAPIBase = srv.URL
	repo := &Repo{Path: "owner/repo", Regexp: `^repo-v.*-linux_amd64\.tar\.gz$`}

	release, err := GetLatestRelease(repo, "-")
	if err != nil {
		t.Fatal(err)
	}
	if release.Version != "v1.2.3" {
		t.Fatalf("expected v1.2.3, got %s", release.Version)
	}
	if release.Link != "https://example.com/repo.tar.gz" {
		t.Fatalf("unexpected link: %s", release.Link)
	}
}

func TestGetLatestReleaseNoMatchingAsset(t *testing.T) {
	defer resetHTTPVars(t)

	srv := httptest.NewServer(jsonServer(`{
		"tag_name": "v1.2.3",
		"assets": [
			{"name": "other-asset", "browser_download_url": "https://example.com/other"}
		]
	}`))
	defer srv.Close()

	ghAPIBase = srv.URL
	repo := &Repo{Path: "owner/repo", Regexp: `^repo-v.*-linux_amd64\.tar\.gz$`}

	if _, err := GetLatestRelease(repo, "-"); err == nil {
		t.Fatal("expected error for no matching asset")
	}
}

func TestGetLatestReleaseBadStatus(t *testing.T) {
	defer resetHTTPVars(t)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	ghAPIBase = srv.URL
	repo := &Repo{Path: "owner/repo", Regexp: `.*`}

	if _, err := GetLatestRelease(repo, "-"); err == nil {
		t.Fatal("expected error for non-OK status")
	}
}

func TestGetLatestReleaseAssets(t *testing.T) {
	defer resetHTTPVars(t)

	srv := httptest.NewServer(jsonServer(`{
		"tag_name": "v9.9.9",
		"assets": [
			{"name": "a", "browser_download_url": "https://example.com/a"},
			{"name": "b", "browser_download_url": "https://example.com/b"}
		]
	}`))
	defer srv.Close()

	ghAPIBase = srv.URL
	resp, err := GetLatestReleaseAssets("owner/repo", "-")
	if err != nil {
		t.Fatal(err)
	}
	if resp.Version != "v9.9.9" || len(resp.Assets) != 2 {
		t.Fatalf("unexpected response: %+v", resp)
	}
}

// rawServer serves raw bytes with the given Content-Length.
func rawServer(content []byte) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", itoa(len(content)))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(content)
	})
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var buf []byte
	for n > 0 {
		buf = append([]byte{byte('0' + n%10)}, buf...)
		n /= 10
	}
	return string(buf)
}

func TestDownloadBinaryPlain(t *testing.T) {
	defer resetHTTPVars(t)

	bin := []byte("hello puff")
	srv := httptest.NewServer(rawServer(bin))
	defer srv.Close()

	dir := tempCfgDir(t)
	repo := &Repo{Path: "owner/repo"}
	release := &Release{Link: srv.URL + "/repo"}

	if err := DownloadBinary(dir, repo, release, "-"); err != nil {
		t.Fatal(err)
	}

	data, err := readBin(dir, "repo")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(data, bin) {
		t.Fatalf("expected %q, got %q", bin, data)
	}
	if _, err := os.Stat(filepath.Join(dir, "bin", "repo.tmp")); err == nil {
		t.Fatal("leftover .tmp file")
	}
}

func TestDownloadBinaryTarGz(t *testing.T) {
	defer resetHTTPVars(t)

	// Build a tar.gz containing a file named "repo".
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)
	body := []byte("extracted binary content")
	if err := tw.WriteHeader(&tar.Header{Name: "repo", Size: int64(len(body))}); err != nil {
		t.Fatal(err)
	}
	if _, err := tw.Write(body); err != nil {
		t.Fatal(err)
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gw.Close(); err != nil {
		t.Fatal(err)
	}
	archive := buf.Bytes()

	srv := httptest.NewServer(rawServer(archive))
	defer srv.Close()

	dir := tempCfgDir(t)
	repo := &Repo{Path: "owner/repo"}
	release := &Release{Link: srv.URL + "/repo.tar.gz"}

	if err := DownloadBinary(dir, repo, release, "-"); err != nil {
		t.Fatal(err)
	}

	data, err := readBin(dir, "repo")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(data, body) {
		t.Fatalf("expected %q, got %q", body, data)
	}
}

func TestDownloadBinaryHTTPError(t *testing.T) {
	defer resetHTTPVars(t)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	dir := tempCfgDir(t)
	repo := &Repo{Path: "owner/repo"}
	release := &Release{Link: srv.URL + "/repo"}

	if err := DownloadBinary(dir, repo, release, "-"); err == nil {
		t.Fatal("expected error for non-OK status")
	}
}