package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"text/tabwriter"
	"time"
	"unicode"

	"github.com/magichourhq/magic-hour-cli/internal/api"
	"github.com/magichourhq/magic-hour-cli/internal/config"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

func addAuth(root *cobra.Command) {
	root.AddCommand(loginCommand(), whoamiCommand(), logoutCommand())
}

func loginCommand() *cobra.Command {
	login := &cobra.Command{Use: "login", Short: "Log in through your browser", Args: cobra.NoArgs}
	interactive := login.Flags().BoolP("interactive", "i", false, "Paste an API key instead of using your browser")
	keyStdin := login.Flags().Bool("key-stdin", false, "Read an API key from stdin")
	login.RunE = func(cmd *cobra.Command, args []string) error {
		if *interactive && *keyStdin {
			return fmt.Errorf("--interactive and --key-stdin cannot be combined")
		}
		ctx, cancel := context.WithTimeout(cmd.Context(), 5*time.Minute)
		defer cancel()
		var key string
		if *interactive {
			if !term.IsTerminal(int(os.Stdin.Fd())) {
				return fmt.Errorf("--interactive requires a terminal; use --key-stdin or MAGIC_HOUR_API_KEY for automation")
			}
			fmt.Fprint(cmd.ErrOrStderr(), "Enter your API key: ")
			data, err := readPassword(ctx)
			fmt.Fprintln(cmd.ErrOrStderr())
			if err != nil {
				return fmt.Errorf("read API key: %w", err)
			}
			key = strings.TrimSpace(string(data))
		} else if *keyStdin {
			data, err := readInput(ctx, cmd.InOrStdin(), 16<<10)
			if err != nil {
				return err
			}
			key = strings.TrimSpace(string(data))
		} else {
			if !term.IsTerminal(int(os.Stdin.Fd())) {
				return fmt.Errorf("browser login requires a terminal; use MAGIC_HOUR_API_KEY or --key-stdin for automation")
			}
			var err error
			key, err = browserLogin(ctx, cmd.ErrOrStderr())
			if err != nil {
				return err
			}
		}
		if key == "" {
			return fmt.Errorf("API key is empty")
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
		if strings.TrimSpace(os.Getenv("MAGIC_HOUR_API_KEY")) != "" {
			fmt.Fprintln(cmd.ErrOrStderr(), "Saved key; MAGIC_HOUR_API_KEY remains active and takes precedence.")
		}
		return authResult(cmd, "logged_in", account.ID)
	}
	return login
}

func whoamiCommand() *cobra.Command {
	return &cobra.Command{Use: "whoami", Short: "Show the account for the active API key", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, args []string) error {
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
		return whoamiResult(cmd, account)
	}}
}

func readPassword(ctx context.Context) ([]byte, error) {
	fd := int(os.Stdin.Fd())
	state, err := term.GetState(fd)
	if err != nil {
		return nil, err
	}
	type result struct {
		value []byte
		err   error
	}
	done := make(chan result, 1)
	go func() {
		value, err := term.ReadPassword(fd)
		done <- result{value, err}
	}()
	select {
	case read := <-done:
		return read.value, read.err
	case <-ctx.Done():
		_ = term.Restore(fd, state)
		return nil, ctx.Err()
	}
}

func logoutCommand() *cobra.Command {
	return &cobra.Command{Use: "logout", Short: "Remove saved API key", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, args []string) error {
		if err := config.Logout(); err != nil {
			return err
		}
		if strings.TrimSpace(os.Getenv("MAGIC_HOUR_API_KEY")) != "" {
			fmt.Fprintln(cmd.ErrOrStderr(), "Saved key removed; MAGIC_HOUR_API_KEY remains active.")
		}
		return authResult(cmd, "logged_out", "")
	}}
}

type accountInfo struct {
	ID           string          `json:"id"`
	Email        *string         `json:"email"`
	Tier         string          `json:"tier"`
	Credits      int             `json:"credits"`
	Subscription json.RawMessage `json:"subscription"`
}

func validateKey(ctx context.Context, key string) (accountInfo, error) {
	var account accountInfo
	if err := api.New(key).Do(ctx, http.MethodGet, "/v1/account", nil, &account); err != nil {
		return accountInfo{}, err
	}
	return account, nil
}

func whoamiResult(cmd *cobra.Command, account accountInfo) error {
	format, _ := cmd.Flags().GetString("format")
	if format == "json" {
		return writeJSON(cmd, map[string]any{
			"status": "authenticated", "account_id": account.ID, "email": account.Email,
			"tier": account.Tier, "credits": account.Credits, "subscription": account.Subscription,
		})
	}
	identity := account.ID
	if account.Email != nil && *account.Email != "" {
		identity = *account.Email + " (" + account.ID + ")"
	}
	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
	fmt.Fprintf(w, "Account:\t%s\nTier:\t%s\nCredits:\t%d\n", identity, account.Tier, account.Credits)
	if len(account.Subscription) > 0 {
		var subscription *struct {
			Name   *string `json:"name"`
			Status string  `json:"status"`
		}
		if err := json.Unmarshal(account.Subscription, &subscription); err != nil {
			return fmt.Errorf("decode subscription: %w", err)
		}
		if subscription != nil {
			name := subscription.Status
			if subscription.Name != nil && *subscription.Name != "" {
				name = *subscription.Name + " (" + subscription.Status + ")"
			}
			fmt.Fprintf(w, "Subscription:\t%s\n", name)
		}
	}
	return w.Flush()
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
