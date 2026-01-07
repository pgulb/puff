package puff

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

var Version = "v0.1.0"

// Repo holds info about repo that a binary can be installed from
type Repo struct {
	Path   string
	Desc   string
	Regexp string
}

// Metadata holds info about a single installed binary and its version
type Metadata struct {
	Path      string   `json:"path"`
	Version   string   `json:"version"`
	NameParts []string `json:"name_parts"`
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
		{
			Path:   "charmbracelet/glow",
			Desc:   "Render markdown on the CLI, with pizzazz!",
			Regexp: `Linux_x86_64\.tar\.gz$`,
		},
		{
			Path:   "nektos/act",
			Desc:   "Run your GitHub Actions locally",
			Regexp: `^act_Linux_x86_64\.tar\.gz$`,
		},
		{
			Path:   "coreos/butane",
			Desc:   "Butane translates human-readable Butane Configs into machine-readable Ignition Configs.",
			Regexp: `^butane-x86_64-unknown-linux-gnu$`,
		},
		{
			Path:   "pkgforge-dev/ghostty-appimage",
			Desc:   "AppImage for Ghostty Terminal Emulator",
			Regexp: `x86_64\.AppImage$`,
		},
		{
			Path:   "go-task/task",
			Desc:   "A task runner / simpler Make alternative written in Go",
			Regexp: `^task_linux_amd64\.tar\.gz$`,
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
	fmt.Println("saving metadata.json")
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

// add Repo to MetadataList if not present,
// returns false if no changes
func AddMetaIfNotExists(
	metadata *MetadataList,
	repo *Repo,
	release *Release,
	nameParts []string,
) (bool, error) {
	for i := range metadata.Metadata {
		if metadata.Metadata[i].Path == repo.Path {
			// no action required
			if metadata.Metadata[i].Version == release.Version {
				fmt.Println("no action required")
				return false, nil
			}
			// update version in metadata entry
			fmt.Println("new version found, updating metadata")
			metadata.Metadata[i].Version = release.Version
			return true, nil
		}
	}
	// add new entry if not found
	fmt.Println("adding new metadata")
	metadata.Metadata = append(metadata.Metadata, Metadata{
		Path:      repo.Path,
		Version:   release.Version,
		NameParts: nameParts,
	})
	return true, nil
}

// extract binary name from repo path
func BinNameFromPath(repo *Repo) (string, error) {
	splitted := strings.Split(repo.Path, "/")
	if len(splitted) == 0 {
		return "", errors.New("invalid repo path")
	}
	return splitted[len(splitted)-1], nil
}

// ask user for name parts to search in binary name in release
func PromptForNameParts() []string {
	fmt.Println("Provide string to search in binary name to pick:")
	fmt.Println("(You can add more strings afterward)")
	var NameParts []string
	for {
		var name string
		fmt.Scanln(&name)
		if name == "" {
			break
		}
		NameParts = append(NameParts, name)
		fmt.Println("Provide another string or press enter:")
	}
	return NameParts
}

// searches for custom repo in metadata
func IsCustomRepoAdded(metadata *MetadataList, path string) Metadata {
	for _, v := range metadata.Metadata {
		if v.Path == path {
			return v
		}
	}
	return Metadata{}
}
