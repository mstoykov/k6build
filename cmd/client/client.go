// Package client implements the client command
package client

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/grafana/k6build"
	"github.com/grafana/k6build/pkg/client"

	"github.com/spf13/cobra"
)

const (
	long = `
k6build client connects to a remote build server
`

	example = `
# build k6 v0.51.0 with k6/x/kubernetes v0.8.0 and k6/x/output-kafka v0.7.0
k6build client -s http://localhost:8000 \
    -k v0.51.0 \
    -p linux/amd64 
    -d k6/x/kubernetes:v0.8.0 \
    -d k6/x/output-kafka:v0.7.0

{
    "id": "62d08b13fdef171435e2c6874eaad0bb35f2f9c7",
    "url": "http://localhost:8000/cache/62d08b13fdef171435e2c6874eaad0bb35f2f9c7/download",
    "dependencies": {
	"k6": "v0.51.0",
	"k6/x/kubernetes": "v0.9.0",
	"k6/x/output-kafka": "v0.7.0"
    },
    "platform": "linux/amd64",
    "checksum": "f4af178bb2e29862c0fc7d481076c9ba4468572903480fe9d6c999fea75f3793"
}

# build k6 v0.51 with k6/x/output-kafka v0.7.0 and download to 'build/k6'
k6build client -s http://localhost:8000
    -p linux/amd64 
    -k v0.51.0 -d k6/x/output-kafka:v0.7.0
    -o build/k6 -q

# check binary
build/k6 version
k6 v0.51.0 (go1.22.2, linux/amd64)
Extensions:
  github.com/grafana/xk6-output-kafka v0.7.0, xk6-kafka [output]



# build latest version of k6 with a version of k6/x/kubernetes greater than v0.8.0
k6build client -s http://localhost:8000 \
    -p linux/amd64 \
    -k v0.50.0 -d 'k6/x/kubernetes:>v0.8.0'
{
   "id": "18035a12975b262430b55988ffe053098d020034",
   "url": "http://localhost:8000/cache/18035a12975b262430b55988ffe053098d020034/download",
   "dependencies": {
       "k6": "v0.50.0",
	"k6/x/kubernetes": "v0.9.0"
    },
   "platform": "linux/amd64",
   "checksum": "255e5d62852af5e4109a0ac6f5818936a91c986919d12d8437e97fb96919847b"
}
`
)

// New creates new cobra command for build client command.
func New() *cobra.Command { //nolint:funlen
	var (
		config   client.BuildServiceClientConfig
		deps     []string
		k6       string
		output   string
		platform string
		quiet    bool
	)

	cmd := &cobra.Command{
		Use:     "client",
		Short:   "build k6 using a remote build server",
		Long:    long,
		Example: example,
		// prevent the usage help to printed to stderr when an error is reported by a subcommand
		SilenceUsage: true,
		// this is needed to prevent cobra to print errors reported by subcommands in the stderr
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, err := client.NewBuildServiceClient(config)
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

			if !quiet {
				encoder := json.NewEncoder(os.Stdout)
				encoder.SetIndent("", "  ")
				err = encoder.Encode(artifact)
				if err != nil {
					return fmt.Errorf("processing response %w", err)
				}
			}

			if output != "" {
				resp, err := http.Get(artifact.URL) //nolint:noctx
				if err != nil {
					return fmt.Errorf("downloading artifact %w", err)
				}

				if resp.StatusCode != http.StatusOK {
					return fmt.Errorf("request failed with status %s", resp.Status)
				}

				outFile, err := os.OpenFile(output, os.O_WRONLY|os.O_CREATE, 0o755) //nolint:gosec
				if err != nil {
					return fmt.Errorf("opening output file %w", err)
				}
				defer func() {
					_ = resp.Body.Close()
				}()

				_, err = io.Copy(outFile, resp.Body)
				if err != nil {
					return fmt.Errorf("copying artifact %w", err)
				}
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&config.URL, "server", "s", "http://localhost:8000", "url for build server")
	cmd.Flags().StringArrayVarP(&deps, "dependency", "d", nil, "list of dependencies in form package:constrains")
	cmd.Flags().StringVarP(&k6, "k6", "k", "*", "k6 version constrains")
	cmd.Flags().StringVarP(&platform, "platform", "p", "", "target platform (default GOOS/GOARCH)")
	cmd.Flags().StringVarP(&output, "output", "o", "", "path to download the artifact as an executable."+
		" If not specified, the artifact is not downloaded.")
	cmd.Flags().BoolVarP(&quiet, "quiet", "q", false, "don't print artifact's details")

	return cmd
}
