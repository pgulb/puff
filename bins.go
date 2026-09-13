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
	"sync"
)

// installPlan is the result of checking one repo for an update.
type installPlan struct {
	repo    Repo
	release *Release
	err     error
}

// installResult is the outcome of installing one repo.
type installResult struct {
	path string
	err  error
}

// checkLatestRelease fetches the latest release for a single repo.
func checkLatestRelease(repo Repo, ghPat string) installPlan {
	release, err := GetLatestRelease(&repo, ghPat)
	return installPlan{repo: repo, release: release, err: err}
}

// checkRepos fetches the latest release for each repo concurrently.
func checkRepos(repos []Repo, ghPat string) []installPlan {
	results := make([]installPlan, len(repos))
	var next int
	var mu sync.Mutex
	var wg sync.WaitGroup
	for range repos {
		wg.Add(1)
		go func() {
			defer wg.Done()
			mu.Lock()
			idx := next
			next++
			mu.Unlock()
			results[idx] = checkLatestRelease(repos[idx], ghPat)
		}()
	}
	wg.Wait()
	return results
}

// saveRelease downloads and installs a single repo, mutating the shared
// metadata under lock only after the download succeeds, so a failed download
// never leaves metadata claiming a binary that is not on disk. It does NOT
// write metadata.json to disk; the caller is responsible for a single atomic
// write after all installs complete.
// Only valid when called from installFeatured (meta and metaMu are non-nil).
func saveRelease(
	cfgDir string,
	repo Repo,
	release *Release,
	ghPat string,
	meta *MetadataList,
	metaMu *sync.Mutex,
) error {
	if release == nil {
		return errors.New("no release to install")
	}

	// Read-only check under lock: is an update actually needed?
	metaMu.Lock()
	needs := needsUpdate(meta, &repo, release)
	metaMu.Unlock()
	if !needs {
		fmt.Printf("%s at %s already installed\n", repo.Path, Yellow(os.Stdout, release.Version))
		return nil
	}

	if err := DownloadBinary(cfgDir, &repo, release, ghPat); err != nil {
		return err
	}

	// Download succeeded; now record the new version under lock.
	metaMu.Lock()
	added, err := AddMetaIfNotExists(meta, &repo, release, nil)
	metaMu.Unlock()
	if err != nil {
		return err
	}
	if !added {
		fmt.Printf("%s at %s already installed\n", repo.Path, Yellow(os.Stdout, release.Version))
		return nil
	}
	fmt.Printf(
		"%s at %s successfully installed!\n", repo.Path, Green(os.Stdout, release.Version),
	)
	return nil
}

// installFeatured installs featured repos concurrently: check all releases in
// parallel, then download and install the ones that need updating, also in
// parallel. metadata.json is NOT written; the caller writes it once.
// Returns the list of errors for repos that failed (empty if all succeeded).
// If progress is non-nil, each completed step (check or install) advances it.
func installFeatured(
	cfgDir string,
	repos []Repo,
	ghPat string,
	meta *MetadataList,
	metaMu *sync.Mutex,
	progress *progressTracker,
) []error {
	plans := checkRepos(repos, ghPat)
	if progress != nil {
		for range plans {
			progress.tick()
		}
	}

	var wg sync.WaitGroup
	results := make([]installResult, len(plans))
	for i, p := range plans {
		if p.err != nil {
			results[i] = installResult{path: p.repo.Path, err: p.err}
			continue
		}
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			p := plans[idx]
			err := saveRelease(cfgDir, p.repo, p.release, ghPat, meta, metaMu)
			results[idx] = installResult{path: p.repo.Path, err: err}
			if progress != nil {
				progress.tick()
			}
		}(i)
	}
	wg.Wait()

	var errs []error
	for _, r := range results {
		if r.err != nil {
			fmt.Fprintf(os.Stderr, "%s\n", Red(os.Stderr, fmt.Sprintf("error installing %s: %v", r.path, r.err)))
			errs = append(errs, r.err)
		}
	}
	return errs
}

