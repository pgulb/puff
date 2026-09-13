package puff

import (
	"net/http"
	"os"
	"path/filepath"
	"testing"
)

// tempCfgDir returns a temporary config directory with bin/ created and a
// skeleton metadata.json written, suitable for testing install/update flows.
func tempCfgDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "bin"), 0750); err != nil {
		t.Fatalf("mkdir bin: %v", err)
	}
	if err := SaveMetadata(&MetadataList{}, dir); err != nil {
		t.Fatalf("save skeleton metadata: %v", err)
	}
	return dir
}

// resetHTTPVars restores the default HTTP seam after a test that overrode it.
func resetHTTPVars(t *testing.T) {
	t.Helper()
	httpClient = &http.Client{}
	ghAPIBase = "https://api.github.com"
}

// readBin reads the installed binary from the bin directory.
func readBin(dir, name string) ([]byte, error) {
	return os.ReadFile(filepath.Join(dir, "bin", name))
}