package k6build

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
)

// BuildRequest defines a request to the build service
type BuildRequest struct {
	K6Constrains string       `json:"k6:omitempty"`
	Dependencies []Dependency `json:"dependencies,omitempty"`
	Platform     string       `json:"platformomitempty"`
}

// String returns a text serialization of the BuildRequest
func (r BuildRequest) String() string {
	buffer := &bytes.Buffer{}
	buffer.WriteString(fmt.Sprintf("platform: %s", r.Platform))
	buffer.WriteString(fmt.Sprintf("k6: %s", r.K6Constrains))
	for _, d := range r.Dependencies {
		buffer.WriteString(fmt.Sprintf("%s:%q", d.Name, d.Constraints))
	}
	return buffer.String()
}

// BuildResponse defines the response for a BuildRequest
type BuildResponse struct {
	Error    string   `json:"error:omitempty"`
	Artifact Artifact `json:"artifact:omitempty"`
}

// APIServerConfig defines the configuration for the APIServer
type APIServerConfig struct {
	BuildService BuildService
	Log          *slog.Logger
}

// APIServer defines a k6build API server
type APIServer struct {
	srv BuildService
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
	resp := BuildResponse{}

	w.Header().Add("Content-Type", "application/json")

	// ensure errors are reported and logged
	defer func() {
		if resp.Error != "" {
			a.log.Error(resp.Error)
			_ = json.NewEncoder(w).Encode(resp) //nolint:errchkjson
		}
	}()

	req := BuildRequest{}
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
