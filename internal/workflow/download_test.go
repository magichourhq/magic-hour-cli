package workflow

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidateOptionsBeforeGeneration(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "image.png")
	if err := ValidateOptions(Options{Output: file}, map[string]any{"image_count": float64(1)}); err != nil {
		t.Fatal(err)
	}
	if err := ValidateOptions(Options{Output: file}, map[string]any{"image_count": float64(4)}); err == nil {
		t.Fatal("accepted one filename for four outputs")
	}
	if err := ValidateOptions(Options{Output: file, NoWait: true}, nil); err == nil {
		t.Fatal("accepted output filename with --no-wait")
	}
	if err := os.WriteFile(file, []byte("keep"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := ValidateOptions(Options{Output: file}, nil); err == nil {
		t.Fatal("accepted existing output filename")
	}
}
