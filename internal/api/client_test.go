package api

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
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

func TestTransferUsesCommandContextInsteadOfAPIRequestTimeout(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.Header.Get("Authorization") != "" {
			t.Error("asset request included API credentials")
		}
		time.Sleep(20 * time.Millisecond)
		_, _ = io.WriteString(w, "image")
	}))
	defer server.Close()

	client := New("test-key")
	client.http = server.Client()
	client.http.Timeout = time.Millisecond
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	var output bytes.Buffer
	if err := client.Transfer(ctx, http.MethodGet, server.URL, nil, 0, &output); err != nil {
		t.Fatal(err)
	}
	if output.String() != "image" {
		t.Fatalf("unexpected asset body %q", output.String())
	}
}
