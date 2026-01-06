package main

import (
	"fmt"
	"log"
	"os"

	puff "github.com/pgulb/puff"
)

func printHelp() {
	fmt.Println("puff - a tool for managing binary installations")
	fmt.Println("Usage:")
	fmt.Println("  puff list -> list available repositories")
	fmt.Println("  puff add <repo> -> install binary from non-listed repo")
	fmt.Println("  puff upd -> update all installed binaries")
	fmt.Println("  puff rm <repo> -> remove installed binary")
	os.Exit(1)
}

func main() {
	// logging init
	cfgDir := puff.MustCreateCfgDir()
	f := puff.MustSetupLog(cfgDir)
	defer f.Close()
	log.Println(os.Args)

	// handle writing github PAT into file if it doesn't exist
	// else read it
	ghPat, err := puff.GetGhPat(cfgDir)
	if err != nil {
		fmt.Printf("error getting gh_pat: %s", err.Error())
		log.Fatalf("error getting gh_pat: %s", err.Error())
	}
	if ghPat == "" {
		err := puff.PromptForGhPat(cfgDir)
		if err != nil {
			fmt.Printf("error writing gh_pat to file: %s", err.Error())
			log.Fatalf("error writing gh_pat to file: %s", err.Error())
		}
	}

	// add puff bin directory to PATH if user wants
	// for ~/.bashrc and ~/.zshrc
	err = puff.MustCreateBinDir(cfgDir)
	if err != nil {
		fmt.Println(err.Error())
		log.Fatal(err.Error())
	}
	prompted, err := puff.WasPromptedForPath(cfgDir)
	if err != nil {
		fmt.Println(err.Error())
		log.Fatal(err.Error())
	}
	if !prompted {
		err = puff.PromptForAddToPath(cfgDir)
		if err != nil {
			fmt.Println(err.Error())
			log.Fatal(err.Error())
		}
	}

	// write metadata.json skeleton file if not exists
	err = puff.MaybeCreateMetadata(cfgDir)
	if err != nil {
		fmt.Println(err.Error())
		log.Fatal(err.Error())
	}

	// commands
	if len(os.Args) < 2 {
		printHelp()
	}
	switch os.Args[1] {
	case "list":
		fmt.Println("Available repositories: ")
		for _, repo := range *puff.AvailableRepos() {
			fmt.Printf("- %s - %s\n", repo.Path, repo.Desc)
		}
	case "add":
		if len(os.Args) < 3 {
			printHelp()
		}
		installRepo := &os.Args[2]
		err := puff.Add(cfgDir, installRepo, ghPat)
		if err != nil {
			fmt.Println(err.Error())
			log.Fatal(err.Error())
		}
	case "upd":
		fmt.Println("Updating all installed binaries")
		metadata, err := puff.GetMetadata(cfgDir)
		if err != nil {
			fmt.Println(err.Error())
			log.Fatal(err.Error())
		}
		err = puff.Update(cfgDir, ghPat, metadata)
		if err != nil {
			fmt.Println(err.Error())
			log.Fatal(err.Error())
		}
	case "rm":
		if len(os.Args) < 3 {
			printHelp()
		}
		removeRepo := &os.Args[2]
		err := puff.Remove(cfgDir, removeRepo)
		if err != nil {
			fmt.Println(err.Error())
			log.Fatal(err.Error())
		}
	default:
		printHelp()
	}
}
