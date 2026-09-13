package puff

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestNeedsUpdate(t *testing.T) {
	repo := &Repo{Path: "owner/repo"}
	release := &Release{Version: "v1.0.0"}

	// Missing entry -> needs update.
	if !needsUpdate(&MetadataList{}, repo, release) {
		t.Fatal("missing entry should need update")
	}

	// Same version -> no update.
	meta := &MetadataList{Metadata: []Metadata{{Path: "owner/repo", Version: "v1.0.0"}}}
	if needsUpdate(meta, repo, release) {
		t.Fatal("same version should not need update")
	}

	// Different version -> update.
	meta.Metadata[0].Version = "v0.9.0"
	if !needsUpdate(meta, repo, release) {
		t.Fatal("different version should need update")
	}
}

func TestAddMetaIfNotExists(t *testing.T) {
	repo := &Repo{Path: "owner/repo"}
	release := &Release{Version: "v1.0.0"}

	// New repo is appended.
	meta := &MetadataList{}
	added, err := AddMetaIfNotExists(meta, repo, release, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !added {
		t.Fatal("expected added=true for new repo")
	}
	if len(meta.Metadata) != 1 || meta.Metadata[0].Path != "owner/repo" || meta.Metadata[0].Version != "v1.0.0" {
		t.Fatalf("unexpected metadata: %+v", meta.Metadata)
	}

	// Same version -> no change.
	added, err = AddMetaIfNotExists(meta, repo, release, nil)
	if err != nil {
		t.Fatal(err)
	}
	if added {
		t.Fatal("expected added=false for same version")
	}
	if len(meta.Metadata) != 1 {
		t.Fatal("metadata should not have grown")
	}

	// Different version -> version updated in place.
	release.Version = "v2.0.0"
	added, err = AddMetaIfNotExists(meta, repo, release, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !added {
		t.Fatal("expected added=true for version bump")
	}
	if meta.Metadata[0].Version != "v2.0.0" {
		t.Fatalf("expected version v2.0.0, got %s", meta.Metadata[0].Version)
	}
	if len(meta.Metadata) != 1 {
		t.Fatal("metadata should not have grown on version bump")
	}
}

func TestBinNameFromPath(t *testing.T) {
	repo := &Repo{Path: "owner/repo"}
	name, err := BinNameFromPath(repo)
	if err != nil || name != "repo" {
		t.Fatalf("expected repo, got %q, %v", name, err)
	}

	repo = &Repo{Path: "a/b/c"}
	name, err = BinNameFromPath(repo)
	if err != nil || name != "c" {
		t.Fatalf("expected c, got %q, %v", name, err)
	}

	// Single-segment path returns that segment.
	repo = &Repo{Path: "puff"}
	name, err = BinNameFromPath(repo)
	if err != nil || name != "puff" {
		t.Fatalf("expected puff, got %q, %v", name, err)
	}
}

func TestIsCustomRepoAdded(t *testing.T) {
	meta := &MetadataList{Metadata: []Metadata{{Path: "owner/repo", Version: "v1.0.0"}}}
	if got := IsCustomRepoAdded(meta, "owner/repo"); got.Version != "v1.0.0" {
		t.Fatalf("expected entry, got %+v", got)
	}
	if got := IsCustomRepoAdded(meta, "missing/repo"); got.Version != "" {
		t.Fatalf("expected zero value, got %+v", got)
	}
}

func TestSaveMetadataIsAtomic(t *testing.T) {
	dir := t.TempDir()
	meta := &MetadataList{Metadata: []Metadata{{Path: "owner/repo", Version: "v1.0.0"}}}

	if err := SaveMetadata(meta, dir); err != nil {
		t.Fatal(err)
	}

	final := filepath.Join(dir, "metadata.json")
	if _, err := os.Stat(final); err != nil {
		t.Fatalf("final file missing: %v", err)
	}
	if _, err := os.Stat(final + ".tmp"); err == nil {
		t.Fatal("leftover .tmp file")
	}

	data, err := os.ReadFile(final)
	if err != nil {
		t.Fatal(err)
	}
	var loaded MetadataList
	if err := json.Unmarshal(data, &loaded); err != nil {
		t.Fatal(err)
	}
	if len(loaded.Metadata) != 1 || loaded.Metadata[0].Path != "owner/repo" || loaded.Metadata[0].Version != "v1.0.0" {
		t.Fatalf("unexpected content: %+v", loaded.Metadata)
	}

	info, err := os.Stat(final)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0600 {
		t.Fatalf("expected 0600 perms, got %v", info.Mode().Perm())
	}
}

func TestGetMetadataRoundTrip(t *testing.T) {
	dir := tempCfgDir(t)
	meta := &MetadataList{Metadata: []Metadata{
		{Path: "owner/a", Version: "v1.0.0", NameParts: []string{"a"}},
		{Path: "owner/b", Version: "v2.0.0"},
	}}
	if err := SaveMetadata(meta, dir); err != nil {
		t.Fatal(err)
	}

	got, err := GetMetadata(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Metadata) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(got.Metadata))
	}
	if got.Metadata[0].Path != "owner/a" || got.Metadata[0].Version != "v1.0.0" || got.Metadata[1].Path != "owner/b" || got.Metadata[1].Version != "v2.0.0" {
		t.Fatalf("unexpected content: %+v", got.Metadata)
	}
}