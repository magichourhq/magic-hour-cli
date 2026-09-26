package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/magichourhq/cli/internal/catalog"
	"github.com/magichourhq/cli/internal/workflow"
)

// resolveStdin only reads a stream when a file flag explicitly contains '-'.
func resolveStdin(ctx context.Context, input io.Reader, op catalog.Operation, values map[string][]string) error {
	var field *catalog.Field
	for i, f := range op.Fields {
		if f.FileKind == "" {
			continue
		}
		for _, value := range values[f.Flag] {
			if value != "-" {
				continue
			}
			if field != nil {
				return fmt.Errorf("only one file input may read stdin")
			}
			field = &op.Fields[i]
		}
	}
	if field == nil {
		return nil
	}
	type readResult struct {
		data []byte
		err  error
	}
	done := make(chan readResult, 1)
	go func() { data, err := io.ReadAll(io.LimitReader(input, (4<<20)+1)); done <- readResult{data, err} }()
	var read readResult
	select {
	case <-ctx.Done():
		return ctx.Err()
	case read = <-done:
	}
	if read.err != nil {
		return read.err
	}
	if len(read.data) > 4<<20 {
		return fmt.Errorf("stdin exceeds 4 MiB")
	}
	var result workflow.Result
	if err := json.Unmarshal(read.data, &result); err != nil {
		return fmt.Errorf("--%s - expects mh JSON output on stdin", field.Flag)
	}
	if result.Status != "complete" {
		return fmt.Errorf("stdin project is not complete (status %q)", result.Status)
	}
	if field.FileKind != "media" && result.Type != field.FileKind {
		return fmt.Errorf("--%s expects %s, stdin contains %s", field.Flag, field.FileKind, result.Type)
	}
	if len(result.Outputs) == 0 {
		return fmt.Errorf("stdin project has no outputs")
	}
	if field.Type != "array" && len(result.Outputs) != 1 {
		return fmt.Errorf("--%s expects one output; select a specific URL or path", field.Flag)
	}
	var refs []string
	for _, output := range result.Outputs {
		ref := output.Path
		if ref == "" {
			ref = output.URL
		}
		if ref == "" {
			return fmt.Errorf("stdin output has no path or URL")
		}
		refs = append(refs, ref)
	}
	var expanded []string
	for _, value := range values[field.Flag] {
		if value == "-" {
			expanded = append(expanded, refs...)
		} else {
			expanded = append(expanded, value)
		}
	}
	values[field.Flag] = expanded
	return nil
}
