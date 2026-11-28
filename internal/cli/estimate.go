package cli

import (
	"fmt"
	"net/http"
	"net/url"

	"github.com/magichourhq/magic-hour-cli/internal/api"
	"github.com/magichourhq/magic-hour-cli/internal/config"
	"github.com/spf13/cobra"
)

func imageEstimateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "estimate",
		Short: "Estimate image generation credits without spending them",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			key, err := config.Key()
			if err != nil {
				return err
			}
			flags := cmd.Flags()
			query := url.Values{}
			for _, field := range []struct{ flag, param string }{
				{"tool", "tool"}, {"model", "model"}, {"resolution", "resolution"},
				{"count", "image_count"}, {"scale-factor", "scale_factor"}, {"mode", "mode"},
			} {
				if flags.Changed(field.flag) {
					query.Set(field.param, flags.Lookup(field.flag).Value.String())
				}
			}
			var result struct {
				EstimatedCredits int `json:"estimated_credits"`
			}
			client := api.New(key, cmd.Root().Version)
			if err := client.Do(cmd.Context(), http.MethodGet, "/v1/estimate/image?"+query.Encode(), nil, &result); err != nil {
				return err
			}
			if format, _ := flags.GetString("format"); format == "json" {
				return writeJSON(cmd, result)
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "Estimated credits: %d\n", result.EstimatedCredits)
			return err
		},
	}
	flags := cmd.Flags()
	flags.String("tool", "", "Image endpoint to estimate, such as ai-image-generator (required)")
	flags.String("model", "", "Model for the image generator or editor")
	flags.String("resolution", "", "Resolution for the image generator, editor, or body swap")
	flags.Int("count", 1, "Image count for the image generator or editor")
	flags.Float64("scale-factor", 0, "Upscaler scale factor (2 or 4)")
	flags.String("mode", "", "Upscaler mode: preserve, balanced, or creative")
	cmd.MarkFlagRequired("tool")
	return cmd
}
