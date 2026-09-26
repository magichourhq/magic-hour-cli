package cli

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestImageGenerateDryRun(t *testing.T) {
	cmd := New("test")
	var output bytes.Buffer
	cmd.SetOut(&output)
	cmd.SetArgs([]string{"image", "generate", "--prompt", "mountain", "--dry-run"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	var request struct {
		Method string `json:"method"`
		Path   string `json:"path"`
		Body   struct {
			Count float64 `json:"image_count"`
			Style struct {
				Prompt string `json:"prompt"`
			} `json:"style"`
		} `json:"body"`
	}
	if err := json.Unmarshal(output.Bytes(), &request); err != nil {
		t.Fatal(err)
	}
	if request.Method != "POST" || request.Path != "/v1/ai-image-generator" || request.Body.Count != 1 || request.Body.Style.Prompt != "mountain" {
		t.Fatalf("unexpected request: %+v", request)
	}
}

func TestImageGenerateRejectsInvalidInputs(t *testing.T) {
	for _, tc := range []struct {
		name, flag, want string
	}{
		{"missing prompt", "", "missing --prompt"},
		{"invalid model", "--prompt mountain --model nonexistent", "--model"},
		{"invalid count", "--prompt mountain --count 0", "--count"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cmd := New("test")
			cmd.SetArgs(append([]string{"image", "generate", "--dry-run"}, strings.Fields(tc.flag)...))
			if err := cmd.Execute(); err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("got %v, want error containing %q", err, tc.want)
			}
		})
	}
}
