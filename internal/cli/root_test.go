package cli

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/magichourhq/magic-hour-cli/internal/workflow"
	"github.com/spf13/cobra"
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

func TestImageHelpShowsUsableInputs(t *testing.T) {
	for _, tc := range []struct {
		args  []string
		want  []string
		avoid []string
	}{
		{[]string{"--help"}, []string{"Generate and edit images with Magic Hour"}, []string{"video, and audio"}},
		{[]string{"image", "generate", "--help"}, []string{"--aspect-ratio string", "14 choices (use completion", "35 choices (use completion"}, []string{"--aspect-ratio 1:1", "nano-banana-2, gpt-image", "Maximum varies by model. (required)"}},
		{[]string{"image", "edit", "--help"}, []string{"--image stringArray", "repeat for multiple images", "Use - for piped mh JSON"}, []string{"This value is either"}},
	} {
		cmd := New("test")
		var output bytes.Buffer
		cmd.SetOut(&output)
		cmd.SetArgs(tc.args)
		if err := cmd.Execute(); err != nil {
			t.Fatal(err)
		}
		for _, want := range tc.want {
			if !strings.Contains(output.String(), want) {
				t.Errorf("%v help missing %q", tc.args, want)
			}
		}
		for _, avoid := range tc.avoid {
			if strings.Contains(output.String(), avoid) {
				t.Errorf("%v help contains %q", tc.args, avoid)
			}
		}
	}
}

func TestFailedResultPreservesIDWithoutPublishingOutput(t *testing.T) {
	result := workflow.Result{ID: "job-1", Type: "image", Status: "complete", Error: "download failed", Outputs: []workflow.Output{{URL: "https://example.com/a.png"}}}
	for _, tc := range []struct {
		format, want, avoid string
	}{
		{"text", "job-1\n", "https://example.com/a.png"},
		{"json", `"error": "download failed"`, `"error": ""`},
	} {
		cmd := &cobra.Command{}
		cmd.Flags().String("format", tc.format, "")
		var output bytes.Buffer
		cmd.SetOut(&output)
		if err := printResult(cmd, result); err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(output.String(), tc.want) || strings.Contains(output.String(), tc.avoid) {
			t.Errorf("format %s: got %q", tc.format, output.String())
		}
	}
}

func TestPipedInputDoesNotCancelOperationContext(t *testing.T) {
	root := New("test")
	root.SetIn(strings.NewReader(`{"type":"image","status":"complete","outputs":[{"url":"https://example.com/a.png"}]}`))
	root.SetOut(&bytes.Buffer{})
	root.SetArgs([]string{"image", "edit", "--image", "-", "--prompt", "make it sunset", "--dry-run"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	edit, _, err := root.Find([]string{"image", "edit"})
	if err != nil {
		t.Fatal(err)
	}
	if err := edit.Context().Err(); err != nil {
		t.Fatalf("stdin canceled the downstream operation context: %v", err)
	}
}
