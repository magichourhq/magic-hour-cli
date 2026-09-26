package cli

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
	"unicode"

	"github.com/magichourhq/cli/internal/api"
	"github.com/magichourhq/cli/internal/config"
	"github.com/spf13/cobra"
)

func addAuth(root *cobra.Command) {
	group := &cobra.Command{Use: "auth", Short: "Manage API credentials without interactive prompts"}
	login := &cobra.Command{Use: "login", Short: "Validate and save MAGIC_HOUR_API_KEY or a key from stdin", Args: cobra.NoArgs}
	stdin := login.Flags().Bool("key-stdin", false, "Read the API key from stdin instead of the environment")
	login.RunE = func(cmd *cobra.Command, args []string) error {
		ctx, cancel := context.WithTimeout(cmd.Context(), time.Minute)
		defer cancel()
		key := strings.TrimSpace(os.Getenv("MAGIC_HOUR_API_KEY"))
		if *stdin {
			data, err := readInput(ctx, cmd.InOrStdin(), 16384)
			if err != nil {
				return err
			}
			key = strings.TrimSpace(string(data))
		}
		if key == "" {
			return fmt.Errorf("set MAGIC_HOUR_API_KEY or pass --key-stdin")
		}
		if strings.IndexFunc(key, unicode.IsSpace) >= 0 {
			return fmt.Errorf("API key must not contain whitespace")
		}
		var account map[string]any
		if err := api.New(key).Do(ctx, http.MethodGet, "/v1/account", nil, &account); err != nil {
			return err
		}
		if err := config.Save(key); err != nil {
			return err
		}
		return authResult(cmd, "logged_in", account["id"])
	}
	status := &cobra.Command{Use: "status", Short: "Validate the active API key", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, args []string) error {
		return apiCommand(cmd, func(ctx context.Context, c *api.Client) error {
			var account map[string]any
			if err := c.Do(ctx, http.MethodGet, "/v1/account", nil, &account); err != nil {
				return err
			}
			return authResult(cmd, "authenticated", account["id"])
		})
	}}
	logout := &cobra.Command{Use: "logout", Short: "Remove saved credentials (environment keys remain active)", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, args []string) error {
		if err := config.Logout(); err != nil {
			return err
		}
		if strings.TrimSpace(os.Getenv("MAGIC_HOUR_API_KEY")) != "" {
			fmt.Fprintln(cmd.ErrOrStderr(), "Saved key removed; MAGIC_HOUR_API_KEY is still set.")
		}
		return authResult(cmd, "logged_out", nil)
	}}
	group.AddCommand(login, status, logout)
	root.AddCommand(group)
}

func authResult(cmd *cobra.Command, status string, account any) error {
	if jsonMode(cmd) {
		return writeJSON(cmd, map[string]any{"status": status, "account_id": account})
	}
	_, err := fmt.Fprintln(cmd.OutOrStdout(), strings.ReplaceAll(status, "_", " "))
	return err
}
