package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	puff "github.com/pgulb/puff"
)

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

	// flag commands

	// puff --list
	listAvailableRepos := flag.Bool("list", false, "list available repositories to download from")

	// puff --add
	installRepo := flag.String("add", "", "install binary from repository")

	// end flags
	flag.Parse()

	// puff --list
	if *listAvailableRepos {
		fmt.Println("Available repositories: ")
		for _, repo := range *puff.AvailableRepos() {
			fmt.Printf("- %s - %s\n", repo.Path, repo.Desc)
		}
		return
	}

	// puff --add
	if *installRepo != "" {
		puff.Add(cfgDir, installRepo, ghPat)
		return
	}
}
