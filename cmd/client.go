package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/grafana/k6build"
	"github.com/spf13/cobra"
)

const (
	clientLong = `
k6build client connects to a remote build server
`

	clientExamples = `
# build k6 v0.50.0 with latest version of k6/x/kubernetes
k6build client -s http://localhost:8000 -k v0.50.0 -d k6/x/kubernetes

# build k6 v0.51.0 with k6/x/kubernetes v0.8.0 and k6/x/output-kafka v0.7.0
k6build client -s http://localhost:8000 \
     -k v0.51.0 \
    -d k6/x/kubernetes:v0.8.0 \
    -d k6/x/output-kafka:v0.7.0

# build latest version of k6 with a version of k6/x/kubernetes greater than v0.8.0
k6build client -s http://localhost:8000 \
    -k v0.50.0 -d 'k6/x/kubernetes:>v0.8.0'
`
)

// NewClient creates new cobra command for build client command.
func NewClient() *cobra.Command {
	var (
		deps     []string
		k6       string
		platform string
		config   k6build.BuildServiceClientConfig
	)

	cmd := &cobra.Command{
		Use:     "client",
		Short:   "build k6 using a remote build server",
		Long:    clientLong,
		Example: clientExamples,
		// prevent the usage help to printed to stderr when an error is reported by a subcommand
		SilenceUsage: true,
		// this is needed to prevent cobra to print errors reported by subcommands in the stderr
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, err := k6build.NewBuildServiceClient(config)
			if err != nil {
				return fmt.Errorf("configuring the client %w", err)
			}

			buildDeps := []k6build.Dependency{}
			for _, d := range deps {
				name, constrains, _ := strings.Cut(d, ":")
				if constrains == "" {
					constrains = "*"
				}
				buildDeps = append(buildDeps, k6build.Dependency{Name: name, Constraints: constrains})
			}

			artifact, err := client.Build(cmd.Context(), platform, k6, buildDeps)
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

	cmd.Flags().StringVarP(&config.URL, "server", "s", "http://localhost:8000", "url for build server")
	cmd.Flags().StringArrayVarP(&deps, "dependency", "d", nil, "list of dependencies in form package:constrains")
	cmd.Flags().StringVarP(&k6, "k6", "k", "*", "k6 version constrains")
	cmd.Flags().StringVarP(&platform, "platform", "p", "", "target platform (default GOOS/GOARCH)")
	_ = cmd.MarkFlagRequired("platform")

	return cmd
}
