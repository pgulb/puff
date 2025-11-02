package puff

import (
	"errors"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
)

// opens logs file, sets logging to it and returns file
func MustSetupLog(cfgDir string) *os.File {
	logFile := filepath.Join(cfgDir, "puff.log")
	f, err := os.OpenFile(logFile, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("error opening file: %v", err)
	}
	log.SetOutput(f)
	return f
}

// creates puff config directory
func MustCreateCfgDir() string {
	userDir, err := os.UserConfigDir()
	if err != nil {
		log.Fatalf("error getting user config dir: %v", err)
	}
	cfgDir := filepath.Join(userDir, "puff")
	err = os.Mkdir(cfgDir, 0750)
	if err != nil {
		if errors.Is(err, fs.ErrExist) {
			return cfgDir
		}
		log.Fatalf("error creating %s config directory: %v", cfgDir, err)
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
	log.Printf("gh_pat written to %s\n", filepath.Join(cfgDir, "gh_pat"))
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
			log.Printf("adding puff bin dir to PATH in %s\n", shell)
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
		log.Println("user was not prompted for adding to PATH yet")
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
		log.Printf("Created bin directory %s\n", binDir)
		fmt.Printf("Created bin directory %s\n", binDir)
	}
	return nil
}
