package main

import (
	"errors"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
)

func mustSetupLog(cfgDir string) *os.File {
	logFile := filepath.Join(cfgDir, "puff.log")
	f, err := os.OpenFile(logFile, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("error opening file: %v", err)
	}
	log.SetOutput(f)
	return f
}

func mustCreateCfgDir() string {
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

func getGhPat(cfgDir string) (string, error) {
	ghPat, err := os.ReadFile(filepath.Join(cfgDir, "gh_pat"))
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return "", nil
		}
		return "", err
	}
	return string(ghPat), nil
}

func promptForGhPat(cfgDir string) error {
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

func promptForAddToPath(cfgDir string) error {
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

func wasPromptedForPath(cfgDir string) (bool, error) {
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

func mustCreateBinDir(cfgDir string) error {
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

func main() {
	// logging init
	cfgDir := mustCreateCfgDir()
	f := mustSetupLog(cfgDir)
	defer f.Close()
	log.Println(os.Args)

	// handle writing github PAT into file if it doesn't exist
	// else read it
	ghPat, err := getGhPat(cfgDir)
	if err != nil {
		fmt.Printf("error getting gh_pat: %s", err.Error())
		log.Fatalf("error getting gh_pat: %s", err.Error())
	}
	if ghPat == "" {
		err := promptForGhPat(cfgDir)
		if err != nil {
			fmt.Printf("error writing gh_pat to file: %s", err.Error())
			log.Fatalf("error writing gh_pat to file: %s", err.Error())
		}
	}

	// add puff bin directory to PATH if user wants
	// for ~/.bashrc and ~/.zshrc
	err = mustCreateBinDir(cfgDir)
	if err != nil {
		fmt.Println(err.Error())
		log.Fatal(err.Error())
	}
	prompted, err := wasPromptedForPath(cfgDir)
	if err != nil {
		fmt.Println(err.Error())
		log.Fatal(err.Error())
	}
	if !prompted {
		err = promptForAddToPath(cfgDir)
		if err != nil {
			fmt.Println(err.Error())
			log.Fatal(err.Error())
		}
	}
}
