package k6build

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

// BuildServiceClientConfig defines the configuration for accessing a remote build service
type BuildServiceClientConfig struct {
	URL string
}

// NewBuildServiceClient returns a new client for a remote build service
func NewBuildServiceClient(config BuildServiceClientConfig) (BuildService, error) {
	return &BuildClient{
		srv: config.URL,
	}, nil
}

// BuildClient defines a client of a build service
type BuildClient struct {
	srv string
}

// Build request building an artidact to a build service
func (r *BuildClient) Build(
	_ context.Context,
	platform string,
	k6Constrains string,
	deps []Dependency,
) (Artifact, error) {
	buildRequest := BuildRequest{
		Platform:     platform,
		K6Constrains: k6Constrains,
		Dependencies: deps,
	}
	marshaled := &bytes.Buffer{}
	err := json.NewEncoder(marshaled).Encode(buildRequest)
	if err != nil {
		return Artifact{}, fmt.Errorf("%w: %w", ErrRequestFailed, err)
	}

	url, err := url.Parse(r.srv)
	if err != nil {
		return Artifact{}, fmt.Errorf("invalid server %w", err)
	}
	url.Path = "/build/"
	resp, err := http.Post(url.String(), "application/json", marshaled) //nolint:noctx
	if err != nil {
		return Artifact{}, fmt.Errorf("%w: %w", ErrRequestFailed, err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	buildResponse := BuildResponse{}
	err = json.NewDecoder(resp.Body).Decode(&buildResponse)
	if err != nil {
		return Artifact{}, fmt.Errorf("%w: %w", ErrRequestFailed, err)
	}

	if resp.StatusCode != http.StatusOK {
		return Artifact{}, fmt.Errorf("%w: %s", ErrRequestFailed, buildResponse.Error)
	}

	if buildResponse.Error != "" {
		return Artifact{}, fmt.Errorf("%w: %s", ErrBuildFailed, buildResponse.Error)
	}

	return buildResponse.Artifact, nil
}
