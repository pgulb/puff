package puff

import (
	"fmt"
	"log"
	"os"
	"strings"
)

func printHelp() {
	fmt.Fprintln(os.Stderr, Bold(os.Stderr, "puff"), Dim(os.Stderr, "- simple binary package manager for GitHub releases"))
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Usage:")
	fmt.Fprintln(os.Stderr, "  "+Cyan(os.Stderr, "list")+"     -> list installed binaries")
	fmt.Fprintln(os.Stderr, "  "+Cyan(os.Stderr, "search")+"  <name (opt.)> -> search pre-added repositories")
	fmt.Fprintln(os.Stderr, "  "+Cyan(os.Stderr, "add")+"     <repo> <repo>... -> install binary from repo(s)")
	fmt.Fprintln(os.Stderr, "  "+Cyan(os.Stderr, "upd")+"     -> update all installed binaries")
	fmt.Fprintln(os.Stderr, "  "+Cyan(os.Stderr, "rm")+"      <repo> <repo>... -> remove installed binary/ies")
	fmt.Fprintln(os.Stderr, "  "+Cyan(os.Stderr, "version")+"|--version|-v -> print puff version")
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

// setup directory structure, gh token etc
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
		fmt.Println("No installed binaries found.")
		return
	}
	rows := make([][]string, 0, len(metadata.Metadata))
	for _, v := range metadata.Metadata {
		name := v.Path
		if parts := strings.Split(v.Path, "/"); len(parts) > 0 {
			name = parts[len(parts)-1]
		}
		rows = append(rows, []string{name, v.Version})
	}
	fmt.Println()
	renderTable(os.Stdout, []string{"Binary", "Version"}, []int{24, 16}, rows)
	fmt.Println()
}

// search featured repositories
func searchCmd() {
	repos := *AvailableRepos()
	if len(os.Args) >= 3 {
		searchTerm := strings.ToLower(os.Args[2])
		filtered := make([]Repo, 0, len(repos))
		for _, repo := range repos {
			if strings.Contains(strings.ToLower(repo.Path), searchTerm) ||
				strings.Contains(strings.ToLower(repo.Desc), searchTerm) {
				filtered = append(filtered, repo)
			}
		}
		repos = filtered
	}
	if len(repos) == 0 {
		fmt.Println("No matching repositories found.")
		return
	}
	rows := make([][]string, 0, len(repos))
	for _, repo := range repos {
		rows = append(rows, []string{repo.Path, truncate(repo.Desc, 80)})
	}
	fmt.Println()
	renderDynamicTable(os.Stdout, []string{"Repository", "Description"}, rows, 12, 8)
	fmt.Println()
}

// install binary/binaries
func addCmd(cfgDir string, ghPat string) {
	if len(os.Args) < 3 {
		printHelp()
	}
	reposToAdd := os.Args[2:]
	for _, installRepo := range reposToAdd {
		fmt.Fprintln(os.Stdout, Cyan(os.Stdout, "installing ")+installRepo)
		err := Add(cfgDir, &installRepo, ghPat)
		if err != nil {
			log.Fatal(err.Error())
		}
	}
}

// update all installed binaries and puff itself
func updCmd(cfgDir string, ghPat string) {
	fmt.Fprintln(os.Stdout, Bold(os.Stdout, "Updating all installed binaries"))
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
		fmt.Println(Bold(os.Stdout, "puff") + " " + Cyan(os.Stdout, Version))
	default:
		printHelp()
	}
}
