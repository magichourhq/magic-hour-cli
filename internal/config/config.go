package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// Key gives the environment precedence over the saved configuration.
func Key() (string, error) {
	if key := strings.TrimSpace(os.Getenv("MAGIC_HOUR_API_KEY")); key != "" {
		return key, nil
	}
	path, err := Path()
	if err != nil {
		return "", err
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
	return "", fmt.Errorf("missing API key; run mh login or set MAGIC_HOUR_API_KEY")
}

func Path() (string, error) {
	if path := os.Getenv("MAGIC_HOUR_CONFIG"); path != "" {
		return path, nil
	}
	base := os.Getenv("XDG_CONFIG_HOME")
	if base == "" {
		if runtime.GOOS == "windows" {
			base, err := os.UserConfigDir()
			if err != nil {
				return "", err
			}
			return filepath.Join(base, "magic-hour", "config.json"), nil
		}
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		base = filepath.Join(home, ".config")
	}
	return filepath.Join(base, "magic-hour", "config.json"), nil
}

func Save(key string) error {
	path, err := Path()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(path), ".config-*")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	err = json.NewEncoder(f).Encode(map[string]string{"api_key": key})
	closeErr := f.Close()
	if err != nil {
		return err
	}
	if closeErr != nil {
		return closeErr
	}
	return os.Rename(f.Name(), path)
}

func Logout() error {
	path, err := Path()
	if err != nil {
		return err
	}
	err = os.Remove(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}
