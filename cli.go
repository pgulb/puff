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

func Run() {
	cfgDir := MustCreateCfgDir()

	// handle writing github PAT into file if it doesn't exist
	// else read it
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

	// add puff bin directory to PATH if user wants
	// for ~/.bashrc and ~/.zshrc
	err = MustCreateBinDir(cfgDir)
	if err != nil {
		log.Fatal(err.Error())
	}
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

	// write metadata.json skeleton file if not exists
	err = MaybeCreateMetadata(cfgDir)
	if err != nil {
		log.Fatal(err.Error())
	}

	// commands
	if len(os.Args) < 2 {
		printHelp()
	}
	switch os.Args[1] {
	case "list":
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
	case "search":
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
	case "add":
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
	case "upd":
		fmt.Println("Updating all installed binaries")
		metadata, err := GetMetadata(cfgDir)
		if err != nil {
			log.Fatal(err.Error())
		}
		err = Update(cfgDir, ghPat, metadata)
		if err != nil {
			log.Fatal(err.Error())
		}
	case "rm":
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
	case "version", "--version", "-v":
		fmt.Println(Version)
	default:
		printHelp()
	}
}
