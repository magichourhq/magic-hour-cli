package api

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

func TestDoCreateUsesAPIKeyAndJSON(t *testing.T) {
	client := New("test-key")
	client.http.Transport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodPost || req.URL.String() != "https://api.magichour.ai/v1/job" {
			t.Errorf("unexpected request: %s %s", req.Method, req.URL)
		}
		if req.Header.Get("Authorization") != "Bearer test-key" || req.Header.Get("Content-Type") != "application/json" {
			t.Errorf("missing API headers: %v", req.Header)
		}
		data, _ := io.ReadAll(req.Body)
		if string(data) != `{"prompt":"mountain"}` {
			t.Errorf("unexpected body: %s", data)
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"id":"job-1"}`)), Header: make(http.Header)}, nil
	})
	var result struct {
		ID string `json:"id"`
	}
	if err := client.Do(context.Background(), http.MethodPost, "/v1/job", map[string]string{"prompt": "mountain"}, &result); err != nil {
		t.Fatal(err)
	}
	if result.ID != "job-1" {
		t.Fatalf("got project ID %q", result.ID)
	}
}

func TestProjectPathRejectsUnsafeIDs(t *testing.T) {
	if path, err := ProjectPath("image", "job-1"); err != nil || path != "/v1/image-projects/job-1" {
		t.Fatalf("got %q, %v", path, err)
	}
	for _, id := range []string{"", "../other", "a?b"} {
		if _, err := ProjectPath("image", id); err == nil {
			t.Errorf("accepted unsafe project ID %q", id)
		}
	}
}
