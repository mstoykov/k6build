package k6build

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// BuildRequest defines a request to the build service
type BuildRequest struct {
	K6Constrains string       `json:"k6:omitempty"`
	Dependencies []Dependency `json:"dependencies,omitempty"`
	Platform     string       `json:"platformomitempty"`
}

// BuildResponse defines the response for a BuildRequest
type BuildResponse struct {
	Error    string   `json:"error:omitempty"`
	Artifact Artifact `json:"artifact:omitempty"`
}

// APIServer defines a k6build API server
type APIServer struct {
	srv BuildService
}

// NewAPIServer creates a new build service API server
// TODO: add logger
func NewAPIServer(srv BuildService) *APIServer {
	return &APIServer{
		srv: srv,
	}
}

// ServeHTTP implements the request handler for the build API server
func (a *APIServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	resp := BuildResponse{}

	w.Header().Add("Content-Type", "application/json")

	// ensure errors are reported
	defer func() {
		if resp.Error != "" {
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

	resp.Artifact = artifact
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resp) //nolint:errchkjson
}
