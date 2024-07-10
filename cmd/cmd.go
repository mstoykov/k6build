// Package cmd offers commands for interacting with the build service
package cmd

import "github.com/spf13/cobra"

// New creates a new root command for k6build
func New() *cobra.Command {
	root := &cobra.Command{
		Use:               "k6build",
		Short:             "Build k6 with various builders.",
		Long:              "Build k6 using one of the supported builders.",
		SilenceUsage:      true,
		SilenceErrors:     true,
		DisableAutoGenTag: true,
		CompletionOptions: cobra.CompletionOptions{DisableDefaultCmd: true},
	}

	root.AddCommand(NewLocal())
	root.AddCommand(NewServer())
	root.AddCommand(NewClient())

	return root
}
