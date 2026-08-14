package config

import (
	"fmt"
	"os"
	"path/filepath"
)

const defaultStoreSubPath = ".rido/store"

type Config struct {
	StoreLocation string
}

func New() (Config, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return Config{}, fmt.Errorf("could not resolve home directory: %w", err)
	}

	// TODO: read from config file `.config/rido/config.json` instead of using default config.
	return Config{StoreLocation: filepath.Join(home, defaultStoreSubPath)}, nil
}
