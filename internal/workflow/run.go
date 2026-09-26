// Package workflow executes operations without depending on Cobra or terminal IO.
package workflow

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/magichourhq/cli/internal/api"
	"github.com/magichourhq/cli/internal/catalog"
)

type Output struct {
	URL       string `json:"url"`
	Path      string `json:"path,omitempty"`
	ExpiresAt string `json:"expires_at,omitempty"`
}

type Result struct {
	ID      string          `json:"id"`
	Type    string          `json:"type"`
	Status  string          `json:"status"`
	Outputs []Output        `json:"outputs"`
	Project json.RawMessage `json:"project,omitempty"`
	Error   string          `json:"error,omitempty"`
}

type Options struct {
	NoWait, NoDownload bool
	Output             string
}

type Runner struct {
	Client *api.Client
	// Progress is optional; recipes can call the runner without terminal output.
	Progress func(string)
}

func (r Runner) note(s string) {
	if r.Progress != nil {
		r.Progress(s)
	}
}

func (r Runner) Generate(ctx context.Context, op catalog.Operation, values map[string][]string, opts Options) (Result, error) {
	result := Result{Type: op.Kind, Outputs: []Output{}}
	body, err := op.Body(values)
	if err != nil {
		return result, err
	}
	if err := ValidateOptions(opts, body); err != nil {
		return result, err
	}
	for _, f := range op.Fields {
		if f.FileKind == "" {
			continue
		}
		for _, value := range values[f.Flag] {
			if _, _, err := inspectInput(value, f.FileKind); err != nil {
				return result, fmt.Errorf("--%s: %w", f.Flag, err)
			}
		}
	}
	prepared := cloneValues(values)
	for _, f := range op.Fields {
		if f.FileKind == "" {
			continue
		}
		for i, value := range prepared[f.Flag] {
			ref, err := r.Upload(ctx, value, f.FileKind)
			if err != nil {
				return result, err
			}
			prepared[f.Flag][i] = ref
		}
	}
	body, err = op.Body(prepared)
	if err != nil {
		return result, err
	}
	r.note("Creating " + op.Kind + " project…")
	if err := r.Client.Do(ctx, http.MethodPost, op.Path, body, &result); err != nil {
		return result, fmt.Errorf("create project: %w; request was not retried", err)
	}
	if result.ID == "" {
		return result, fmt.Errorf("create response did not include a project ID; request was not retried")
	}
	result.Status = "queued"
	r.note("Created " + op.Kind + " project " + result.ID)
	if opts.NoWait {
		return result, nil
	}
	return r.Finish(ctx, result.Type, result.ID, opts)
}

func (r Runner) Get(ctx context.Context, kind, id string) (Result, error) {
	result := Result{ID: id, Type: kind, Outputs: []Output{}}
	path, err := api.ProjectPath(kind, id)
	if err != nil {
		return result, err
	}
	var response struct {
		Status    string   `json:"status"`
		Downloads []Output `json:"downloads"`
		Enabled   *bool    `json:"enabled"`
		Error     *struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := r.Client.Do(ctx, http.MethodGet, path, nil, &result.Project); err != nil {
		return result, err
	}
	if err := json.Unmarshal(result.Project, &response); err != nil {
		return result, err
	}
	result.Status = response.Status
	if response.Downloads != nil {
		result.Outputs = response.Downloads
	}
	if response.Enabled != nil && !*response.Enabled {
		return result, fmt.Errorf("project %s was deleted", id)
	}
	if result.Status == "error" || result.Status == "canceled" {
		message := result.Status
		if response.Error != nil {
			message += ": " + response.Error.Message
		}
		return result, fmt.Errorf("project %s: %s", id, message)
	}
	return result, nil
}

func (r Runner) Finish(ctx context.Context, kind, id string, opts Options) (Result, error) {
	lastStatus := ""
	for {
		result, err := r.Get(ctx, kind, id)
		if err != nil {
			return result, err
		}
		if result.Status != lastStatus {
			r.note(id + ": " + result.Status)
			lastStatus = result.Status
		}
		switch result.Status {
		case "complete":
			if !opts.NoDownload {
				return r.Download(ctx, result, opts.Output)
			}
			return result, nil
		case "queued", "rendering":
		default:
			return result, fmt.Errorf("project %s has non-pollable status %q", id, result.Status)
		}
		timer := time.NewTimer(2 * time.Second)
		select {
		case <-ctx.Done():
			timer.Stop()
			return result, fmt.Errorf("waiting for %s: %w; resume with mh %s wait %s", id, ctx.Err(), kind, id)
		case <-timer.C:
		}
	}
}
