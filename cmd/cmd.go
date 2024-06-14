// Package cmd contains build cobra command factory function.
package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/grafana/k6build"
	"github.com/grafana/k6catalog"
	"github.com/grafana/k6foundry"
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
	var (
		deps         []string
		k6Constrains string
		platform     string
		registry     string
		verbose      bool
	)

	cmd := &cobra.Command{
		Use:   "k6build",
		Short: "k6 build service",
		Long:  long,
		// prevent the usage help to printed to stderr when an error is reported by a subcommand
		SilenceUsage: true,
		// this is needed to prevent cobra to print errors reported by subcommands in the stderr
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			catalog, err := k6catalog.NewCatalogFromJSON(registry)
			if err != nil {
				return fmt.Errorf("loading catalog %w", err)
			}

			opts := k6foundry.NativeBuilderOpts{}
			// This is required to pass environment variables like the github access credential
			opts.CopyEnv = true

			if verbose {
				opts.Verbose = true
				opts.Stdout = os.Stdout
				opts.Stderr = os.Stderr
				opts.LogLevel = "DEBUG"
			}
			builder, err := k6foundry.NewNativeBuilder(context.TODO(), opts)
			if err != nil {
				return fmt.Errorf("setting up builder %w", err)
			}

			cache, err := k6build.NewTempFileCache()
			if err != nil {
				return fmt.Errorf("creating build cache %w", err)
			}

			srv := k6build.NewBuildService(catalog, builder, cache)

			buildDeps := []k6build.Dependency{}
			for _, d := range deps {
				name, constrains, _ := strings.Cut(d, ":")
				if constrains == "" {
					constrains = "*"
				}
				buildDeps = append(buildDeps, k6build.Dependency{Name: name, Constraints: constrains})
			}

			artifact, err := srv.Build(context.TODO(), platform, k6Constrains, buildDeps)
			if err != nil {
				return fmt.Errorf("building %w", err)
			}

			encoder := json.NewEncoder(os.Stdout)
			encoder.SetIndent("", "  ")
			encoder.Encode(artifact)

			return nil
		},
	}

	cmd.Flags().StringArrayVarP(&deps, "dependency", "d", nil, "list of dependencies in form package:constrains")
	cmd.Flags().StringVarP(&k6Constrains, "k6-constrains", "k", "*", "k6 version constrains")
	cmd.Flags().StringVarP(&platform, "platform", "p", "", "target platform (default GOOS/GOARCH)")
	cmd.MarkFlagRequired("platform")
	cmd.Flags().StringVarP(&registry, "catalog", "c", "catalog.json", "dependencies catalog")
	cmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "print build process output")

	return cmd
}
