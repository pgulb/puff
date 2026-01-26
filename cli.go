package puff

import (
	"fmt"
	"log"
	"os"
	"strings"
)

func printHelp() {
	fmt.Println("puff - simple binary package manager for GitHub releases")
	fmt.Println("Usage:")
	fmt.Println("  puff list -> list installed binaries")
	fmt.Println("  puff search <name (opt.)> -> search pre-added repositories")
	fmt.Println("  puff add <repo> <repo>... -> install binary from repo(s)")
	fmt.Println("  puff upd -> update all installed binaries")
	fmt.Println("  puff rm <repo> <repo>... -> remove installed binary/ies")
	fmt.Println("  puff version|--version|-v -> print puff version")
	os.Exit(1)
}

// handle writing github PAT into file if it doesn't exist
// else read it
func setupGhPat(cfgDir string) string {
	ghPat, err := GetGhPat(cfgDir)
	if err != nil {
		log.Fatalf("error getting gh_pat: %s", err.Error())
	}
	if ghPat == "" {
		err := PromptForGhPat(cfgDir)
		if err != nil {
			log.Fatalf("error writing gh_pat to file: %s", err.Error())
		}
	}
	return ghPat
}

// add puff bin directory to PATH if user wants
// for ~/.bashrc and ~/.zshrc
func addToPath(cfgDir string) {
	prompted, err := WasPromptedForPath(cfgDir)
	if err != nil {
		log.Fatal(err.Error())
	}
	if !prompted {
		err = PromptForAddToPath(cfgDir)
		if err != nil {
			log.Fatal(err.Error())
		}
	}
}

// setup directiry structure, gh token etc
func setup() (string, string) {
	cfgDir := MustCreateCfgDir()
	ghPat := setupGhPat(cfgDir)

	err := MustCreateBinDir(cfgDir)
	if err != nil {
		log.Fatal(err.Error())
	}
	addToPath(cfgDir)

	err = MaybeCreateMetadata(cfgDir)
	if err != nil {
		log.Fatal(err.Error())
	}
	return cfgDir, ghPat
}

// list installed binaries
func listCmd(cfgDir string) {
	metadata, err := GetMetadata(cfgDir)
	if err != nil {
		log.Fatal(err.Error())
	}
	if len(metadata.Metadata) == 0 {
		log.Fatal("No installed binaries found.")
	} else {
		for _, v := range metadata.Metadata {
			fmt.Printf("- %s (version: %s)\n", v.Path, v.Version)
		}
	}
}

// search featured repositories
func searchCmd() {
	if len(os.Args) < 3 {
		for _, repo := range *AvailableRepos() {
			fmt.Printf("- %s - %s\n", repo.Path, repo.Desc)
		}
	} else {
		searchTerm := os.Args[2]
		for _, repo := range *AvailableRepos() {
			if strings.Contains(
				strings.ToLower(repo.Path),
				strings.ToLower(searchTerm)) || strings.Contains(
				strings.ToLower(repo.Desc), strings.ToLower(searchTerm)) {
				fmt.Printf("- %s - %s\n", repo.Path, repo.Desc)
			}
		}
	}
}

// install binary/binaries
func addCmd(cfgDir string, ghPat string) {
	if len(os.Args) < 3 {
		printHelp()
	}
	reposToAdd := os.Args[2:]
	for _, installRepo := range reposToAdd {
		err := Add(cfgDir, &installRepo, ghPat)
		if err != nil {
			log.Fatal(err.Error())
		}
	}
}

// update all installed binaries and puff itself
func updCmd(cfgDir string, ghPat string) {
	fmt.Println("Updating all installed binaries")
	metadata, err := GetMetadata(cfgDir)
	if err != nil {
		log.Fatal(err.Error())
	}
	err = Update(cfgDir, ghPat, metadata)
	if err != nil {
		log.Fatal(err.Error())
	}
}

// remove installed binary/binaries
func rmCmd(cfgDir string) {
	if len(os.Args) < 3 {
		printHelp()
	}
	reposToRemove := os.Args[2:]
	for _, removeRepo := range reposToRemove {
		err := Remove(cfgDir, &removeRepo)
		if err != nil {
			log.Fatal(err.Error())
		}
	}
}

func Run() {
	cfgDir, ghPat := setup()

	if len(os.Args) < 2 {
		printHelp()
	}
	switch os.Args[1] {
	case "list":
		listCmd(cfgDir)
	case "search":
		searchCmd()
	case "add":
		addCmd(cfgDir, ghPat)
	case "upd":
		updCmd(cfgDir, ghPat)
	case "rm":
		rmCmd(cfgDir)
	case "version", "--version", "-v":
		fmt.Println(Version)
	default:
		printHelp()
	}
}
