// Package cmd contains build cobra command factory function.
package cmd

import (
	"github.com/spf13/cobra"

	"github.com/grafana/k6build/cmd/cache"
	"github.com/grafana/k6build/cmd/local"
	"github.com/grafana/k6build/cmd/remote"
	"github.com/grafana/k6build/cmd/server"
)

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

	root.AddCommand(cache.New())
	root.AddCommand(remote.New())
	root.AddCommand(local.New())
	root.AddCommand(server.New())

	return root
}
