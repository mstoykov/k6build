// Package server implements the build server command
package server

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"github.com/grafana/k6build"
	"github.com/grafana/k6build/pkg/local"
	server "github.com/grafana/k6build/pkg/server"

	"github.com/spf13/cobra"
)

const (
	long = `
starts a k6build server
`

	example = `
# start the build server using a custom catalog
k6build server -c /path/to/catalog.json

# start the server the build server using a custom GOPROXY
k6build server -e GOPROXY=http://localhost:80`
)

// New creates new cobra command for the server command.
func New() *cobra.Command {
	var (
		config   local.BuildServiceConfig
		logLevel string
		port     int
	)

	cmd := &cobra.Command{
		Use:     "server",
		Short:   "k6 build service",
		Long:    long,
		Example: example,
		// prevent the usage help to printed to stderr when an error is reported by a subcommand
		SilenceUsage: true,
		// this is needed to prevent cobra to print errors reported by subcommands in the stderr
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			// set log
			ll, err := k6build.ParseLogLevel(logLevel)
			if err != nil {
				return fmt.Errorf("parsing log level %w", err)
			}

			log := slog.New(
				slog.NewTextHandler(
					os.Stderr,
					&slog.HandlerOptions{
						Level: ll,
					},
				),
			)

			buildSrv, err := local.NewBuildService(cmd.Context(), config)
			if err != nil {
				return fmt.Errorf("creating local build service  %w", err)
			}

			apiConfig := server.APIServerConfig{
				BuildService: buildSrv,
				Log:          log,
			}
			buildAPI := server.NewAPIServer(apiConfig)

			srv := http.NewServeMux()
			srv.Handle("POST /build/", http.StripPrefix("/build", buildAPI))

			listerAddr := fmt.Sprintf("0.0.0.0:%d", port)
			log.Info("starting server", "address", listerAddr)
			err = http.ListenAndServe(listerAddr, srv) //nolint:gosec
			if err != nil {
				log.Info("server ended", "error", err.Error())
			}
			log.Info("ending server")

			return nil
		},
	}

	cmd.Flags().StringVarP(&config.Catalog, "catalog", "c", "catalog.json", "dependencies catalog")
	cmd.Flags().StringVar(&config.CacheURL, "cache-url", "http://localhost:9000", "cache server url")
	cmd.Flags().BoolVarP(&config.Verbose, "verbose", "v", false, "print build process output")
	cmd.Flags().BoolVarP(&config.CopyGoEnv, "copy-go-env", "g", true, "copy go environment")
	cmd.Flags().StringToStringVarP(&config.BuildEnv, "env", "e", nil, "build environment variables")
	cmd.Flags().IntVarP(&port, "port", "p", 8000, "port server will listen")
	cmd.Flags().StringVarP(&logLevel, "log-level", "l", "INFO", "log level")

	return cmd
}
