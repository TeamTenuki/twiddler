package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

var (
	ErrNoHomeDir = errors.New("no home directory to store config at")
	ErrConfigDir = errors.New("config directory isn't a directory")
)

type Config struct {
	TwitchClientID string `json:"twitch-client-id"`
	TwitchSecret   string `json:"twitch-secret"`
	DiscordAPI     string `json:"discord-api-key"`
}

// Dir returns default config directory. Currently it is a simply "$HOME/.config/twiddler".
func Dir() (string, error) {
	homedir, exists := os.LookupEnv("HOME")
	if !exists {
		return "", ErrNoHomeDir
	}

	configDir := filepath.Join(homedir, ".config", "twiddler")
	fi, err := os.Stat(configDir)
	if os.IsNotExist(err) {
		if err := os.MkdirAll(configDir, 0777); err != nil {
			return "", fmt.Errorf("can't create config dir: %w", err)
		}
		fi, err = os.Stat(configDir)
	}

	if err != nil {
		return "", fmt.Errorf("can't find config directory: %w", err)
	}

	if !fi.IsDir() {
		return "", ErrConfigDir
	}

	return configDir, nil
}

// Parse tries to parse config from a given path.
// If path is an empty string, it will try to parse from a default config directory (see Dir).
func Parse(path string) (*Config, error) {
	if path == "" {
		configDir, err := Dir()
		if err != nil {
			return nil, err
		}

		path = filepath.Join(configDir, "config.json")
	}

	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	config := Config{}
	if err := json.NewDecoder(f).Decode(&config); err != nil {
		return nil, err
	}

	return &config, nil
}
