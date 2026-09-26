package workflow

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/magichourhq/magic-hour-cli/internal/api"
	"github.com/magichourhq/magic-hour-cli/internal/catalog"
)

type transportFunc func(*http.Request) (*http.Response, error)

func (f transportFunc) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

func jsonResponse(status int, body string) *http.Response {
	return &http.Response{StatusCode: status, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}
}

func imageGenerateOperation(t *testing.T) catalog.Operation {
	t.Helper()
	for _, op := range catalog.Operations {
		if op.Group == "image" && op.Name == "generate" {
			return op
		}
	}
	t.Fatal("image generate command missing")
	return catalog.Operation{}
}

func TestRunnerCreatesAndFinishesProject(t *testing.T) {
	original := http.DefaultTransport
	t.Cleanup(func() { http.DefaultTransport = original })
	creates, polls := 0, 0
	http.DefaultTransport = transportFunc(func(req *http.Request) (*http.Response, error) {
		switch {
		case req.Method == http.MethodPost && req.URL.Path == "/v1/ai-image-generator":
			creates++
			var body struct {
				Count float64 `json:"image_count"`
				Style struct {
					Prompt string `json:"prompt"`
				} `json:"style"`
			}
			if err := json.NewDecoder(req.Body).Decode(&body); err != nil || body.Count != 1 || body.Style.Prompt != "mountain" {
				t.Errorf("unexpected creation body: %+v, %v", body, err)
			}
			return jsonResponse(200, `{"id":"job-1"}`), nil
		case req.Method == http.MethodGet && req.URL.Path == "/v1/image-projects/job-1":
			polls++
			return jsonResponse(200, `{"status":"complete","downloads":[{"url":"https://example.com/output.png"}]}`), nil
		default:
			t.Errorf("unexpected request: %s %s", req.Method, req.URL)
			return jsonResponse(404, `{}`), nil
		}
	})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	result, err := (Runner{Client: api.New("test-key")}).Generate(ctx, imageGenerateOperation(t), map[string][]string{"prompt": {"mountain"}}, Options{NoDownload: true})
	if err != nil {
		t.Fatal(err)
	}
	if creates != 1 || polls != 1 || result.ID != "job-1" || result.Status != "complete" || len(result.Outputs) != 1 {
		t.Fatalf("creates=%d, polls=%d, result=%+v", creates, polls, result)
	}
}
