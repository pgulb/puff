package puff

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// creates puff config directory
func MustCreateCfgDir() string {
	userDir, err := os.UserConfigDir()
	if err != nil {
		fmt.Printf("error getting user config dir: %v", err)
		os.Exit(1)
	}
	cfgDir := filepath.Join(userDir, "puff")
	err = os.Mkdir(cfgDir, 0750)
	if err != nil {
		if errors.Is(err, fs.ErrExist) {
			return cfgDir
		}
		fmt.Printf("error creating %s config directory: %v", cfgDir, err)
		os.Exit(1)
	} else {
		fmt.Printf("Created config directory %s\n", cfgDir)
	}
	return cfgDir
}

// reads Github PAT from file
func GetGhPat(cfgDir string) (string, error) {
	ghPat, err := os.ReadFile(filepath.Join(cfgDir, "gh_pat"))
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return "", nil
		}
		return "", err
	}
	return string(ghPat), nil
}

// asks for Github PAT and writes it to file
func PromptForGhPat(cfgDir string) error {
	var ghPat string
	fmt.Print("Enter your github personal access token: ")
	fmt.Scanln(&ghPat)
	err := os.WriteFile(filepath.Join(cfgDir, "gh_pat"), []byte(ghPat), 0600)
	if err != nil {
		return err
	}
	fmt.Printf("gh_pat written to %s\n", filepath.Join(cfgDir, "gh_pat"))
	return nil
}

// asks if user wants to add puff bin directory to PATH,
// for .bashrc and .zshrc
func PromptForAddToPath(cfgDir string) error {
	var path string
	shells := []string{".bashrc", ".zshrc"}
	for _, shell := range shells {
		fmt.Printf("Add puff directory to PATH in ~/%s? (y/n): ", shell)
		fmt.Scanln(&path)
		if path == "y" {
			home, err := os.UserHomeDir()
			if err != nil {
				return err
			}
			fmt.Printf("adding puff bin dir to PATH in %s\n", shell)
			binDir := filepath.Join(cfgDir, "bin")
			f, err := os.OpenFile(filepath.Join(home, shell), os.O_APPEND|os.O_WRONLY, 0600)
			if err != nil {
				return err
			}
			defer f.Close()
			_, err = fmt.Fprintf(f, "export PATH=$PATH:%s\n", binDir)
			if err != nil {
				return err
			}
		}
	}
	err := os.WriteFile(filepath.Join(cfgDir, "path_asked"), []byte(""), 0600)
	if err != nil {
		return err
	}
	return nil
}

// checks if user was prompted for adding puff bin directory to PATH
func WasPromptedForPath(cfgDir string) (bool, error) {
	_, err := os.Stat(filepath.Join(cfgDir, "path_asked"))
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return false, nil
		}
		fmt.Println("user was not prompted for adding to PATH yet")
		return false, err
	}
	return true, nil
}

// creates puff bin directory
func MustCreateBinDir(cfgDir string) error {
	binDir := filepath.Join(cfgDir, "bin")
	err := os.Mkdir(binDir, 0750)
	if err != nil {
		if errors.Is(err, fs.ErrExist) {
			return nil
		}
		return err
	} else {
		fmt.Printf("Created bin directory %s\n", binDir)
	}
	return nil
}

// saves skeleton metadata.json if no file exists
func MaybeCreateMetadata(cfgDir string) error {
	metadataFile := filepath.Join(cfgDir, "metadata.json")
	_, err := os.Stat(metadataFile)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			metadata := &MetadataList{}
			fmt.Println("saving skeleton metadata.json")
			err = SaveMetadata(metadata, cfgDir)
			if err != nil {
				return err
			}
		}
		return err
	}
	return nil
}
