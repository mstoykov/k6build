// Package cmd contains build cobra command factory function.
package cmd

import (
	"errors"

	"github.com/spf13/cobra"
)

var ErrTargetPlatformUndefined = errors.New("target platform is required") //nolint:revive

const (
	long = `
k6 build service returns artifacts that satisfies certain dependencies
`
)

// New creates new cobra command for resolve command.
func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "k6build",
		Short: "k6 build service",
		Long:  long,
		// prevent the usage help to printed to stderr when an error is reported by a subcommand
		SilenceUsage: true,
		// this is needed to prevent cobra to print errors reported by subcommands in the stderr
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return nil
		},
	}

	return cmd
}
