package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Key gives the environment precedence over the saved configuration.
func Key() (string, error) {
	if key := strings.TrimSpace(os.Getenv("MAGIC_HOUR_API_KEY")); key != "" {
		return key, nil
	}
	path := os.Getenv("MAGIC_HOUR_CONFIG")
	if path == "" {
		base := os.Getenv("XDG_CONFIG_HOME")
		if base == "" {
			home, err := os.UserHomeDir()
			if err != nil {
				return "", err
			}
			base = filepath.Join(home, ".config")
		}
		path = filepath.Join(base, "magic-hour", "config.json")
	}
	data, err := os.ReadFile(path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return "", fmt.Errorf("read config: %w", err)
	}
	var cfg struct {
		APIKey string `json:"api_key"`
	}
	if len(data) > 0 {
		if err := json.Unmarshal(data, &cfg); err != nil {
			return "", fmt.Errorf("invalid config JSON: %w", err)
		}
	}
	if key := strings.TrimSpace(cfg.APIKey); key != "" {
		return key, nil
	}
	return "", fmt.Errorf("missing API key; set MAGIC_HOUR_API_KEY")
}
