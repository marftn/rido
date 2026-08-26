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

	path := filepath.Join(configDir(home), configFolder, configFilename)
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return cfg, nil
	}
	if err != nil {
		return emptyConfig(), fmt.Errorf("could not read %s: %w", path, err)
	}

	var fileCfg Config
	if e := json.Unmarshal(data, &fileCfg); e != nil {
		return emptyConfig(), fmt.Errorf("could not parse %s: %w", path, e)
	}
	if fileCfg.StoreRoot != "" {
		cfg.StoreRoot = expandHome(fileCfg.StoreRoot, home)
	}

	return cfg, nil
}

// configDir returns the location of the user's config folder: `$XDG_CONFIG_HOME` or
// `~/.config`.
// We do not use `os.UserConfigDir` because it points to `~/Library/Application Support`
// on macOS. Relative `$XDG_CONFIG_HOME` values are ignored so the config path
// does not depend on rido's working directory.
func configDir(home string) string {
	if dir := os.Getenv("XDG_CONFIG_HOME"); filepath.IsAbs(dir) {
		return dir
	}

	return filepath.Join(home, ".config")
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
