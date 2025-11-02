package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

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
		fmt.Printf("installing %s\n", *installRepo)
		found := false
		for _, repo := range *puff.AvailableRepos() {
			if repo.Path == *installRepo {
				release, err := puff.GetLatestRelease(&repo, ghPat)
				if err != nil {
					fmt.Println(err.Error())
					log.Fatal(err.Error())
				}
				fmt.Printf("version found: %s\n", release.Version)
				metadata, err := puff.GetMetadata(cfgDir)
				if err != nil {
					fmt.Println(err.Error())
					log.Fatal(err.Error())
				}
				added, err := puff.AddMetaIfNotExists(metadata, &repo, release, nil)
				if err != nil {
					fmt.Println(err.Error())
					log.Fatal(err.Error())
				}
				if added {
					err = puff.SaveMetadata(metadata, cfgDir)
					if err != nil {
						fmt.Println(err.Error())
						log.Fatal(err.Error())
					}
					err := puff.DownloadBinary(cfgDir, &repo, release, ghPat)
					if err != nil {
						fmt.Println(err.Error())
						log.Fatal(err.Error())
					}
					fmt.Printf(
						"%s at version %s successfully installed!\n",
						repo.Path,
						release.Version,
					)
				} else {
					log.Printf("%s at version %s already installed\n", repo.Path, release.Version)
					fmt.Printf("%s at version %s already installed\n", repo.Path, release.Version)
				}
				found = true
				break
			}
		}
		if !found {
			fmt.Println("binary not found in featured repos")
			log.Println("binary not in featured repos")
			ghResp, err := puff.GetLatestReleaseAssets(*installRepo, ghPat)
			if err != nil {
				fmt.Println(err.Error())
				log.Fatal(err.Error())
			}
			if ghResp != nil {
				if ghResp.Assets != nil {
					if len(ghResp.Assets) == 0 {
						log.Println("no assets found in release")
						fmt.Println("no assets found in release")
						return
					}
					fmt.Println("\nAvailable binaries:")
					for _, v := range ghResp.Assets {
						fmt.Println(v.Name)
					}
					nameParts := puff.PromptForNameParts()
					for _, v := range ghResp.Assets {
						containsAll := false
						for _, part := range nameParts {
							if strings.Contains(v.Name, part) {
								containsAll = true
							} else {
								containsAll = false
								break
							}
						}
						if containsAll {
							metadata, err := puff.GetMetadata(cfgDir)
							if err != nil {
								fmt.Println(err.Error())
								log.Fatal(err.Error())
							}
							repo := &puff.Repo{Path: *installRepo}
							release := &puff.Release{
								Version: ghResp.Version,
								Link:    v.URL,
							}
							added, err := puff.AddMetaIfNotExists(metadata,
								repo,
								release,
								nameParts,
							)
							if added {
								err = puff.SaveMetadata(metadata, cfgDir)
								if err != nil {
									fmt.Println(err.Error())
									log.Fatal(err.Error())
								}
								err := puff.DownloadBinary(cfgDir, repo, release, ghPat)
								if err != nil {
									fmt.Println(err.Error())
									log.Fatal(err.Error())
								}
								fmt.Printf(
									"%s at version %s successfully installed!\n",
									repo.Path,
									release.Version,
								)
							} else {
								log.Printf("%s at version %s already installed\n", repo.Path, release.Version)
								fmt.Printf("%s at version %s already installed\n", repo.Path, release.Version)
							}
							break
						}
					}
				} else {
					log.Println("no assets found in release")
					fmt.Println("no assets found in release")
				}
			} else {
				log.Println("no release found")
				fmt.Println("no release found")
			}
		}
		return
	}
}
