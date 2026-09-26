package cli

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
	"unicode"

	"github.com/magichourhq/magic-hour-cli/internal/api"
	"github.com/magichourhq/magic-hour-cli/internal/config"
	"github.com/spf13/cobra"
)

func addAuth(root *cobra.Command) {
	group := &cobra.Command{Use: "auth", Short: "Manage API credentials"}
	login := &cobra.Command{Use: "login", Short: "Validate and save an API key", Args: cobra.NoArgs}
	keyStdin := login.Flags().Bool("key-stdin", false, "Read API key from stdin instead of MAGIC_HOUR_API_KEY")
	login.RunE = func(cmd *cobra.Command, args []string) error {
		ctx, cancel := context.WithTimeout(cmd.Context(), time.Minute)
		defer cancel()
		key := strings.TrimSpace(os.Getenv("MAGIC_HOUR_API_KEY"))
		if *keyStdin {
			data, err := readInput(ctx, cmd.InOrStdin(), 16<<10)
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
		account, err := validateKey(ctx, key)
		if err != nil {
			return err
		}
		if err := config.Save(key); err != nil {
			return err
		}
		return authResult(cmd, "logged_in", account)
	}
	status := &cobra.Command{Use: "status", Short: "Validate active API key", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, args []string) error {
		key, err := config.Key()
		if err != nil {
			return err
		}
		ctx, cancel := context.WithTimeout(cmd.Context(), time.Minute)
		defer cancel()
		account, err := validateKey(ctx, key)
		if err != nil {
			return err
		}
		return authResult(cmd, "authenticated", account)
	}}
	logout := &cobra.Command{Use: "logout", Short: "Remove saved API key", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, args []string) error {
		if err := config.Logout(); err != nil {
			return err
		}
		if strings.TrimSpace(os.Getenv("MAGIC_HOUR_API_KEY")) != "" {
			fmt.Fprintln(cmd.ErrOrStderr(), "Saved key removed; MAGIC_HOUR_API_KEY remains active.")
		}
		return authResult(cmd, "logged_out", "")
	}}
	group.AddCommand(login, status, logout)
	root.AddCommand(group)
}

func validateKey(ctx context.Context, key string) (string, error) {
	var account struct {
		ID string `json:"id"`
	}
	if err := api.New(key).Do(ctx, http.MethodGet, "/v1/account", nil, &account); err != nil {
		return "", err
	}
	return account.ID, nil
}

func authResult(cmd *cobra.Command, status, account string) error {
	format, _ := cmd.Flags().GetString("format")
	if format == "json" {
		return writeJSON(cmd, map[string]string{"status": status, "account_id": account})
	}
	message := strings.ReplaceAll(status, "_", " ")
	if account != "" {
		message += " as " + account
	}
	_, err := fmt.Fprintln(cmd.OutOrStdout(), message)
	return err
}
