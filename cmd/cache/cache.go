// Package cache implements the cache command
package cache

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"github.com/grafana/k6build"
	"github.com/grafana/k6build/pkg/cache/file"
	"github.com/grafana/k6build/pkg/cache/server"

	"github.com/spf13/cobra"
)

const (
	long = `
Starts a k6build cache server. 

The cache server offers a REST API for storing and downloading objects.

Objects can be retrieved by a download url returned when the object is stored.

The --download-url specifies the base URL for downloading objects. This is necessary to allow
downloading the objects from different machines.
`

	example = `
# start the cache server serving an external url
k6build cache --download-url http://external.url

# store object from same host
curl -x POST http://localhost:9000/cache/objectID -d "object content" | jq .
{
	"Error": "",
	"Object": {
	  "ID": "objectID",
	  "Checksum": "17d3eb873fe4b1aac4f9d2505aefbb5b53b9a7f34a6aadd561be104c0e9d678b",
	  "URL": "http://external.url:9000/cache/objectID/download"
	}
      }

# download object from another machine using the external url
curl http://external.url:9000/cache/objectID/download
`
)

// New creates new cobra command for cache command.
func New() *cobra.Command {
	var (
		cacheDir    string
		cacheSrvURL string
		port        int
		logLevel    string
	)

	cmd := &cobra.Command{
		Use:     "cache",
		Short:   "k6build cache server",
		Long:    long,
		Example: example,
		// prevent the usage help to printed to stderr when an error is reported by a subcommand
		SilenceUsage: true,
		// this is needed to prevent cobra to print errors reported by subcommands in the stderr
		SilenceErrors: true,
		RunE: func(_ *cobra.Command, _ []string) error {
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

			cache, err := file.NewFileCache(cacheDir)
			if err != nil {
				return fmt.Errorf("creating cache %w", err)
			}

			// FIXME: this will not work across machines
			if cacheSrvURL == "" {
				cacheSrvURL = fmt.Sprintf("http://localhost:%d/cache", port)
			}
			config := server.CacheServerConfig{
				BaseURL: cacheSrvURL,
				Cache:   cache,
				Log:     log,
			}
			cacheSrv := server.NewCacheServer(config)

			srv := http.NewServeMux()
			srv.Handle("/cache/", http.StripPrefix("/cache", cacheSrv))

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

	cmd.Flags().StringVarP(&cacheDir, "cache-dir", "c", "/tmp/cache/objectstore", "cache directory")
	cmd.Flags().IntVarP(&port, "port", "p", 9000, "port server will listen")
	cmd.Flags().StringVarP(&cacheSrvURL,
		"download-url", "d", "", "base url used for downloading objects."+
			"\nIf not specified http://localhost:<port>/cache is used",
	)
	cmd.Flags().StringVarP(&logLevel, "log-level", "l", "INFO", "log level")

	return cmd
}
