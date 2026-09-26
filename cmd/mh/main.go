package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/signal"

	"github.com/magichourhq/magic-hour-cli/internal/api"
	"github.com/magichourhq/magic-hour-cli/internal/cli"
)

var version = "dev"

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	root := cli.New(version)
	if err := root.ExecuteContext(ctx); err != nil {
		format, _ := root.PersistentFlags().GetString("format")
		if format == "json" {
			failure := map[string]any{"message": err.Error()}
			var apiErr *api.Error
			if errors.As(err, &apiErr) {
				failure["code"], failure["status"] = apiErr.Code, apiErr.Status
			}
			_ = json.NewEncoder(os.Stderr).Encode(map[string]any{"error": failure})
		} else {
			fmt.Fprintln(os.Stderr, "Error:", err)
		}
		os.Exit(1)
	}
}
