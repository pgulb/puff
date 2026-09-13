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
	"strings"
	"sync"
	"testing"
	"time"
)

// trackedHandler records the maximum number of concurrent requests it serves.
type trackedHandler struct {
	mu        sync.Mutex
	active    int
	maxActive int
	respond   func(w http.ResponseWriter, r *http.Request)
}

func (h *trackedHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.mu.Lock()
	h.active++
	if h.active > h.maxActive {
		h.maxActive = h.active
	}
	h.mu.Unlock()
	time.Sleep(10 * time.Millisecond)
	if h.respond != nil {
		h.respond(w, r)
	}
}

func (h *trackedHandler) max() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.maxActive
}

// makeTarGz builds a .tar.gz archive containing a single file named fname
// with the given contents.
func makeTarGz(t *testing.T, fname string, contents []byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)
	if err := tw.WriteHeader(&tar.Header{Name: fname, Size: int64(len(contents))}); err != nil {
		t.Fatal(err)
	}
	if _, err := tw.Write(contents); err != nil {
		t.Fatal(err)
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestCheckReposParallel(t *testing.T) {
	defer resetHTTPVars(t)
	trk := &trackedHandler{}
	srv := httptest.NewServer(trk)
	defer srv.Close()
	ghAPIBase = srv.URL
	repos := []Repo{
		{Path: "owner/a", Regexp: `^a-v.*\.tar\.gz$`},
		{Path: "owner/b", Regexp: `^b-v.*\.tar\.gz$`},
		{Path: "owner/c", Regexp: `^c-v.*\.tar\.gz$`},
	}
	trk.respond = func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		// The request path is /repos/owner/<name>/releases/latest.
		parts := strings.Split(r.URL.Path, "/")
		name := parts[3] // "a", "b", or "c"
		_, _ = io.WriteString(w, `{"tag_name":"v1.0.0","assets":[{"name":"`+name+`-v1.0.0.tar.gz","browser_download_url":"https://example.com/`+name+`.tar.gz"}]}`)
	}
	plans := checkRepos(repos, "-")
	if len(plans) != 3 {
		t.Fatalf("expected 3 plans, got %d", len(plans))
	}
	for i, p := range plans {
		if p.err != nil {
			t.Fatalf("plan %d error: %v", i, p.err)
		}
		if p.release == nil || p.release.Version != "v1.0.0" {
			t.Fatalf("plan %d unexpected release: %+v", i, p.release)
		}
	}
	if trk.max() < 2 {
		t.Fatalf("expected parallel checks (maxActive >= 2), got %d", trk.max())
	}
}

func TestInstallFeaturedAllUpToDate(t *testing.T) {
	defer resetHTTPVars(t)
	var assetURL string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, `{"tag_name":"v1.0.0","assets":[{"name":"a-v1.0.0.tar.gz","browser_download_url":"`+assetURL+`"}]}`)
	}))
	assetURL = srv.URL + "/a-v1.0.0.tar.gz"
	defer srv.Close()
	ghAPIBase = srv.URL
	dir := tempCfgDir(t)
	meta := &MetadataList{Metadata: []Metadata{{Path: "owner/a", Version: "v1.0.0"}}}
	errs := installFeatured(dir, []Repo{{Path: "owner/a", Regexp: `^a-v.*\.tar\.gz$`}}, "-", meta, &sync.Mutex{}, nil)
	if len(errs) != 0 {
		t.Fatalf("expected no errors, got %v", errs)
	}
	entries, err := os.ReadDir(filepath.Join(dir, "bin"))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("expected empty bin/, got %d entries", len(entries))
	}
}

func TestInstallFeaturedSomeUpdate(t *testing.T) {
	defer resetHTTPVars(t)
	bin := []byte("binary content")
	archive := makeTarGz(t, "a", bin)
	var assetURL string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "releases/latest") {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = io.WriteString(w, `{"tag_name":"v2.0.0","assets":[{"name":"a-v2.0.0.tar.gz","browser_download_url":"`+assetURL+`"}]}`)
			return
		}
		w.Header().Set("Content-Length", itoa(len(archive)))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(archive)
	}))
	assetURL = srv.URL + "/a-v2.0.0.tar.gz"
	defer srv.Close()
	ghAPIBase = srv.URL
	dir := tempCfgDir(t)
	meta := &MetadataList{Metadata: []Metadata{{Path: "owner/a", Version: "v1.0.0"}}}
	errs := installFeatured(dir, []Repo{{Path: "owner/a", Regexp: `^a-v.*\.tar\.gz$`}}, "-", meta, &sync.Mutex{}, nil)
	if len(errs) != 0 {
		t.Fatalf("expected no errors, got %v", errs)
	}
	data, err := readBin(dir, "a")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(data, bin) {
		t.Fatalf("expected %q, got %q", bin, data)
	}
	if _, err := os.Stat(filepath.Join(dir, "bin", "a.tmp")); err == nil {
		t.Fatal("leftover .tmp file")
	}
	if len(meta.Metadata) != 1 || meta.Metadata[0].Version != "v2.0.0" {
		t.Fatalf("metadata not updated: %+v", meta.Metadata)
	}
}

func TestInstallFeaturedErrorCollected(t *testing.T) {
	defer resetHTTPVars(t)
	bin := []byte("binary content")
	archive := makeTarGz(t, "a", bin)
	var aURL, bURL string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.Contains(r.URL.Path, "releases/latest") && strings.Contains(r.URL.Path, "owner/a"):
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = io.WriteString(w, `{"tag_name":"v2.0.0","assets":[{"name":"a-v2.0.0.tar.gz","browser_download_url":"`+aURL+`"}]}`)
		case strings.Contains(r.URL.Path, "releases/latest") && strings.Contains(r.URL.Path, "owner/b"):
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = io.WriteString(w, `{"tag_name":"v2.0.0","assets":[{"name":"b-v2.0.0.tar.gz","browser_download_url":"`+bURL+`"}]}`)
		case strings.Contains(r.URL.Path, "a-v2.0.0"):
			w.Header().Set("Content-Length", itoa(len(archive)))
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(archive)
		default:
			w.WriteHeader(http.StatusInternalServerError)
		}
	}))
	aURL = srv.URL + "/a-v2.0.0.tar.gz"
	bURL = srv.URL + "/b-v2.0.0.tar.gz"
	defer srv.Close()
	ghAPIBase = srv.URL
	dir := tempCfgDir(t)
	meta := &MetadataList{Metadata: []Metadata{
		{Path: "owner/a", Version: "v1.0.0"},
		{Path: "owner/b", Version: "v1.0.0"},
	}}
	errs := installFeatured(dir, []Repo{
		{Path: "owner/a", Regexp: `^a-v.*\.tar\.gz$`},
		{Path: "owner/b", Regexp: `^b-v.*\.tar\.gz$`},
	}, "-", meta, &sync.Mutex{}, nil)
	if len(errs) != 1 {
		t.Fatalf("expected 1 error, got %d: %v", len(errs), errs)
	}
	// Repo "a" installed, "b" failed.
	data, err := readBin(dir, "a")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(data, bin) {
		t.Fatalf("expected install for a, got %q", data)
	}
	if _, err := os.Stat(filepath.Join(dir, "bin", "b")); err == nil {
		t.Fatal("repo b should not have installed")
	}
	// Metadata only reflects the successful install.
	if len(meta.Metadata) != 2 || meta.Metadata[0].Version != "v2.0.0" || meta.Metadata[1].Version != "v1.0.0" {
		t.Fatalf("unexpected metadata: %+v", meta.Metadata)
	}
}
