// Package server implements a build server
package server

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"

	"github.com/grafana/k6build"
	"github.com/grafana/k6build/pkg/api"
)

// APIServerConfig defines the configuration for the APIServer
type APIServerConfig struct {
	BuildService k6build.BuildService
	Log          *slog.Logger
}

// APIServer defines a k6build API server
type APIServer struct {
	srv k6build.BuildService
	log *slog.Logger
}

// NewAPIServer creates a new build service API server
// TODO: add logger
func NewAPIServer(config APIServerConfig) *APIServer {
	log := config.Log
	if log == nil {
		log = slog.New(
			slog.NewTextHandler(
				io.Discard,
				&slog.HandlerOptions{},
			),
		)
	}
	return &APIServer{
		srv: config.BuildService,
		log: log,
	}
}

// ServeHTTP implements the request handler for the build API server
func (a *APIServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	resp := api.BuildResponse{}

	w.Header().Add("Content-Type", "application/json")

	// ensure errors are reported and logged
	defer func() {
		if resp.Error != "" {
			a.log.Error(resp.Error)
			_ = json.NewEncoder(w).Encode(resp) //nolint:errchkjson
		}
	}()

	req := api.BuildRequest{}
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		resp.Error = fmt.Sprintf("invalid request: %s", err.Error())
		return
	}

	a.log.Debug("processing", "request", req.String())

	artifact, err := a.srv.Build( //nolint:contextcheck
		context.Background(),
		req.Platform,
		req.K6Constrains,
		req.Dependencies,
	)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		resp.Error = fmt.Sprintf("building artifact: %s", err.Error())
		return
	}

	a.log.Debug("returning", "artifact", artifact.String())

	resp.Artifact = artifact
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resp) //nolint:errchkjson
}
