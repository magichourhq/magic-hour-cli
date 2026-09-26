package workflow

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

var extensions = map[string]string{
	"png": "image", "jpg": "image", "jpeg": "image", "jfif": "image", "heic": "image", "heif": "image", "webp": "image", "avif": "image", "jp2": "image", "tiff": "image", "tif": "image", "bmp": "image",
	"mp4": "video", "m4v": "video", "mov": "video", "webm": "video",
	"mp3": "audio", "wav": "audio", "aac": "audio", "flac": "audio", "weba": "audio", "m4a": "audio", "opus": "audio", "ogg": "audio", "oga": "audio", "aiff": "audio", "amr": "audio",
}

// inspectInput distinguishes local files from URLs and durable API file paths.
// All inputs are inspected before the first upload starts.
func inspectInput(value, expected string) (local bool, kind string, err error) {
	if value == "" {
		return false, "", fmt.Errorf("empty file input")
	}
	if strings.Contains(value, "://") {
		u, e := url.Parse(value)
		if e != nil || (u.Scheme != "https" && u.Scheme != "http") || u.Host == "" || u.User != nil {
			return false, "", fmt.Errorf("invalid media URL")
		}
		kind = extensions[strings.ToLower(strings.TrimPrefix(filepath.Ext(u.Path), "."))]
		if expected != "media" && kind != "" && kind != expected {
			return false, "", fmt.Errorf("expected %s input, URL has %s extension", expected, kind)
		}
		return false, expected, nil
	}
	info, e := os.Stat(value)
	if os.IsNotExist(e) && (strings.HasPrefix(value, "api-assets/") || strings.HasPrefix(value, "image/") || strings.HasPrefix(value, "video/") || strings.HasPrefix(value, "audio/")) {
		return false, expected, nil
	}
	if e != nil {
		return false, "", fmt.Errorf("read input %s: %w", value, e)
	}
	if !info.Mode().IsRegular() || info.Size() == 0 {
		return false, "", fmt.Errorf("input must be a nonempty regular file: %s", value)
	}
	ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(value), "."))
	kind = extensions[ext]
	if expected == "audio" && ext == "webm" {
		kind = "audio"
	}
	if kind == "" {
		return false, "", fmt.Errorf("unsupported input extension .%s", ext)
	}
	if expected != "media" && expected != kind {
		return false, "", fmt.Errorf("expected %s input, got %s: %s", expected, kind, value)
	}
	return true, kind, nil
}

func (r Runner) Upload(ctx context.Context, value, expected string) (string, error) {
	local, kind, err := inspectInput(value, expected)
	if err != nil || !local {
		return value, err
	}
	f, err := os.Open(value)
	if err != nil {
		return "", err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return "", err
	}
	var response struct {
		Items []struct {
			URL  string `json:"upload_url"`
			Path string `json:"file_path"`
		} `json:"items"`
	}
	body := map[string]any{"items": []map[string]string{{"type": kind, "extension": strings.ToLower(strings.TrimPrefix(filepath.Ext(value), "."))}}}
	r.note("Uploading " + value)
	if err := r.Client.Do(ctx, http.MethodPost, "/v1/files/upload-urls", body, &response); err != nil {
		return "", err
	}
	if len(response.Items) != 1 || response.Items[0].Path == "" {
		return "", fmt.Errorf("invalid upload URL response")
	}
	item := response.Items[0]
	if err := r.Client.Transfer(ctx, http.MethodPut, item.URL, f, info.Size(), nil); err != nil {
		return "", err
	}
	return item.Path, nil
}

func cloneValues(values map[string][]string) map[string][]string {
	copy := make(map[string][]string, len(values))
	for k, v := range values {
		copy[k] = append([]string{}, v...)
	}
	return copy
}