// check if custom repo api response is valid
func validateGhCustomResp(ghResp *GithubResponse) error {
	if ghResp == nil {
		return errors.New("no release found")
	}
	if ghResp.Assets == nil {
		return errors.New("no assets section in release response")
	}
	if len(ghResp.Assets) == 0 {
		return errors.New("no assets found in release")
	}
	return nil
}

// addCustom handles the add command for custom repos (sequential, interactive).
func addCustom(cfgDir string, installRepo *string, ghPat string) error {
	fmt.Fprintln(os.Stdout, Dim(os.Stdout, "binary not found in featured repos"))
	ghResp, err := GetLatestReleaseAssets(*installRepo, ghPat)
	if err != nil {
		return err
	}
	err = validateGhCustomResp(ghResp)
	if err != nil {
		return err
	}
	meta, err := GetMetadata(cfgDir)
	if err != nil {
		return err
	}
	isAdded := IsCustomRepoAdded(meta, *installRepo)
	repo := Repo{Path: *installRepo}
	var nameParts []string
	if isAdded.Version != "" {
		fmt.Printf("%s custom repo already added\n", *installRepo)
		nameParts = isAdded.NameParts
	} else {
		fmt.Fprintln(os.Stdout, "\n"+Bold(os.Stdout, "Available binaries:"))
		for _, v := range ghResp.Assets {
			fmt.Fprintln(os.Stdout, Dim(os.Stdout, v.Name))
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
			// Sequential, interactive install: mutate metadata only after a
			// successful download, then write atomically.
			if !needsUpdate(meta, &repo, release) {
				fmt.Printf("%s at %s already installed\n", repo.Path, Yellow(os.Stdout, release.Version))
				break
			}
			if err := DownloadBinary(cfgDir, &repo, release, ghPat); err != nil {
				return err
			}
			added, err := AddMetaIfNotExists(meta, &repo, release, nameParts)
			if err != nil {
				return err
			}
			if !added {
				fmt.Printf("%s at %s already installed\n", repo.Path, Yellow(os.Stdout, release.Version))
				break
			}
			fmt.Printf(
				"%s at %s successfully installed!\n", repo.Path, Green(os.Stdout, release.Version),
			)
			if err := SaveMetadata(meta, cfgDir); err != nil {
				return err
			}
			break
		}
	}
	return nil
}

// addFeatured installs a single featured repo (sequential, for single-repo add).
// Loads its own metadata copy and writes it atomically on success.
func addFeatured(cfgDir string, repo Repo, ghPat string) error {
	plan := checkLatestRelease(repo, ghPat)
	if plan.err != nil {
		return plan.err
	}
	fmt.Fprintf(os.Stdout, "latest version: %s\n", Cyan(os.Stdout, plan.release.Version))

	meta, err := GetMetadata(cfgDir)
	if err != nil {
		return err
	}
	if !needsUpdate(meta, &repo, plan.release) {
		fmt.Printf("%s at %s already installed\n", repo.Path, Yellow(os.Stdout, plan.release.Version))
		return nil
	}
	if err := DownloadBinary(cfgDir, &repo, plan.release, ghPat); err != nil {
		return err
	}
	added, err := AddMetaIfNotExists(meta, &repo, plan.release, nil)
	if err != nil {
		return err
	}
	if !added {
		fmt.Printf("%s at %s already installed\n", repo.Path, Yellow(os.Stdout, plan.release.Version))
		return nil
	}
	fmt.Printf(
		"%s at %s successfully installed!\n", repo.Path, Green(os.Stdout, plan.release.Version),
	)
	return SaveMetadata(meta, cfgDir)
}

// Add installs a single repo. Featured repos are installed concurrently when
// called from Update; for a single add, the featured path is sequential.
// metadata.json is written once, atomically, after the install completes.
func Add(cfgDir string, installRepo *string, ghPat string) error {
	for _, repo := range *AvailableRepos() {
		if repo.Path == *installRepo {
			return addFeatured(cfgDir, repo, ghPat)
		}
	}
	return addCustom(cfgDir, installRepo, ghPat)
}

