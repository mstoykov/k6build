// Package local implements the local build command
package local

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/grafana/k6build"
	"github.com/grafana/k6build/pkg/local"

	"github.com/spf13/cobra"
)

var ErrTargetPlatformUndefined = errors.New("target platform is required") //nolint:revive

const (
	long = `
k6build local builder returns artifacts that satisfies certain dependencies
`

	example = `
# build k6 v0.50.0 with latest version of k6/x/kubernetes
k6build local -k v0.50.0 -d k6/x/kubernetes

# build k6 v0.51.0 with k6/x/kubernetes v0.8.0 and k6/x/output-kafka v0.7.0
k6build local -k v0.51.0 \
    -d k6/x/kubernetes:v0.8.0 \
    -d k6/x/output-kafka:v0.7.0

# build latest version of k6 with a version of k6/x/kubernetes greater than v0.8.0
k6build local -k v0.50.0 -d 'k6/x/kubernetes:>v0.8.0'

# build k6 v0.50.0 with latest version of k6/x/kubernetes using a custom catalog
k6build local -k v0.50.0 -d k6/x/kubernetes \
    -c /path/to/catalog.json

# build k6 v0.50.0 using a custom GOPROXY
k6build local -k v0.50.0 -e GOPROXY=http://localhost:80
`
)

// New creates new cobra command for local build command.
func New() *cobra.Command {
	var (
		deps     []string
		k6       string
		platform string
		config   local.BuildServiceConfig
	)

	cmd := &cobra.Command{
		Use:     "local",
		Short:   "build using a local builder",
		Long:    long,
		Example: example,
		// prevent the usage help to printed to stderr when an error is reported by a subcommand
		SilenceUsage: true,
		// this is needed to prevent cobra to print errors reported by subcommands in the stderr
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			srv, err := local.NewBuildService(cmd.Context(), config)
			if err != nil {
				return fmt.Errorf("configuring the build service %w", err)
			}

			buildDeps := []k6build.Dependency{}
			for _, d := range deps {
				name, constrains, _ := strings.Cut(d, ":")
				if constrains == "" {
					constrains = "*"
				}
				buildDeps = append(buildDeps, k6build.Dependency{Name: name, Constraints: constrains})
			}

			artifact, err := srv.Build(cmd.Context(), platform, k6, buildDeps)
			if err != nil {
				return fmt.Errorf("building %w", err)
			}

			encoder := json.NewEncoder(os.Stdout)
			encoder.SetIndent("", "  ")
			err = encoder.Encode(artifact)
			if err != nil {
				return fmt.Errorf("processing object %w", err)
			}

			return nil
		},
	}

	cmd.Flags().StringArrayVarP(&deps, "dependency", "d", nil, "list of dependencies in form package:constrains")
	cmd.Flags().StringVarP(&k6, "k6", "k", "*", "k6 version constrains")
	cmd.Flags().StringVarP(&platform, "platform", "p", "", "target platform (default GOOS/GOARCH)")
	_ = cmd.MarkFlagRequired("platform")
	cmd.Flags().StringVarP(&config.Catalog, "catalog", "c", "catalog.json", "dependencies catalog")
	cmd.Flags().StringVarP(&config.CacheDir, "cache-dir", "f", "/tmp/buildservice", "cache dir")
	cmd.Flags().BoolVarP(&config.Verbose, "verbose", "v", false, "print build process output")
	cmd.Flags().BoolVarP(&config.CopyGoEnv, "copy-go-env", "g", true, "copy go environment")
	cmd.Flags().StringToStringVarP(&config.BuildEnv, "env", "e", nil, "build environment variables")

	return cmd
}
