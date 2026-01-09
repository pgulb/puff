package puff

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
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
	c.Timeout = 600 * time.Second
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
		// unpack .tar.gz
		fmt.Printf("unpacking %s\n", assetName)
		bodyReader := bytes.NewReader(bodyBytes)
		zr, err := gzip.NewReader(bodyReader)
		if err != nil {
			return err
		}
		defer zr.Close()
		degzippedBytes, err := io.ReadAll(zr)
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
			if nameToCompare == binName {
				// Write the file
				binBytes, err := io.ReadAll(tr)
				if err != nil {
					return err
				}
				writePath := savePath
				if binName == "puff" {
					writePath = tempPath
				}
				fmt.Printf("writing %s to %s\n", binName, writePath)
				err = os.WriteFile(writePath, binBytes, 0750)
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
		}
		return errors.New("binary not found in tar.gz archive")
	} else {
		// save directly
		writePath := savePath
		if binName == "puff" {
			writePath = tempPath
		}
		fmt.Printf("writing %s to %s\n", binName, writePath)
		err := os.WriteFile(writePath, bodyBytes, 0750)
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
	}
	return nil
}

// downloads a binary and puts into bin directory
func DownloadBinary(cfgDir string, repo *Repo, release *Release, ghPat string) error {
	fmt.Printf("downloading %s\n", release.Link)
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
		rawSize := resp.Header.Get("Content-Length")
		size, err := strconv.ParseInt(rawSize, 10, 64)
		if err != nil {
			return err
		}
		bodyBytes := make([]byte, size)
		offset := 0
		buf := make([]byte, 65536)
		for offset < int(size) {
			percent := float64(offset) / float64(size) * 100
			fmt.Printf("\r%.2f%%", percent)
			n, err := resp.Body.Read(buf)
			if err != nil && err != io.EOF {
				return err
			}
			if n == 0 {
				break
			}
			copy(bodyBytes[offset:], buf[:n])
			offset += n
		}
		if offset != int(size) {
			return fmt.Errorf("expected %d bytes, got %d", size, offset)
		}
		fmt.Printf("\n")
		binName, err := BinNameFromPath(repo)
		if err != nil {
			return err
		}
		err = saveOrUnpack(
			cfgDir,
			bodyBytes,
			binName,
			strings.Split(release.Link, "/")[len(strings.Split(release.Link, "/"))-1],
		)
		if err != nil {
			return err
		}
	} else {
		return fmt.Errorf("API returned status %v", resp.StatusCode)
	}

	return nil
}

// returns all assets from API for custom repo
func GetLatestReleaseAssets(path string, ghPat string) (*GithubResponse, error) {
	RepoUrl, err := url.JoinPath("https://api.github.com", "repos", path, "releases/latest")
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
		return &releaseJson, nil
	} else {
		return nil, err
	}
}
