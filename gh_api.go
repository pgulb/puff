package puff

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
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

// finds latest version and download link for a Repo
func GetLatestRelease(repo *Repo, ghPat string) (*Release, error) {
	RepoUrl, err := url.JoinPath("https://api.github.com", "repos", repo.Path, "releases/latest")
	c := http.DefaultClient
	c.Timeout = 15 * time.Second

	req, err := http.NewRequest("GET", RepoUrl, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+ghPat)

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
