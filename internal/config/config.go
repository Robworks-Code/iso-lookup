package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Config holds user settings persisted to config.json.
type Config struct {
	DocsDir   string `json:"docs_dir"`
	IndexFile string `json:"index_file"`
	Pager     string `json:"pager"`
}

// Dir returns the iso-lookup config/cache directory (XDG-aware).
func Dir() (string, error) {
	base := os.Getenv("XDG_CONFIG_HOME")
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		base = filepath.Join(home, ".config")
	}
	return filepath.Join(base, "iso-lookup"), nil
}

func configPath() (string, error) {
	d, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(d, "config.json"), nil
}

// CachePath returns the path to the built gob index.
func CachePath() (string, error) {
	d, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(d, "catalog.gob"), nil
}

// Load reads config.json, returning a zero-value Config if it does not exist.
func Load() (Config, error) {
	p, err := configPath()
	if err != nil {
		return Config{}, err
	}
	b, err := os.ReadFile(p)
	if os.IsNotExist(err) {
		return Config{}, nil
	}
	if err != nil {
		return Config{}, err
	}
	var c Config
	if err := json.Unmarshal(b, &c); err != nil {
		return Config{}, err
	}
	return c, nil
}

// Save writes config.json, creating the directory if needed.
func Save(c Config) error {
	d, err := Dir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(d, 0o755); err != nil {
		return err
	}
	p := filepath.Join(d, "config.json")
	b, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(p, b, 0o644)
}
