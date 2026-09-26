package cli

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"slices"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/magichourhq/cli/internal/api"
	"github.com/magichourhq/cli/internal/config"
	"github.com/magichourhq/cli/internal/workflow"
	"github.com/spf13/cobra"
)

func apiCommand(cmd *cobra.Command, fn func(context.Context, *api.Client) error) error {
	return apiCommandWithTimeout(cmd, 2*time.Minute, fn)
}

func apiCommandWithTimeout(cmd *cobra.Command, timeout time.Duration, fn func(context.Context, *api.Client) error) error {
	if timeout <= 0 {
		return fmt.Errorf("--timeout must be positive")
	}
	key, err := config.Key()
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(cmd.Context(), timeout)
	defer cancel()
	return fn(ctx, api.New(key))
}

func addUtilities(root *cobra.Command) {
	account := &cobra.Command{Use: "account", Short: "Account and credit information"}
	account.AddCommand(&cobra.Command{Use: "get", Short: "Get credits and subscription", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, args []string) error {
		return apiCommand(cmd, func(ctx context.Context, c *api.Client) error {
			var result map[string]any
			if err := c.Do(ctx, http.MethodGet, "/v1/account", nil, &result); err != nil {
				return err
			}
			if jsonMode(cmd) {
				return writeJSON(cmd, result)
			}
			_, err := fmt.Fprintf(cmd.OutOrStdout(), "Account: %v\nTier: %v\nCredits: %v\n", result["id"], result["tier"], result["credits"])
			return err
		})
	}})
	root.AddCommand(account)

	saved := &cobra.Command{Use: "saved-items", Short: "Reusable characters, references, voices, and brand assets"}
	list := &cobra.Command{Use: "list", Short: "List saved items (one page)", Args: cobra.NoArgs}
	kind := list.Flags().String("type", "", "Filter by saved item type")
	limit := list.Flags().Int("limit", 20, "Page size (1–100)")
	cursor := list.Flags().String("cursor", "", "Pagination cursor from the previous response")
	kinds := []string{"character", "reference", "voice", "moodboard", "brand_kit"}
	list.RegisterFlagCompletionFunc("type", choices(kinds))
	list.RunE = func(cmd *cobra.Command, args []string) error {
		if *limit < 1 || *limit > 100 {
			return fmt.Errorf("--limit must be between 1 and 100")
		}
		if *kind != "" && !slices.Contains(kinds, *kind) {
			return fmt.Errorf("invalid --type; choose %v", kinds)
		}
		q := url.Values{"limit": {strconv.Itoa(*limit)}}
		if *kind != "" {
			q.Set("type", *kind)
		}
		if *cursor != "" {
			q.Set("cursor", *cursor)
		}
		return apiCommand(cmd, func(ctx context.Context, c *api.Client) error {
			var result struct {
				Items      []map[string]any `json:"items"`
				NextCursor *string          `json:"next_cursor"`
			}
			if err := c.Do(ctx, http.MethodGet, "/v1/saved-items?"+q.Encode(), nil, &result); err != nil {
				return err
			}
			if jsonMode(cmd) {
				return writeJSON(cmd, result)
			}
			w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 4, 2, ' ', 0)
			fmt.Fprintln(w, "ID\tTYPE\tNAME")
			for _, item := range result.Items {
				fmt.Fprintf(w, "%v\t%v\t%v\n", item["id"], item["type"], item["name"])
			}
			if err := w.Flush(); err != nil {
				return err
			}
			if result.NextCursor != nil {
				fmt.Fprintln(cmd.ErrOrStderr(), "Next page: mh saved-items list --cursor", strconv.Quote(*result.NextCursor))
			}
			return nil
		})
	}
	saved.AddCommand(list)
	root.AddCommand(saved)

	files := &cobra.Command{Use: "files", Short: "Upload reusable media"}
	upload := &cobra.Command{Use: "upload PATH...", Short: "Upload local files and return durable API paths", Args: cobra.MinimumNArgs(1)}
	fileType := upload.Flags().String("type", "media", "Asset type: image, video, audio, or media (infer from extension)")
	timeout := upload.Flags().Duration("timeout", 30*time.Minute, "Maximum time for the entire upload batch")
	upload.RegisterFlagCompletionFunc("type", choices([]string{"image", "video", "audio", "media"}))
	upload.RunE = func(cmd *cobra.Command, args []string) error {
		if !slices.Contains([]string{"image", "video", "audio", "media"}, *fileType) {
			return fmt.Errorf("invalid --type")
		}
		// Inspect every input before performing any upload.
		for _, file := range args {
			info, err := os.Stat(file)
			if err != nil {
				return err
			}
			if !info.Mode().IsRegular() {
				return fmt.Errorf("not a regular file: %s", file)
			}
			if err := workflow.ValidateInput(file, *fileType); err != nil {
				return err
			}
		}
		return apiCommandWithTimeout(cmd, *timeout, func(ctx context.Context, c *api.Client) error {
			r := workflow.Runner{Client: c, Progress: func(s string) { fmt.Fprintln(cmd.ErrOrStderr(), s) }}
			paths := make([]string, 0, len(args))
			for _, file := range args {
				path, err := r.Upload(ctx, file, *fileType)
				if err != nil {
					if len(paths) > 0 {
						if jsonMode(cmd) {
							_ = writeJSON(cmd, map[string]any{"file_paths": paths, "error": err.Error()})
						} else {
							fmt.Fprintln(cmd.OutOrStdout(), strings.Join(paths, "\n"))
						}
					}
					return err
				}
				paths = append(paths, path)
			}
			if jsonMode(cmd) {
				return writeJSON(cmd, map[string]any{"file_paths": paths})
			}
			_, err := fmt.Fprintln(cmd.OutOrStdout(), strings.Join(paths, "\n"))
			return err
		})
	}
	files.AddCommand(upload)
	root.AddCommand(files)
	addAuth(root)
}

func jsonMode(cmd *cobra.Command) bool {
	format, _ := cmd.Flags().GetString("format")
	return format == "json"
}
