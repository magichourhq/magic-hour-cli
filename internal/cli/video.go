package cli

import (
	"fmt"
	"slices"
	"strings"

	"github.com/magichourhq/magic-hour-cli/internal/catalog"
	"github.com/spf13/cobra"
)

// The OpenAPI operations stay separate; --image picks the endpoint at runtime.
func videoGenerateCommand(textOp, imageOp catalog.Operation) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "generate",
		Short: "Generate video from text or an image",
		Example: "mh video generate --prompt 'A corgi running through a field' --aspect-ratio 16:9\n" +
			"mh video generate --image photo.png --prompt 'Gentle camera movement'",
		Args: cobra.NoArgs,
	}
	fields := append([]catalog.Field{}, textOp.Fields...)
	seen := map[string]int{}
	for i := range fields {
		fields[i].Required = false // Required fields depend on selected endpoint.
		seen[fields[i].Flag] = i
	}
	for _, f := range imageOp.Fields {
		if i, ok := seen[f.Flag]; ok {
			for _, value := range f.Enum {
				if !slices.Contains(fields[i].Enum, value) {
					fields[i].Enum = append(fields[i].Enum, value)
				}
			}
			continue
		}
		f.Required = false
		fields = append(fields, f)
	}
	configureOperation(cmd, fields, func(cmd *cobra.Command) catalog.Operation {
		if cmd.Flags().Changed("image") {
			return imageOp
		}
		return textOp
	})
	run := cmd.RunE
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("image") && cmd.Flags().Changed("aspect-ratio") {
			return fmt.Errorf("--aspect-ratio is text-to-video only; omit it when using --image")
		}
		if !cmd.Flags().Changed("image") && cmd.Flags().Changed("end-image") {
			return fmt.Errorf("--end-image requires --image")
		}
		return run(cmd, args)
	}
	cmd.RegisterFlagCompletionFunc("model", func(cmd *cobra.Command, _ []string, prefix string) ([]string, cobra.ShellCompDirective) {
		textModels, imageModels := fieldChoices(textOp, "model"), fieldChoices(imageOp, "model")
		if cmd.Flags().Changed("image") {
			return choices(imageModels)(cmd, nil, prefix)
		}
		var matches []string
		for _, model := range append(slices.Clone(textModels), imageModels...) {
			if !strings.HasPrefix(model, prefix) || slices.Contains(matches, model) {
				continue
			}
			if !slices.Contains(textModels, model) {
				matches = append(matches, cobra.CompletionWithDesc(model, "image only"))
			} else if !slices.Contains(imageModels, model) {
				matches = append(matches, cobra.CompletionWithDesc(model, "text only"))
			} else {
				matches = append(matches, model)
			}
		}
		return matches, cobra.ShellCompDirectiveNoFileComp
	})
	return cmd
}

func fieldChoices(op catalog.Operation, flag string) []string {
	for _, f := range op.Fields {
		if f.Flag == flag {
			return f.Enum
		}
	}
	return nil
}
