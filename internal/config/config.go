package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

const (
	defaultStoreSubPath = ".rido/store"
	configFolder        = "rido"
	configFilename      = "config.json"
)

type Config struct {
	StoreRoot string `json:"storeRoot"`
}

func New() (Config, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return emptyConfig(), fmt.Errorf("could not resolve home directory: %w", err)
	}
	cfg := Config{StoreRoot: filepath.Join(home, defaultStoreSubPath)}

	confDir, err := os.UserConfigDir()
	if err != nil {
		// Not having a user config dir is not an issue. We just use default config.
		return cfg, nil
	}

	path := filepath.Join(confDir, configFolder, configFilename)
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return cfg, nil
	}
	if err != nil {
		return emptyConfig(), fmt.Errorf("could not read %s: %w", path, err)
	}

	var fileCfg Config
	if err := json.Unmarshal(data, &fileCfg); err != nil {
		return emptyConfig(), fmt.Errorf("could not parse %s: %w", path, err)
	}
	if fileCfg.StoreRoot != "" {
		cfg.StoreRoot = expandHome(fileCfg.StoreRoot, home)
	}

	return cfg, nil
}

// emptyConfig returns `Config{}`. It is meant to be used on error returns, where
// the caller must read the error and not the value.
// It is used mainly to avoid the `exhaustruct` linter to complain every time we
// return an empty config struct.
func emptyConfig() (cfg Config) {
	return
}

func expandHome(path, home string) string {
	if path == "~" {
		return home
	}
	if after, ok := strings.CutPrefix(path, "~/"); ok {
		return filepath.Join(home, after)
	}

	return path
}
