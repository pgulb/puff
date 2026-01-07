package main

import (
	"fmt"
	"os"
	"strings"

	puff "github.com/pgulb/puff"
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

func main() {
	cfgDir := puff.MustCreateCfgDir()

	// handle writing github PAT into file if it doesn't exist
	// else read it
	ghPat, err := puff.GetGhPat(cfgDir)
	if err != nil {
		fmt.Printf("error getting gh_pat: %s", err.Error())
	}
	if ghPat == "" {
		err := puff.PromptForGhPat(cfgDir)
		if err != nil {
			fmt.Printf("error writing gh_pat to file: %s", err.Error())
		}
	}

	// add puff bin directory to PATH if user wants
	// for ~/.bashrc and ~/.zshrc
	err = puff.MustCreateBinDir(cfgDir)
	if err != nil {
		fmt.Println(err.Error())
	}
	prompted, err := puff.WasPromptedForPath(cfgDir)
	if err != nil {
		fmt.Println(err.Error())
	}
	if !prompted {
		err = puff.PromptForAddToPath(cfgDir)
		if err != nil {
			fmt.Println(err.Error())
		}
	}

	// write metadata.json skeleton file if not exists
	err = puff.MaybeCreateMetadata(cfgDir)
	if err != nil {
		fmt.Println(err.Error())
	}

	// commands
	if len(os.Args) < 2 {
		printHelp()
	}
	switch os.Args[1] {
	case "list":
		metadata, err := puff.GetMetadata(cfgDir)
		if err != nil {
			fmt.Println(err.Error())
		}
		if len(metadata.Metadata) == 0 {
			fmt.Println("No installed binaries found.")
			return
		} else {
			for _, v := range metadata.Metadata {
				fmt.Printf("- %s (version: %s)\n", v.Path, v.Version)
			}
		}
	case "search":
		if len(os.Args) < 3 {
			for _, repo := range *puff.AvailableRepos() {
				fmt.Printf("- %s - %s\n", repo.Path, repo.Desc)
			}
		} else {
			searchTerm := os.Args[2]
			for _, repo := range *puff.AvailableRepos() {
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
			err := puff.Add(cfgDir, &installRepo, ghPat)
			if err != nil {
				fmt.Println(err.Error())
			}
		}
	case "upd":
		fmt.Println("Updating all installed binaries")
		metadata, err := puff.GetMetadata(cfgDir)
		if err != nil {
			fmt.Println(err.Error())
		}
		err = puff.Update(cfgDir, ghPat, metadata)
		if err != nil {
			fmt.Println(err.Error())
		}
	case "rm":
		if len(os.Args) < 3 {
			printHelp()
		}
		reposToRemove := os.Args[2:]
		for _, removeRepo := range reposToRemove {
			err := puff.Remove(cfgDir, &removeRepo)
			if err != nil {
				fmt.Println(err.Error())
			}
		}
	case "version", "--version", "-v":
		fmt.Println(puff.Version)
	default:
		printHelp()
	}
}
