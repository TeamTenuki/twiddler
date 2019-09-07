package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

var (
	ErrNoHomeDir = errors.New("no home directory to store config at")
	ErrConfigDir = errors.New("config directory isn't a directory")
)

func Dir() (string, error) {
	homedir, exists := os.LookupEnv("HOME")
	if !exists {
		return "", ErrNoHomeDir
	}

	configDir := filepath.Join(homedir, ".config", "twiddler")
	fi, err := os.Lstat(configDir)
	if os.IsNotExist(err) {
		if err := os.MkdirAll(configDir, 0777); err != nil {
			return "", fmt.Errorf("Can't create config dir: %w", err)
		}
		fi, err = os.Lstat(configDir)
	}

	if err != nil {
		return "", fmt.Errorf("Can't find config directory: %w", err)
	}

	if !fi.IsDir() {
		return "", fmt.Errorf("Config directory isn't a directory")
	}

	return configDir, nil
}
