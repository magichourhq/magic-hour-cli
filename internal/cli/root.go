package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/magichourhq/magic-hour-cli/internal/catalog"
	"github.com/magichourhq/magic-hour-cli/internal/workflow"
	"github.com/spf13/cobra"
)

func New(version string) *cobra.Command {
	root := &cobra.Command{Use: "mh", Short: "Generate and edit images with Magic Hour", Version: version, SilenceUsage: true, SilenceErrors: true}
	var format string
	root.PersistentFlags().StringVar(&format, "format", "text", "Result format: text or json")
	root.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		if format != "text" && format != "json" {
			return fmt.Errorf("--format must be text or json")
		}
		return nil
	}
	root.RegisterFlagCompletionFunc("format", choices([]string{"text", "json"}))
	addAuth(root)
	groups := map[string]*cobra.Command{}
	for _, op := range catalog.Operations {
		group := groups[op.Group]
		if group == nil {
			group = &cobra.Command{Use: op.Group, Short: "Create and manage " + op.Kind + " projects"}
			groups[op.Group] = group
			root.AddCommand(group)
			addManagement(group, op.Kind)
		}
		group.AddCommand(operationCommand(op))
	}
	root.AddCommand(&cobra.Command{Use: "schema [group command]", Short: "Print generated command definitions as JSON", Args: func(cmd *cobra.Command, args []string) error {
		if len(args) != 0 && len(args) != 2 {
			return fmt.Errorf("usage: mh schema [group command]")
		}
		return nil
	}, RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return writeJSON(cmd, catalog.Operations)
		}
		for _, op := range catalog.Operations {
			if op.Group == args[0] && op.Name == args[1] {
				return writeJSON(cmd, op)
			}
		}
		return fmt.Errorf("unknown command %s", strings.Join(args, " "))
	}})
	root.AddCommand(&cobra.Command{Use: "version", Short: "Print CLI version", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, args []string) error {
		_, err := fmt.Fprintln(cmd.OutOrStdout(), version)
		return err
	}})
	return root
}

func operationCommand(op catalog.Operation) *cobra.Command {
	cmd := &cobra.Command{Use: op.Name, Short: op.Summary, Example: op.Example, Args: cobra.NoArgs}
	opts, timeout := executionFlags(cmd)
	var dryRun bool
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Validate inputs and print the request without making API calls")
	for _, f := range op.Fields {
		help := strings.ReplaceAll(f.Help, "`", "'")
		if len(f.Enum) > 0 {
			help = strings.TrimSuffix(help, ".")
		}
		if f.Required && f.Default == "" {
			help += " (required)"
		}
		if len(f.Enum) > 0 {
			if len(f.Enum) <= 10 {
				help += "; choices: " + strings.Join(f.Enum, ", ")
			} else {
				help += fmt.Sprintf("; %d choices (use completion or mh schema %s %s)", len(f.Enum), op.Group, op.Name)
			}
		}
		if f.Type == "array" {
			cmd.Flags().StringArray(f.Flag, nil, help)
		} else {
			cmd.Flags().String(f.Flag, f.Default, help)
		}
		if len(f.Enum) > 0 {
			cmd.RegisterFlagCompletionFunc(f.Flag, choices(f.Enum))
		} else if f.FileKind == "" {
			cmd.RegisterFlagCompletionFunc(f.Flag, choices(nil))
		}
	}
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		if *timeout <= 0 {
			return fmt.Errorf("--timeout must be positive")
		}
		readCtx, cancel := context.WithTimeout(cmd.Context(), *timeout)
		defer cancel()
		values := map[string][]string{}
		for _, f := range op.Fields {
			if !cmd.Flags().Changed(f.Flag) {
				continue
			}
			if f.Type == "array" {
				values[f.Flag], _ = cmd.Flags().GetStringArray(f.Flag)
			} else {
				s, _ := cmd.Flags().GetString(f.Flag)
				values[f.Flag] = []string{s}
			}
		}
		if err := resolveStdin(readCtx, cmd.InOrStdin(), op, values); err != nil {
			return err
		}
		body, err := op.Body(values)
		if err != nil {
			return err
		}
		if dryRun {
			return writeJSON(cmd, map[string]any{"method": "POST", "path": op.Path, "body": body})
		}
		return execute(cmd, *timeout, func(ctx context.Context, runner workflow.Runner) (workflow.Result, error) {
			return runner.Generate(ctx, op, values, *opts)
		})
	}
	return cmd
}

func choices(values []string) func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
	return func(_ *cobra.Command, _ []string, prefix string) ([]string, cobra.ShellCompDirective) {
		var matches []string
		for _, value := range values {
			if strings.HasPrefix(value, prefix) {
				matches = append(matches, value)
			}
		}
		return matches, cobra.ShellCompDirectiveNoFileComp
	}
}

func writeJSON(cmd *cobra.Command, value any) error {
	enc := json.NewEncoder(cmd.OutOrStdout())
	enc.SetIndent("", "  ")
	return enc.Encode(value)
}
