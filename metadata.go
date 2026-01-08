package puff

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

var Version = "v0.5.0"

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
		{
			Path:   "eza-community/eza",
			Desc:   "A modern alternative to ls",
			Regexp: `^eza_x86_64-unknown-linux-musl\.tar\.gz$`,
		},
		{
			Path:   "starship/starship",
			Desc:   "The minimal, blazing-fast, and infinitely customizable prompt for any shell!",
			Regexp: `^starship-x86_64-unknown-linux-musl\.tar\.gz$`,
		},
		{
			Path:   "bootandy/dust",
			Desc:   "A more intuitive version of du in rust",
			Regexp: `^dust-v.*-x86_64-unknown-linux-musl\.tar\.gz$`,
		},
		{
			Path:   "mikefarah/yq",
			Desc:   "yq is a portable command-line YAML, JSON, XML, CSV, TOML, HCL and properties processor",
			Regexp: `^yq_linux_amd64$`,
		},
		{
			Path:   "jesseduffield/lazydocker",
			Desc:   "The lazier way to manage everything docker",
			Regexp: `^lazydocker_.*_Linux_x86_64\.tar\.gz$`,
		},
		{
			Path:   "jesseduffield/lazygit",
			Desc:   "simple terminal UI for git commands",
			Regexp: `^lazygit_.*_linux_x86_64\.tar\.gz$`,
		},
		{
			Path:   "fastfetch-cli/fastfetch",
			Desc:   "A maintained, feature-rich and performance oriented, neofetch like system information tool",
			Regexp: `^fastfetch-linux-amd64\.tar\.gz$`,
		},
		{
			Path:   "sharkdp/fd",
			Desc:   "Simple, fast and user-friendly alternative to find",
			Regexp: `^fd-.*-x86_64-unknown-linux-musl\.tar\.gz$`,
		},
		{
			Path:   "stedolan/jq",
			Desc:   "Lightweight and flexible command-line JSON processor",
			Regexp: `^jq-linux64$`,
		},
		{
			Path:   "dbrgn/tealdeer",
			Desc:   "A fast tldr client for simplified and community-driven man pages",
			Regexp: `^tealdeer-linux-x86_64-musl$`,
		},
		{
			Path:   "ducaale/xh",
			Desc:   "Friendly and fast tool for sending HTTP requests",
			Regexp: `^xh-.*-x86_64-unknown-linux-musl\.tar\.gz$`,
		},
		{
			Path:   "sharkdp/bat",
			Desc:   "A cat clone with syntax highlighting and Git integration",
			Regexp: `^bat-v.*-x86_64-unknown-linux-musl\.tar\.gz$`,
		},
		{
			Path:   "Y2Z/monolith",
			Desc:   "Save complete web pages as single HTML files",
			Regexp: `^monolith-gnu-linux-x86_64$`,
		},
		{
			Path:   "topgrade-rs/topgrade",
			Desc:   "Upgrade all tools on the system",
			Regexp: `^topgrade-v\d+\.\d+\.\d+-x86_64-unknown-linux-musl\.tar\.gz$`,
		},
		{
			Path:   "fullstorydev/grpcurl",
			Desc:   "A command-line tool for interacting with gRPC servers",
			Regexp: `^grpcurl_.*_linux_x86_64\.tar\.gz$`,
		},
		{
			Path:   "derailed/k9s",
			Desc:   "Kubernetes CLI to manage your clusters in style",
			Regexp: `^k9s_Linux_amd64\.tar\.gz$`,
		},
		{
			Path:   "astral-sh/uv",
			Desc:   "An extremely fast Python package installer and resolver, written in Rust.",
			Regexp: `^uv-x86_64-unknown-linux-gnu\.tar\.gz$`,
		},
		{
			Path:   "astral-sh/ruff",
			Desc:   "An extremely fast Python linter and code formatter, written in Rust.",
			Regexp: `^ruff-x86_64-unknown-linux-gnu\.tar\.gz$`,
		},
		{
			Path:   "astral-sh/ty",
			Desc:   "Static type checker for Python",
			Regexp: `^ty-x86_64-unknown-linux-gnu\.tar\.gz$`,
		},
		{
			Path:   "junegunn/fzf",
			Desc:   "A command-line fuzzy finder",
			Regexp: `^fzf-.*-linux_amd64\.tar\.gz$`,
		},
		{
			Path:   "dandavison/delta",
			Desc:   "A syntax-highlighting pager for git, diff, and grep output",
			Regexp: `^delta-.*-x86_64-unknown-linux-gnu\.tar\.gz$`,
		},
		{
			Path:   "theryangeary/choose",
			Desc:   "A human-friendly and fast alternative to cut and (sometimes) awk",
			Regexp: `^choose-x86_64-unknown-linux-gnu$`,
		},
		{
			Path:   "direnv/direnv",
			Desc:   "Unclutter your .profile with an extensible shell environment manager",
			Regexp: `^direnv\.linux-amd64$`,
		},
		{
			Path:   "lsd-rs/lsd",
			Desc:   "The next gen ls command",
			Regexp: `^lsd-v.*-x86_64-unknown-linux-gnu\.tar\.gz$`,
		},
		{
			Path:   "zellij-org/zellij",
			Desc:   "A terminal workspace with batteries included",
			Regexp: `^zellij.*x86_64.*linux.*\.tar\.gz$`,
		},
		{
			Path:   "aquasecurity/trivy",
			Desc:   "Scanner for vulnerabilities in container images, file systems, and Git repositories",
			Regexp: `^trivy_.*_Linux-64bit\.tar\.gz$`,
		},
		{
			Path:   "FiloSottile/age",
			Desc:   "A simple, modern and secure file encryption tool",
			Regexp: `linux-amd64\.tar\.gz$`,
		},
		{
			Path:   "kubernetes/kompose",
			Desc:   "A tool to convert Docker Compose files to Kubernetes manifests",
			Regexp: `^kompose-linux-amd64$`,
		},
		{
			Path:   "wagoodman/dive",
			Desc:   "A tool for exploring each layer in a docker image",
			Regexp: `dive_.*_linux_amd64\.tar\.gz`,
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
