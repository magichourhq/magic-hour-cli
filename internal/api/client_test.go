package api

import (
	"context"
	"errors"
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

func TestCreateErrorIsNotRetried(t *testing.T) {
	client := New("test-key")
	calls := 0
	client.http.Transport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		calls++
		return &http.Response{
			StatusCode: 429,
			Body:       io.NopCloser(strings.NewReader(`{"code":"rate_limited","message":"try later"}`)),
			Header:     http.Header{"Retry-After": {"3"}},
		}, nil
	})
	err := client.Do(context.Background(), http.MethodPost, "/v1/job", map[string]string{"prompt": "mountain"}, nil)
	var apiErr *Error
	if calls != 1 || !errors.As(err, &apiErr) || apiErr.Status != 429 || apiErr.RetryAfter != 3*time.Second || !Retryable(err) {
		t.Fatalf("calls=%d, error=%v", calls, err)
	}
	if Retryable(&Error{Status: 400}) {
		t.Fatal("classified invalid request as retryable")
	}
}

func TestTransferDoesNotSendAPIKeyOrPrintSignedURL(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if got := req.Header.Get("Authorization"); got != "" {
			t.Errorf("asset request carried API authorization: %q", got)
		}
		w.WriteHeader(http.StatusForbidden)
	}))
	defer server.Close()
	client := New("api-secret")
	client.http = server.Client()
	err := client.Transfer(context.Background(), http.MethodGet, server.URL+"/asset?signature=signed-secret", nil, 0, io.Discard)
	if err == nil || strings.Contains(err.Error(), "signed-secret") || strings.Contains(err.Error(), "api-secret") {
		t.Fatalf("unsafe transfer error: %v", err)
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
