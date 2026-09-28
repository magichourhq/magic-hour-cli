package cli

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"text/tabwriter"
	"time"

	"github.com/magichourhq/magic-hour-cli/internal/api"
	"github.com/magichourhq/magic-hour-cli/internal/config"
	"github.com/magichourhq/magic-hour-cli/internal/workflow"
	"github.com/spf13/cobra"
	"golang.org/x/term"
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
	progress, finishProgress := newProgress(cmd.ErrOrStderr(), progressTitle(cmd))
	runner := workflow.Runner{Client: api.New(key), Progress: progress}
	result, err := fn(ctx, runner)
	finishProgress(err)
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

func progressTitle(cmd *cobra.Command) string {
	if cmd.Parent() == nil {
		return "Project"
	}
	kind := cmd.Parent().Name()
	kind = strings.ToUpper(kind[:1]) + kind[1:]
	switch cmd.Name() {
	case "generate":
		return kind + " generation"
	case "edit":
		return kind + " edit"
	default:
		return kind + " project"
	}
}

func newProgress(w io.Writer, title string) (func(string, string), func(error)) {
	plain := func(message, _ string) { fmt.Fprintln(w, message) }
	file, ok := w.(*os.File)
	if !ok || !term.IsTerminal(int(file.Fd())) || os.Getenv("TERM") == "" || os.Getenv("TERM") == "dumb" {
		return plain, func(error) {}
	}

	frames := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	var mu sync.Mutex
	message := ""
	shown := false
	started := time.Time{}
	frame := 0
	draw := func() {
		fmt.Fprintf(w, "\r\x1b[2K  %s %s · %s", frames[frame%len(frames)], message, time.Since(started).Truncate(time.Second))
	}
	complete := func(mark string) {
		if message == "" {
			return
		}
		label := message
		if mark == "✓" {
			if strings.HasPrefix(label, "Downloading ") {
				label = "Downloaded " + strings.TrimPrefix(label, "Downloading ")
			} else if strings.HasPrefix(label, "Uploading ") {
				label = "Uploaded " + strings.TrimPrefix(label, "Uploading ")
			}
		}
		fmt.Fprintf(w, "\r\x1b[2K  %s %s · %s\n", mark, label, time.Since(started).Truncate(time.Second))
	}
	progress := func(_, next string) {
		mu.Lock()
		defer mu.Unlock()
		if !shown {
			fmt.Fprintln(w, title)
			shown = true
		}
		if strings.HasPrefix(message, "Creating ") && strings.HasPrefix(next, "Created ") {
			fmt.Fprintf(w, "\r\x1b[2K  ✓ %s\n", next)
			message = ""
			return
		}
		if message != "" {
			mark := "✓"
			if strings.HasPrefix(message, "Poll failed") {
				mark = "!"
			}
			complete(mark)
		}
		message, started, frame = next, time.Now(), 0
		draw()
	}
	stop := make(chan struct{})
	stopped := make(chan struct{})
	go func() {
		defer close(stopped)
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				mu.Lock()
				if message != "" {
					frame++
					draw()
				}
				mu.Unlock()
			case <-stop:
				return
			}
		}
	}()
	finish := func(err error) {
		close(stop)
		<-stopped
		mu.Lock()
		defer mu.Unlock()
		if message != "" {
			mark := "✓"
			if err != nil {
				mark = "✗"
			} else if strings.HasPrefix(message, "Poll failed") {
				mark = "!"
			}
			complete(mark)
		}
		if shown {
			fmt.Fprintln(w)
		}
	}
	return progress, finish
}

func printResult(cmd *cobra.Command, result workflow.Result) error {
	format, _ := cmd.Flags().GetString("format")
	if format == "json" {
		return writeJSON(cmd, result)
	}
	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
	if result.Error != "" {
		fmt.Fprintf(w, "Error:\t%s\n", result.Error)
	} else {
		for _, output := range result.Outputs {
			if output.Path != "" {
				fmt.Fprintf(w, "File:\t%s\n", output.Path)
			} else {
				fmt.Fprintf(w, "URL:\t%s\n", output.URL)
			}
		}
	}
	if result.CreditsCharged != nil {
		fmt.Fprintf(w, "Credits charged:\t%d\n", *result.CreditsCharged)
	}
	fmt.Fprintf(w, "Project ID:\t%s\n", result.ID)
	if result.Error == "" && (result.Status != "complete" || len(result.Outputs) == 0) {
		fmt.Fprintf(w, "Status:\t%s\n", result.Status)
	}
	return w.Flush()
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
