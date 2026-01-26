package puff

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// download binary and save metadata entry
func saveRelease(cfgDir string, repo Repo, release *Release, ghPat string) error {
	metadata, err := GetMetadata(cfgDir)
	if err != nil {
		return err
	}
	added, err := AddMetaIfNotExists(metadata, &repo, release, nil)
	if err != nil {
		return err
	}
	if added {
		err := DownloadBinary(cfgDir, &repo, release, ghPat)
		if err != nil {
			return err
		}
		err = SaveMetadata(metadata, cfgDir)
		if err != nil {
			return err
		}
		fmt.Printf(
			"%s at version %s successfully installed!\n", repo.Path, release.Version,
		)
	} else {
		fmt.Printf("%s at version %s already installed\n", repo.Path, release.Version)
	}
	return nil
}

// handling add command for featured repos
func addFeatured(cfgDir string, repo Repo, ghPat string) error {
	release, err := GetLatestRelease(&repo, ghPat)
	if err != nil {
		return err
	}
	fmt.Printf("latest version: %s\n", release.Version)
	err = saveRelease(cfgDir, repo, release, ghPat)
	if err != nil {
		return err
	}
	return nil
}

// check if custom repo api response is valid
func validateGhCustomResp(ghResp *GithubResponse) error {
	if ghResp == nil {
		return errors.New("no release found")
	}
	if ghResp.Assets == nil {
		return errors.New(("no assets section in release response"))
	}
	if len(ghResp.Assets) == 0 {
		return errors.New(("no assets found in release"))
	}
	return nil
}

// handling add command for custom repos
func addCustom(cfgDir string, installRepo *string, ghPat string) error {
	fmt.Println("binary not found in featured repos")
	ghResp, err := GetLatestReleaseAssets(*installRepo, ghPat)
	if err != nil {
		return err
	}
	err = validateGhCustomResp(ghResp)
	if err != nil {
		return err
	}
	metadata, err := GetMetadata(cfgDir)
	if err != nil {
		return err
	}
	isAdded := IsCustomRepoAdded(metadata, *installRepo)
	repo := Repo{Path: *installRepo}
	var nameParts []string
	if isAdded.Version != "" {
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
			release := &Release{Version: ghResp.Version, Link: asset.URL}
			err = saveRelease(cfgDir, repo, release, ghPat)
			if err != nil {
				return err
			}
			break
		}
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

	fmt.Println("updating puff")
	puffRepo := Repo{Path: "pgulb/puff"}
	puffRelease, err := GetLatestRelease(&puffRepo, ghPat)
	if err != nil {
		return err
	}
	if puffRelease.Version != Version {
		fmt.Printf("new version available: %s\n", puffRelease.Version)
		err := DownloadBinary(cfgDir, &puffRepo, puffRelease, ghPat)
		if err != nil {
			return err
		}
		fmt.Printf("puff updated to version %s\n", puffRelease.Version)
	} else {
		fmt.Printf("puff is up to date at version %s\n", Version)
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
			var removed bool
			for _, v := range binaries {
				if v.Name() == expectedBinary {
					fmt.Printf("removing %s binary\n", v.Name())
					removed = true
					err := os.Remove(filepath.Join(binDir, v.Name()))
					if err != nil {
						return err
					}
					break
				}
			}
			if !removed {
				return fmt.Errorf("binary for %s not found to remove", expectedBinary)
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

// saves binary to bin
func saveBin(savePath string, tempPath string, binName string, fileBytes []byte) error {
	writePath := savePath
	if binName == "puff" {
		writePath = tempPath
	}
	fmt.Printf("writing %s to %s\n", binName, writePath)
	err := os.WriteFile(writePath, fileBytes, 0750)
	if err != nil {
		return err
	}
	if binName == "puff" {
		err = os.Rename(tempPath, savePath)
		if err != nil {
			return err
		}
		fmt.Printf("replaced %s with new version\n", savePath)
	}
	return nil
}

// decompresses .tar.gz file and returns tar bytes
func degzip(assetName string, fileBytes []byte) ([]byte, error) {
	fmt.Printf("unpacking %s\n", assetName)
	bodyReader := bytes.NewReader(fileBytes)
	zr, err := gzip.NewReader(bodyReader)
	if err != nil {
		return nil, err
	}
	defer zr.Close()
	degzippedBytes, err := io.ReadAll(zr)
	if err != nil {
		return nil, err
	}
	return degzippedBytes, nil
}

// handle extracting binary from .tar.gz file and saving it to bin
func saveFromTgz(
	savePath string,
	tempPath string,
	binName string,
	assetName string,
	fileBytes []byte,
) error {
	degzippedBytes, err := degzip(assetName, fileBytes)
	if err != nil {
		return err
	}
	degzippedBody := bytes.NewReader(degzippedBytes)
	tr := tar.NewReader(degzippedBody)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break // End of archive
		}
		if err != nil {
			return err
		}
		var nameToCompare string
		// handle possible paths in tarball
		if strings.Contains(hdr.Name, "/") {
			nameToCompare = strings.Split(hdr.Name, "/")[len(strings.Split(hdr.Name, "/"))-1]
		} else {
			nameToCompare = hdr.Name
		}
		// save binary
		if nameToCompare == binName {
			binBytes, err := io.ReadAll(tr)
			if err != nil {
				return err
			}
			if string(binBytes[:3]) == "#!/" {
				// binary is a script/autocomplete definition
				continue
			}
			err = saveBin(savePath, tempPath, binName, binBytes)
			if err != nil {
				return err
			}
			return nil
		}
	}
	return errors.New("binary not found in tar.gz archive")
}

// saves binary directly to bin or unpacks it if it's .tar.gz
func saveOrUnpack(cfgDir string, bodyBytes []byte, binName string, assetName string) error {
	savePath := filepath.Join(cfgDir, "bin", binName)
	tempPath := savePath + ".tmp"
	matched, err := regexp.MatchString(`\.tar\.gz$`, assetName)
	if err != nil {
		return err
	}
	matchedTgz, err := regexp.MatchString(`\.tgz$`, assetName)
	if err != nil {
		return err
	}
	if matched || matchedTgz {
		err = saveFromTgz(savePath, tempPath, binName, assetName, bodyBytes)
		if err != nil {
			return err
		}
	} else {
		// save directly
		err = saveBin(savePath, tempPath, binName, bodyBytes)
		if err != nil {
			return err
		}
	}
	return nil
}
