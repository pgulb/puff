package puff

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"time"
)

// Release holds latest version and download link for a Repo
type Release struct {
	Version string
	Link    string
}

// GithubResponse holds response from Github API for a release
type GithubResponse struct {
	Version string `json:"tag_name"`
	Assets  []struct {
		Name string `json:"name"`
		URL  string `json:"browser_download_url"`
	}
}

// returns authenticated *http.Client and *http.Request
func AuthedClient(url string, ghPat string) (*http.Client, *http.Request, error) {
	c := http.DefaultClient
	c.Timeout = 15 * time.Second
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, nil, err
	}
	req.Header.Set("Authorization", "Bearer "+ghPat)
	return c, req, nil
}

// finds latest version and download link for a Repo
func GetLatestRelease(repo *Repo, ghPat string) (*Release, error) {
	RepoUrl, err := url.JoinPath("https://api.github.com", "repos", repo.Path, "releases/latest")
	c, req, err := AuthedClient(RepoUrl, ghPat)
	if err != nil {
		return nil, err
	}

	resp, err := c.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}
		var releaseJson GithubResponse
		err = json.Unmarshal(bodyBytes, &releaseJson)
		if err != nil {
			return nil, err
		}
		release := &Release{
			Version: releaseJson.Version,
		}

		validName := regexp.MustCompile(repo.Regexp)
		for _, asset := range releaseJson.Assets {
			if validName.MatchString(asset.Name) {
				release.Link = asset.URL
				return release, nil
			}
		}
		return nil, errors.New("No regexp matching name found in release assets")
	} else {
		return nil, err
	}
}

// downloads a binary and puts into bin directory
func DownloadBinary(cfgDir string, repo *Repo, release *Release, ghPat string) error {
	c, req, err := AuthedClient(release.Link, ghPat)
	if err != nil {
		return err
	}

	resp, err := c.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		binName, err := BinNameFromPath(repo)
		if err != nil {
			return err
		}
		savePath := filepath.Join(cfgDir, "bin", binName)
		log.Printf("writing %s to %s\n", binName, savePath)
		err = os.WriteFile(savePath, bodyBytes, 0750)
		if err != nil {
			return err
		}
	} else {
		return fmt.Errorf("API returned status %v", resp.StatusCode)
	}

	return nil
}
