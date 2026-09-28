// Package workflow executes operations without depending on Cobra or terminal IO.
package workflow

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/magichourhq/magic-hour-cli/internal/api"
	"github.com/magichourhq/magic-hour-cli/internal/catalog"
)

type Output struct {
	URL       string `json:"url"`
	Path      string `json:"path,omitempty"`
	ExpiresAt string `json:"expires_at,omitempty"`
}

type Result struct {
	ID             string   `json:"id"`
	Type           string   `json:"type"`
	Status         string   `json:"status"`
	CreditsCharged *int     `json:"credits_charged"`
	Outputs        []Output `json:"outputs"`
	Error          string   `json:"error,omitempty"`
}

type Options struct {
	NoWait, NoDownload bool
	Output             string
}

type Runner struct {
	Client *api.Client
	// Progress is optional; recipes can call the runner without terminal output.
	Progress func(message, label string)
}

func (r Runner) note(s string) {
	r.noteLabel(s, s)
}

func (r Runner) noteLabel(message, label string) {
	if r.Progress != nil {
		r.Progress(message, label)
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
	r.noteLabel("Creating "+op.Kind+" project…", "Creating project…")
	if err := r.Client.Do(ctx, http.MethodPost, op.Path, body, &result); err != nil {
		return result, fmt.Errorf("create project: %w; request was not retried", err)
	}
	if result.ID == "" {
		return result, fmt.Errorf("create response did not include a project ID; request was not retried")
	}
	result.Status = "queued"
	r.noteLabel("Created "+op.Kind+" project "+result.ID, "Created project "+result.ID)
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
		Status         string   `json:"status"`
		CreditsCharged *int     `json:"credits_charged"`
		Downloads      []Output `json:"downloads"`
		Enabled        *bool    `json:"enabled"`
		Error          *struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	var project json.RawMessage
	if err := r.Client.Do(ctx, http.MethodGet, path, nil, &project); err != nil {
		return result, err
	}
	if err := json.Unmarshal(project, &response); err != nil {
		return result, err
	}
	result.Status = response.Status
	result.CreditsCharged = response.CreditsCharged
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
	failedPolls := 0
	for {
		result, err := r.Get(ctx, kind, id)
		delay := 2 * time.Second
		if err != nil {
			if ctx.Err() != nil {
				return result, fmt.Errorf("waiting for %s: %w; resume with mh %s wait %s", id, ctx.Err(), kind, id)
			}
			if !retryablePoll(err) || failedPolls == 3 {
				return result, fmt.Errorf("poll %s: %w; resume with mh %s wait %s", id, err, kind, id)
			}
			failedPolls++
			delay = time.Duration(1<<failedPolls) * time.Second
			r.note(fmt.Sprintf("Poll failed (%d/3); retrying…", failedPolls))
		} else {
			failedPolls = 0
			if result.Status != lastStatus {
				status := result.Status
				if status != "" {
					status = strings.ToUpper(status[:1]) + status[1:]
				}
				r.noteLabel(id+": "+result.Status, status)
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
		}
		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return result, fmt.Errorf("waiting for %s: %w; resume with mh %s wait %s", id, ctx.Err(), kind, id)
		case <-timer.C:
		}
	}
}

func retryablePoll(err error) bool {
	var apiErr *api.Error
	if errors.As(err, &apiErr) {
		return apiErr.Status == http.StatusRequestTimeout || apiErr.Status == http.StatusTooManyRequests || apiErr.Status >= 500
	}
	var urlErr *url.Error
	return errors.As(err, &urlErr)
}