// Update updates all installed binaries concurrently, then self-updates puff
// sequentially if every repo update succeeded. metadata.json is written once,
// atomically, at the end.
func Update(cfgDir string, ghPat string, metadata *MetadataList) error {
	fmt.Fprintln(os.Stdout, Bold(os.Stdout, "──")+" "+Bold(os.Stdout, "updating installed binaries")+" "+Dim(os.Stdout, "──"))

	// Convert installed metadata entries into Repo values for checking.
	repos := make([]Repo, len(metadata.Metadata))
	for i, m := range metadata.Metadata {
		repos[i] = Repo{Path: m.Path}
	}
	// Total steps = per-repo check + per-repo install, plus the self-update.
	total := len(repos)*2 + 1
	progress := newProgressTracker(os.Stdout, total, "updating binaries")
	metaMu := &sync.Mutex{}
	errs := installFeatured(cfgDir, repos, ghPat, metadata, metaMu, progress)

	if len(errs) == 0 {
		progress.tick() // self-update check
		fmt.Fprintln(os.Stdout, "\n"+Bold(os.Stdout, "updating puff"))
		puffRepo := Repo{Path: "pgulb/puff"}
		puffRelease, err := GetLatestRelease(&puffRepo, ghPat)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s\n", Red(os.Stderr, fmt.Sprintf("error checking for puff update: %v", err)))
			errs = append(errs, err)
		} else if puffRelease.Version != Version {
			fmt.Fprintf(os.Stdout, "new version available: %s\n", Cyan(os.Stdout, puffRelease.Version))
			err := DownloadBinary(cfgDir, &puffRepo, puffRelease, ghPat)
			if err != nil {
				fmt.Fprintf(os.Stderr, "%s\n", Red(os.Stderr, fmt.Sprintf("error updating puff: %v", err)))
				errs = append(errs, err)
			} else {
				progress.tick() // self-update install
				fmt.Fprintf(os.Stdout, "puff updated to %s\n", Green(os.Stdout, puffRelease.Version))
			}
		} else {
			progress.tick() // self-update install (no-op)
			fmt.Fprintf(os.Stdout, "puff is up to date at %s\n", Green(os.Stdout, Version))
		}
	} else {
		progress.finish()
		fmt.Fprintf(os.Stderr, "%s\n", Yellow(os.Stderr, fmt.Sprintf("skipping puff self-update due to %d repo error(s)", len(errs))))
		return SaveMetadata(metadata, cfgDir)
	}
	progress.finish()
	return SaveMetadata(metadata, cfgDir)
}

// Remove removes installed binaries and their metadata entries.
func Remove(cfgDir string, removeRepo *string) error {
	meta, err := GetMetadata(cfgDir)
	if err != nil {
		return err
	}
	for _, v := range meta.Metadata {
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
					fmt.Fprintf(os.Stdout, "removing %s binary\n", Dim(os.Stdout, v.Name()))
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
			fmt.Fprintf(os.Stdout, "removing %s from metadata\n", Dim(os.Stdout, *removeRepo))
			var newMeta []Metadata
			for _, metaEntry := range meta.Metadata {
				if metaEntry.Path != *removeRepo {
					newMeta = append(newMeta, metaEntry)
				}
			}
			meta.Metadata = newMeta
			return SaveMetadata(meta, cfgDir)
		}
	}
	return nil
}

// saveBin writes a binary atomically: always write to a temp file then rename
// over the final path, so a failed or interrupted download never leaves a
// half-written executable in place.
func saveBin(savePath string, tempPath string, binName string, fileBytes []byte) error {
	fmt.Fprintf(os.Stdout, "writing %s to %s\n", Dim(os.Stdout, binName), Dim(os.Stdout, tempPath))
	err := os.WriteFile(tempPath, fileBytes, 0750)
	if err != nil {
		return err
	}
	err = os.Rename(tempPath, savePath)
	if err != nil {
		return err
	}
	fmt.Fprintf(os.Stdout, "installed %s\n", Green(os.Stdout, binName))
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

// saveOrUnpack saves binary directly to bin or unpacks it if it's .tar.gz
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