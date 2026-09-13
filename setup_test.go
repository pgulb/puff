package puff

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestGetGhPat(t *testing.T) {
	dir := t.TempDir()

	// Missing file -> empty string, no error.
	pat, err := GetGhPat(dir)
	if err != nil || pat != "" {
		t.Fatalf("expected empty pat, got %q, %v", pat, err)
	}

	// Existing file -> its content.
	if err := os.WriteFile(filepath.Join(dir, "gh_pat"), []byte("ghp_token"), 0600); err != nil {
		t.Fatal(err)
	}
	pat, err = GetGhPat(dir)
	if err != nil || pat != "ghp_token" {
		t.Fatalf("expected ghp_token, got %q, %v", pat, err)
	}
}

func TestMaybeCreateMetadata(t *testing.T) {
	dir := t.TempDir()

	// No file -> creates skeleton.
	if err := MaybeCreateMetadata(dir); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(dir, "metadata.json"))
	if err != nil {
		t.Fatal(err)
	}
	var meta MetadataList
	if err := json.Unmarshal(data, &meta); err != nil {
		t.Fatal(err)
	}
	if len(meta.Metadata) != 0 {
		t.Fatalf("expected empty skeleton, got %+v", meta.Metadata)
	}

	// Second call -> no-op, file still valid.
	if err := MaybeCreateMetadata(dir); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "metadata.json")); err != nil {
		t.Fatalf("metadata.json missing: %v", err)
	}

	// Pre-existing file -> untouched.
	if err := os.WriteFile(filepath.Join(dir, "metadata.json"), []byte(`{"metadata":[{"path":"owner/repo","version":"v1.0.0"}]}`), 0600); err != nil {
		t.Fatal(err)
	}
	if err := MaybeCreateMetadata(dir); err != nil {
		t.Fatal(err)
	}
	got, err := GetMetadata(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Metadata) != 1 || got.Metadata[0].Path != "owner/repo" || got.Metadata[0].Version != "v1.0.0" {
		t.Fatalf("pre-existing file was overwritten: %+v", got.Metadata)
	}
}

func TestMustCreateBinDir(t *testing.T) {
	dir := t.TempDir()

	if err := MustCreateBinDir(dir); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(filepath.Join(dir, "bin"))
	if err != nil || !info.IsDir() {
		t.Fatalf("bin dir missing: %v", err)
	}

	// Idempotent.
	if err := MustCreateBinDir(dir); err != nil {
		t.Fatal(err)
	}
}

func TestWasPromptedForPath(t *testing.T) {
	dir := t.TempDir()

// Missing -> false.
	prompted, err := WasPromptedForPath(dir)
	if err != nil || prompted {
		t.Fatalf("expected false, got %v, %v", prompted, err)
	}

	// Present -> true.
	if err := os.WriteFile(filepath.Join(dir, "path_asked"), []byte(""), 0600); err != nil {
		t.Fatal(err)
	}
	prompted, err = WasPromptedForPath(dir)
	if err != nil || !prompted {
		t.Fatalf("expected true, got %v, %v", prompted, err)
	}
}