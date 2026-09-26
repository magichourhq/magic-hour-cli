package config

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestSavedKeyAndEnvironmentPrecedence(t *testing.T) {
	path := filepath.Join(t.TempDir(), "magic-hour", "config.json")
	t.Setenv("MAGIC_HOUR_CONFIG", path)
	t.Setenv("MAGIC_HOUR_API_KEY", "")
	if err := Save("saved-key"); err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS != "windows" {
		info, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		if info.Mode().Perm() != 0600 {
			t.Fatalf("config permissions: %v", info.Mode().Perm())
		}
	}
	if key, err := Key(); err != nil || key != "saved-key" {
		t.Fatalf("saved key: %q, %v", key, err)
	}
	t.Setenv("MAGIC_HOUR_API_KEY", "env-key")
	if key, err := Key(); err != nil || key != "env-key" {
		t.Fatalf("environment key: %q, %v", key, err)
	}
	if err := Logout(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("saved config still exists: %v", err)
	}
	if key, err := Key(); err != nil || key != "env-key" {
		t.Fatalf("logout affected environment key: %q, %v", key, err)
	}
}
