package puff

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Repo holds info about repo that a binary can be installed from
type Repo struct {
	Path   string
	Desc   string
	Regexp string
}

// Metadata holds info about a single installed binary and its version
type Metadata struct {
	Path    string `json:"path"`
	Version string `json:"version"`
}

// MetadataList is used to store all installed bins' metadata on disk
type MetadataList struct {
	Metadata []Metadata `json:"metadata"`
}

// returns fixed list of pre-added repositories
func AvailableRepos() *[]Repo {
	return &[]Repo{
		{
			Path:   "pgulb/plasma",
			Desc:   "Docker container controller with own HTTP API",
			Regexp: `\blinux-amd64\b`,
		},
	}
}

// reads metadata from metadata.json file
func GetMetadata(cfgDir string) (*MetadataList, error) {
	metadataFile := filepath.Join(cfgDir, "metadata.json")
	data, err := os.ReadFile(metadataFile)
	if err != nil {
		return &MetadataList{}, err
	}
	var metadata MetadataList
	err = json.Unmarshal(data, &metadata)
	if err != nil {
		return &MetadataList{}, err
	}
	return &metadata, nil
}

// stores MetadataList into metadata.json
func SaveMetadata(meta *MetadataList, cfgDir string) error {
	metadataFile := filepath.Join(cfgDir, "metadata.json")
	data, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return err
	}
	err = os.WriteFile(metadataFile, data, 0600)
	if err != nil {
		return err
	}
	return nil
}
