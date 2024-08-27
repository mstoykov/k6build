// Package client implements a client for a build service
package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"

	"github.com/grafana/k6build"
	"github.com/grafana/k6build/pkg/api"
)

var (
	ErrBuildFailed   = errors.New("build failed")   //nolint:revive
	ErrRequestFailed = errors.New("request failed") //nolint:revive
)

// BuildServiceClientConfig defines the configuration for accessing a remote build service
type BuildServiceClientConfig struct {
	URL string
}

// NewBuildServiceClient returns a new client for a remote build service
func NewBuildServiceClient(config BuildServiceClientConfig) (k6build.BuildService, error) {
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
	deps []k6build.Dependency,
) (k6build.Artifact, error) {
	buildRequest := api.BuildRequest{
		Platform:     platform,
		K6Constrains: k6Constrains,
		Dependencies: deps,
	}
	marshaled := &bytes.Buffer{}
	err := json.NewEncoder(marshaled).Encode(buildRequest)
	if err != nil {
		return k6build.Artifact{}, fmt.Errorf("%w: %w", ErrRequestFailed, err)
	}

	url, err := url.Parse(r.srv)
	if err != nil {
		return k6build.Artifact{}, fmt.Errorf("invalid server %w", err)
	}
	url.Path = "/build/"
	resp, err := http.Post(url.String(), "application/json", marshaled) //nolint:noctx
	if err != nil {
		return k6build.Artifact{}, fmt.Errorf("%w: %w", ErrRequestFailed, err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	buildResponse := api.BuildResponse{}
	err = json.NewDecoder(resp.Body).Decode(&buildResponse)
	if err != nil {
		return k6build.Artifact{}, fmt.Errorf("%w: %w", ErrRequestFailed, err)
	}

	if resp.StatusCode != http.StatusOK {
		return k6build.Artifact{}, fmt.Errorf("%w: %s", ErrRequestFailed, buildResponse.Error)
	}

	if buildResponse.Error != "" {
		return k6build.Artifact{}, fmt.Errorf("%w: %s", ErrBuildFailed, buildResponse.Error)
	}

	return buildResponse.Artifact, nil
}
