// Package api handles authenticated JSON requests. Asset transfers use separate,
// unauthenticated requests so API credentials never accompany download URLs.
package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Client struct {
	key  string
	http *http.Client
}

func New(key string) *Client {
	return &Client{key: key, http: &http.Client{Timeout: 2 * time.Minute, CheckRedirect: func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}}}
}

type Error struct {
	Status  int    `json:"status"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (e *Error) Error() string { return fmt.Sprintf("API %d (%s): %s", e.Status, e.Code, e.Message) }

func (c *Client) Do(ctx context.Context, method, path string, body, result any) error {
	var input io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return err
		}
		input = bytes.NewReader(data)
	}
	req, err := http.NewRequestWithContext(ctx, method, "https://api.magichour.ai"+path, input)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.key)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "magic-hour-cli")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("%s %s: %w", method, path, err)
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20+1))
	if err != nil {
		return err
	}
	if len(data) > 4<<20 {
		return fmt.Errorf("API response exceeds 4 MiB")
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		e := &Error{Status: resp.StatusCode}
		_ = json.Unmarshal(data, e)
		e.Status = resp.StatusCode
		if e.Message == "" {
			e.Message = http.StatusText(resp.StatusCode)
		}
		return e
	}
	if result == nil {
		return nil
	}
	if err := json.Unmarshal(data, result); err != nil {
		return fmt.Errorf("invalid API response: %w", err)
	}
	return nil
}

// Transfer streams an asset without API authentication. Redirects are rejected;
// signed upload and output URLs should point directly to their storage object.
func (c *Client) Transfer(ctx context.Context, method, address string, input io.Reader, size int64, output io.Writer) error {
	u, err := url.Parse(address)
	if err != nil || u.Scheme != "https" || u.Host == "" || u.User != nil {
		return fmt.Errorf("asset URL must be an absolute HTTPS URL without credentials")
	}
	req, err := http.NewRequestWithContext(ctx, method, address, input)
	if err != nil {
		return fmt.Errorf("invalid asset request")
	}
	if input != nil {
		req.ContentLength = size
		req.Header.Set("Content-Type", "application/octet-stream")
	}
	resp, err := c.http.Do(req)
	if err != nil {
		// Signed URL query strings are credentials, so do not print url.Error.
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return fmt.Errorf("asset %s failed for host %s", method, u.Host)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("asset %s returned HTTP %d", method, resp.StatusCode)
	}
	if output != nil {
		_, err = io.Copy(output, resp.Body)
		return err
	}
	return nil
}

func ProjectPath(kind, id string) (string, error) {
	if kind != "image" && kind != "video" && kind != "audio" && kind != "face-detection" {
		return "", fmt.Errorf("unknown project type %q", kind)
	}
	if id == "" || strings.ContainsAny(id, "/\\?#") || id == "." || id == ".." {
		return "", fmt.Errorf("invalid project ID")
	}
	if kind == "face-detection" {
		return "/v1/face-detection/" + url.PathEscape(id), nil
	}
	return "/v1/" + kind + "-projects/" + url.PathEscape(id), nil
}
