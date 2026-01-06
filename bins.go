package puff

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
)

// handling add command for featured repos
func addFeatured(cfgDir string, repo Repo, ghPat string) error {
	release, err := GetLatestRelease(&repo, ghPat)
	if err != nil {
		return err
	}
	fmt.Printf("latest version: %s\n", release.Version)
	metadata, err := GetMetadata(cfgDir)
	if err != nil {
		return err
	}
	added, err := AddMetaIfNotExists(metadata, &repo, release, nil)
	if err != nil {
		return err
	}
	if added {
		err = SaveMetadata(metadata, cfgDir)
		if err != nil {
			return err
		}
		err := DownloadBinary(cfgDir, &repo, release, ghPat)
		if err != nil {
			return err
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
	return nil
}

// handling add command for custom repos
func addCustom(cfgDir string, installRepo *string, ghPat string) error {
	fmt.Println("binary not found in featured repos")
	log.Println("binary not in featured repos")
	ghResp, err := GetLatestReleaseAssets(*installRepo, ghPat)
	if err != nil {
		return err
	}
	if ghResp != nil {
		if ghResp.Assets != nil {
			if len(ghResp.Assets) == 0 {
				log.Println("no assets found in release")
				fmt.Println("no assets found in release")
				return nil
			}
			metadata, err := GetMetadata(cfgDir)
			if err != nil {
				return err
			}
			isAdded := IsCustomRepoAdded(metadata, *installRepo)
			repo := Repo{Path: *installRepo}
			var nameParts []string
			if isAdded.Version != "" {
				log.Printf("%s custom repo already added\n", *installRepo)
				fmt.Printf("%s custom repo already added\n", *installRepo)
				nameParts = isAdded.NameParts
			} else {
				fmt.Println("\nAvailable binaries:")
				for _, v := range ghResp.Assets {
					fmt.Println(v.Name)
				}
				nameParts = PromptForNameParts()
			}
			for _, asset := range ghResp.Assets {
				containsAll := false
				for _, part := range nameParts {
					if strings.Contains(asset.Name, part) {
						containsAll = true
					} else {
						containsAll = false
						break
					}
				}
				if containsAll {
					metadata, err := GetMetadata(cfgDir)
					if err != nil {
						return err
					}
					release := &Release{
						Version: ghResp.Version,
						Link:    asset.URL,
					}
					added, err := AddMetaIfNotExists(metadata,
						&repo,
						release,
						nameParts,
					)
					if added {
						err = SaveMetadata(metadata, cfgDir)
						if err != nil {
							return err
						}
						err := DownloadBinary(cfgDir, &repo, release, ghPat)
						if err != nil {
							return err
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
	return nil
}

// handling add command
func Add(cfgDir string, installRepo *string, ghPat string) error {
	fmt.Printf("installing %s\n", *installRepo)
	found := false
	for _, repo := range *AvailableRepos() {
		if repo.Path == *installRepo {
			if err := addFeatured(cfgDir, repo, ghPat); err != nil {
				return err
			}
			found = true
			break
		}
	}
	if !found {
		if err := addCustom(cfgDir, installRepo, ghPat); err != nil {
			return err
		}
	}
	return nil
}

// handling update command
func Update(cfgDir string, ghPat string, metadata *MetadataList) error {
	fmt.Print("---\n\n")
	for _, m := range metadata.Metadata {
		err := Add(cfgDir, &m.Path, ghPat)
		if err != nil {
			return err
		}
		fmt.Print("---\n\n")
	}
	return nil
}

func Remove(cfgDir string, removeRepo *string) error {
	metadata, err := GetMetadata(cfgDir)
	if err != nil {
		return err
	}
	for _, v := range metadata.Metadata {
		if v.Path == *removeRepo {
			expectedBinary := strings.Split(v.Path, "/")[1]
			binDir := filepath.Join(cfgDir, "bin")
			binaries, err := os.ReadDir(binDir)
			if err != nil {
				return err
			}
			for _, v := range binaries {
				if strings.Contains(v.Name(), expectedBinary) {
					var input string
					fmt.Printf("Do you want to remove %s? (y/n): ", v.Name())
					fmt.Scanln(&input)
					if strings.ToLower(input) == "y" {
						err := os.Remove(filepath.Join(binDir, v.Name()))
						if err != nil {
							return err
						}
					} else {
						fmt.Println("skipping removal")
						return nil
					}
				}
			}
			fmt.Printf("removing %s from metadata\n", *removeRepo)
			var newMeta []Metadata
			for _, metaEntry := range metadata.Metadata {
				if metaEntry.Path != *removeRepo {
					newMeta = append(newMeta, metaEntry)
				}
			}
			metadata.Metadata = newMeta
			err = SaveMetadata(metadata, cfgDir)
			if err != nil {
				return err
			}
		}
	}
	return nil
}
