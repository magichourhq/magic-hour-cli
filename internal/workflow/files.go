package workflow

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/magichourhq/cli/internal/catalog"
)

var extensions = map[string]string{
	"gif": "gif",
	"png": "image", "jpg": "image", "jpeg": "image", "jfif": "image", "heic": "image", "heif": "image", "webp": "image", "avif": "image", "jp2": "image", "tiff": "image", "tif": "image", "bmp": "image",
	"mp4": "video", "m4v": "video", "mov": "video", "webm": "video",
	"mp3": "audio", "wav": "audio", "aac": "audio", "flac": "audio", "weba": "audio", "m4a": "audio", "opus": "audio", "ogg": "audio", "oga": "audio", "aiff": "audio", "amr": "audio",
}

func ValidateInput(value, kind string) error { _, _, err := inspectInput(value, kind); return err }

func AcceptsMedia(expected, actual string) bool {
	if actual == "gif" {
		return expected == "video-or-gif"
	}
	return expected == "media" || expected == actual || (expected == "video-or-gif" && (actual == "video" || actual == "gif"))
}

func MediaKind(value string) string {
	if u, err := url.Parse(value); err == nil && u.Scheme != "" {
		value = u.Path
	}
	return extensions[strings.ToLower(strings.TrimPrefix(filepath.Ext(value), "."))]
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
		kind = MediaKind(u.Path)
		if expected == "audio" && strings.EqualFold(filepath.Ext(u.Path), ".webm") {
			kind = "audio"
		}
		if kind != "" && !AcceptsMedia(expected, kind) {
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
	if !AcceptsMedia(expected, kind) {
		return false, "", fmt.Errorf("expected %s input, got %s: %s", expected, kind, value)
	}
	return true, kind, nil
}

func (r Runner) Upload(ctx context.Context, value, expected string) (string, error) {
	local, kind, err := inspectInput(value, expected)
	if err != nil || !local {
		return value, err
	}
	// Upload URLs accept GIF extensions under image/video, not a separate gif type.
	if kind == "gif" {
		kind = "image"
		if expected == "video-or-gif" {
			kind = "video"
		}
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

func (r Runner) prepareFiles(ctx context.Context, op catalog.Operation, body map[string]any) error {
	// Run a read-only validation pass before any presigned URLs or uploads.
	for _, upload := range []bool{false, true} {
		for _, file := range op.Files {
			_, err := visitFiles(body, file.Path, func(value string) (string, error) {
				if upload {
					return r.Upload(ctx, value, file.Kind)
				}
				_, _, err := inspectInput(value, file.Kind)
				return value, err
			})
			if err != nil {
				return fmt.Errorf("%s: %w", strings.Join(file.Path, "."), err)
			}
		}
	}
	return nil
}

func visitFiles(node any, path []string, fn func(string) (string, error)) (any, error) {
	if len(path) == 0 {
		value, ok := node.(string)
		if !ok {
			return nil, fmt.Errorf("file reference must be a string")
		}
		return fn(value)
	}
	if path[0] == "*" {
		items, ok := node.([]any)
		if !ok {
			return nil, fmt.Errorf("file references must be an array")
		}
		for i, v := range items {
			next, err := visitFiles(v, path[1:], fn)
			if err != nil {
				return nil, err
			}
			items[i] = next
		}
		return items, nil
	}
	object, ok := node.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("file parent must be an object")
	}
	value, exists := object[path[0]]
	if !exists {
		return node, nil
	}
	next, err := visitFiles(value, path[1:], fn)
	if err != nil {
		return nil, err
	}
	object[path[0]] = next
	return object, nil
}
