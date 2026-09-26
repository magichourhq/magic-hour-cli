package workflow

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
)

func ValidateOptions(opts Options, body map[string]any) error {
	if opts.Output != "" && (opts.NoWait || opts.NoDownload) {
		return fmt.Errorf("--output cannot be combined with --no-wait or --no-download")
	}
	if opts.Output == "" {
		return nil
	}
	info, err := os.Stat(opts.Output)
	if err == nil {
		if info.IsDir() {
			return nil
		}
		return fmt.Errorf("output already exists: %s", opts.Output)
	}
	if !os.IsNotExist(err) {
		return err
	}
	if count, ok := body["image_count"].(float64); ok && count > 1 {
		return fmt.Errorf("--output must be an existing directory when --count exceeds one")
	}
	parent, err := os.Stat(filepath.Dir(opts.Output))
	if err != nil || !parent.IsDir() {
		return fmt.Errorf("output directory does not exist: %s", filepath.Dir(opts.Output))
	}
	return nil
}

var safeID = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)
var safeExtension = regexp.MustCompile(`^\.[A-Za-z0-9]{1,8}$`)

func (r Runner) Download(ctx context.Context, result Result, output string) (Result, error) {
	if result.Status != "complete" {
		return result, fmt.Errorf("project %s is %s; wait for completion first", result.ID, result.Status)
	}
	if len(result.Outputs) == 0 {
		return result, fmt.Errorf("project %s has no output files", result.ID)
	}
	if !safeID.MatchString(result.ID) {
		return result, fmt.Errorf("project ID cannot be used as a filename")
	}
	dir := "."
	if info, err := os.Stat(output); err == nil && info.IsDir() {
		dir, output = output, ""
	}
	if output != "" && len(result.Outputs) != 1 {
		return result, fmt.Errorf("--output must be an existing directory for multiple files")
	}
	for i := range result.Outputs {
		file := output
		if file == "" {
			u, err := url.Parse(result.Outputs[i].URL)
			if err != nil {
				return result, fmt.Errorf("invalid output URL")
			}
			ext := path.Ext(u.Path)
			if !safeExtension.MatchString(ext) {
				ext = ".bin"
			}
			file = filepath.Join(dir, fmt.Sprintf("mh-%s-%d%s", result.ID, i+1, ext))
		}
		f, err := os.OpenFile(file, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
		if err != nil {
			return result, fmt.Errorf("create output: %w", err)
		}
		r.note("Downloading " + file)
		err = r.Client.Transfer(ctx, http.MethodGet, result.Outputs[i].URL, nil, 0, f)
		closeErr := f.Close()
		if err != nil || closeErr != nil {
			_ = os.Remove(file)
			if err == nil {
				err = closeErr
			}
			return result, fmt.Errorf("download %s: %w", result.ID, err)
		}
		absolute, err := filepath.Abs(file)
		if err != nil {
			return result, err
		}
		result.Outputs[i].Path = absolute
	}
	return result, nil
}
