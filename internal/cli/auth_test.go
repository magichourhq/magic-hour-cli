package cli

import (
	"bytes"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/magichourhq/magic-hour-cli/internal/config"
)

type authTransport func(*http.Request) (*http.Response, error)

func (f authTransport) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

func TestAuthLoginStatusLogout(t *testing.T) {
	previous := http.DefaultTransport
	http.DefaultTransport = authTransport(func(req *http.Request) (*http.Response, error) {
		status, body := http.StatusOK, `{"id":"account-1"}`
		if req.Method != http.MethodGet || req.URL.Path != "/v1/account" || req.Header.Get("Authorization") != "Bearer valid-key" {
			status, body = http.StatusUnauthorized, `{"code":"unauthorized","message":"Invalid key"}`
		}
		return &http.Response{StatusCode: status, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}, nil
	})
	t.Cleanup(func() { http.DefaultTransport = previous })
	path := filepath.Join(t.TempDir(), "config.json")
	t.Setenv("MAGIC_HOUR_CONFIG", path)
	t.Setenv("MAGIC_HOUR_API_KEY", "")

	run := func(input string, args ...string) (string, error) {
		cmd := New("test")
		var out bytes.Buffer
		cmd.SetOut(&out)
		cmd.SetIn(strings.NewReader(input))
		cmd.SetArgs(args)
		err := cmd.Execute()
		return out.String(), err
	}

	out, err := run("valid-key\n", "login", "--key-stdin", "--format", "json")
	if err != nil || !strings.Contains(out, `"account_id": "account-1"`) || strings.Contains(out, "valid-key") {
		t.Fatalf("login: output %q, error %v", out, err)
	}
	if key, err := config.Key(); err != nil || key != "valid-key" {
		t.Fatalf("saved key: %q, %v", key, err)
	}
	out, err = run("", "whoami")
	if err != nil || !strings.Contains(out, "Account:  account-1") {
		t.Fatalf("status: output %q, error %v", out, err)
	}
	if _, err := run("invalid-key\n", "login", "--key-stdin"); err == nil {
		t.Fatal("invalid login succeeded")
	}
	if key, err := config.Key(); err != nil || key != "valid-key" {
		t.Fatalf("invalid login replaced saved key: %q, %v", key, err)
	}
	if _, err := run("", "logout"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("logout left config: %v", err)
	}
}
