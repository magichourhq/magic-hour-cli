package cli

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/magichourhq/magic-hour-cli/internal/api"
	"github.com/magichourhq/magic-hour-cli/internal/config"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func estimateCommand(media string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "estimate",
		Short: "Estimate " + media + " generation credits without spending them",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			key, err := config.Key()
			if err != nil {
				return err
			}
			flags := cmd.Flags()
			query := url.Values{}
			cmd.LocalNonPersistentFlags().Visit(func(flag *pflag.Flag) {
				name := strings.ReplaceAll(flag.Name, "-", "_")
				if name == "count" {
					name = "image_count"
				}
				query.Set(name, flag.Value.String())
			})
			var result struct {
				EstimatedCredits int `json:"estimated_credits"`
			}
			client := api.New(key, cmd.Root().Version)
			if err := client.Do(cmd.Context(), http.MethodGet, "/v1/estimate/"+media+"?"+query.Encode(), nil, &result); err != nil {
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
	flags.String("tool", "", media+" tool to estimate (required)")
	switch media {
	case "image":
		flags.String("model", "", "Model for the image generator or editor")
		flags.String("resolution", "", "Resolution for the image generator, editor, or body swap")
		flags.Int("count", 1, "Image count for the image generator or editor")
		flags.Float64("scale-factor", 0, "Upscaler scale factor (2 or 4)")
		flags.String("mode", "", "Upscaler mode: preserve, balanced, or creative")
	case "video":
		flags.Float64("end-seconds", 0, "End time in seconds (required)")
		flags.Float64("start-seconds", 0, "Start time in seconds")
		flags.String("model", "", "Video model")
		flags.String("resolution", "", "Output resolution")
		flags.Bool("audio", false, "Include generated audio")
		flags.Bool("has-video-reference", false, "The request uses a video reference")
		cmd.MarkFlagRequired("end-seconds")
	case "audio":
		flags.Int("character-count", 0, "Prompt character count for voice generation or cloning")
		flags.Int("duration", 0, "Duration in seconds for music generation")
	}
	cmd.MarkFlagRequired("tool")
	return cmd
}
