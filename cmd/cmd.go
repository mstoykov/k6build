// Package cmd offers commands for interacting with the build service
package cmd

import "github.com/spf13/cobra"

// New creates a new root command for k6build
func New() *cobra.Command {
	root := &cobra.Command{}
	root.AddCommand(NewLocal())

	return root
}
