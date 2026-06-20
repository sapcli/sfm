// Package internal provides shared CLI helpers for the sfm command tree.
//
// It exposes global pointer vars set by cobra init (Username, Password, etc.),
// MustClient for constructing an authenticated sfm.Client, Print for formatted
// output, and config-file helpers (ReadConfig, WriteConfig) for persisting
// credentials to the user's config directory.
package internal

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Config represents the persisted CLI configuration.
type Config struct {
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
}

// ConfigFilePath returns the platform-appropriate config file path.
//
//	Linux:   ~/.config/sfm/config.json
//	macOS:   ~/Library/Application Support/sfm/config.json
//	Windows: %APPDATA%\sfm\config.json
func ConfigFilePath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("cannot determine config dir: %w", err)
	}
	return filepath.Join(dir, "sfm", "config.json"), nil
}

// ReadConfig reads the config file from disk. Returns nil (no error)
// if the file does not exist.
func ReadConfig() (*Config, error) {
	path, err := ConfigFilePath()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading config: %w", err)
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}
	return &cfg, nil
}

// WriteConfig writes the config to disk with 0600 permissions.
func WriteConfig(cfg *Config) error {
	path, err := ConfigFilePath()
	if err != nil {
		return err
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("creating config dir: %w", err)
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("encoding config: %w", err)
	}
	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("writing config: %w", err)
	}
	return nil
}

// ClearConfig removes the config file from disk.
func ClearConfig() error {
	path, err := ConfigFilePath()
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("removing config: %w", err)
	}
	return nil
}
