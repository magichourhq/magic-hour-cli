package cli

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/magichourhq/magic-hour-cli/internal/api"
	"github.com/magichourhq/magic-hour-cli/internal/config"
	"github.com/magichourhq/magic-hour-cli/internal/workflow"
	"github.com/spf13/cobra"
)

func executionFlags(cmd *cobra.Command) (*workflow.Options, *time.Duration) {
	opts := &workflow.Options{}
	cmd.Flags().BoolVar(&opts.NoWait, "no-wait", false, "Return the project ID without waiting")
	cmd.Flags().BoolVar(&opts.NoDownload, "no-download", false, "Wait for completion and return output URLs")
	cmd.Flags().StringVar(&opts.Output, "output", "", "Output filename, or an existing directory for multiple files; never overwrite")
	timeout := cmd.Flags().Duration("timeout", 30*time.Minute, "Time limit for stdin and a fresh limit for execution")
	return opts, timeout
}

func execute(cmd *cobra.Command, timeout time.Duration, fn func(context.Context, workflow.Runner) (workflow.Result, error)) error {
	if timeout <= 0 {
		return fmt.Errorf("--timeout must be positive")
	}
	key, err := config.Key()
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(cmd.Context(), timeout)
	defer cancel()
	runner := workflow.Runner{Client: api.New(key), Progress: func(message string) { fmt.Fprintln(cmd.ErrOrStderr(), message) }}
	result, err := fn(ctx, runner)
	if result.ID != "" {
		if err != nil {
			result.Error = err.Error()
		}
		if outErr := printResult(cmd, result); outErr != nil && err == nil {
			return outErr
		}
	}
	return err
}

func printResult(cmd *cobra.Command, result workflow.Result) error {
	format, _ := cmd.Flags().GetString("format")
	if format == "json" {
		return writeJSON(cmd, result)
	}
	if _, err := fmt.Fprintln(cmd.OutOrStdout(), "Project ID:", result.ID); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(cmd.OutOrStdout(), "Status:", result.Status); err != nil {
		return err
	}
	if result.CreditsCharged != nil {
		if _, err := fmt.Fprintln(cmd.OutOrStdout(), "Credits charged:", *result.CreditsCharged); err != nil {
			return err
		}
	}
	if result.Error != "" {
		_, err := fmt.Fprintln(cmd.OutOrStdout(), "Error:", result.Error)
		return err
	}
	for _, output := range result.Outputs {
		value := output.Path
		if value == "" {
			value = output.URL
		}
		if _, err := fmt.Fprintln(cmd.OutOrStdout(), "Output:", value); err != nil {
			return err
		}
	}
	return nil
}

func addManagement(group *cobra.Command, kind string) {
	for _, action := range []string{"get", "wait", "download", "delete"} {
		cmd := &cobra.Command{Use: action + " ID", Short: action + " an existing " + kind + " project", Args: cobra.ExactArgs(1)}
		timeout := cmd.Flags().Duration("timeout", 30*time.Minute, "Maximum time for the operation")
		output := ""
		if action == "download" {
			cmd.Flags().StringVar(&output, "output", "", "Output filename or existing directory; never overwrite")
		}
		cmd.RunE = func(cmd *cobra.Command, args []string) error {
			return execute(cmd, *timeout, func(ctx context.Context, r workflow.Runner) (workflow.Result, error) {
				id := args[0]
				switch action {
				case "get":
					return r.Get(ctx, kind, id)
				case "wait":
					return r.Finish(ctx, kind, id, workflow.Options{NoDownload: true})
				case "download":
					result, err := r.Get(ctx, kind, id)
					if err != nil {
						return result, err
					}
					return r.Download(ctx, result, output)
				case "delete":
					path, err := api.ProjectPath(kind, id)
					if err != nil {
						return workflow.Result{}, err
					}
					result := workflow.Result{ID: id, Type: kind, Outputs: []workflow.Output{}}
					err = r.Client.Do(ctx, http.MethodDelete, path, nil, nil)
					if err == nil {
						result.Status = "deleted"
					}
					return result, err
				}
				return workflow.Result{}, fmt.Errorf("unknown project action")
			})
		}
		group.AddCommand(cmd)
	}
}
