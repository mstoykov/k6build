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

const (
	defaultAuthType = "Bearer"
)

var (
	ErrBuildFailed   = errors.New("build failed")   //nolint:revive
	ErrRequestFailed = errors.New("request failed") //nolint:revive
)

// BuildServiceClientConfig defines the configuration for accessing a remote build service
type BuildServiceClientConfig struct {
	// URL to build service
	URL string
	// Authorization credentials passed in the Authorization: <type> <credentials> header
	// See AuthorizationType
	Authorization string
	// AuthorizationType type of credentials in the Authorization: <type> <credentials> header
	// For example, "Bearer", "Token", "Basic". Defaults to "Bearer"
	AuthorizationType string
	// Headers custom request headers
	Headers map[string]string
}

// NewBuildServiceClient returns a new client for a remote build service
func NewBuildServiceClient(config BuildServiceClientConfig) (k6build.BuildService, error) {
	return &BuildClient{
		srvURL:   config.URL,
		auth:     config.Authorization,
		authType: config.AuthorizationType,
		headers:  config.Headers,
	}, nil
}

// BuildClient defines a client of a build service
type BuildClient struct {
	srvURL   string
	authType string
	auth     string
	headers  map[string]string
}

// Build request building an artidact to a build service
func (r *BuildClient) Build(
	ctx context.Context,
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

	url, err := url.Parse(r.srvURL)
	if err != nil {
		return k6build.Artifact{}, fmt.Errorf("invalid server %w", err)
	}
	url.Path = "/build/"

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url.String(), marshaled)
	if err != nil {
		return k6build.Artifact{}, fmt.Errorf("%w: %w", ErrRequestFailed, err)
	}
	req.Header.Add("Content-Type", "application/json")

	// add authorization header "Authorization: <type> <auth>"
	if r.auth != "" {
		authType := r.authType
		if authType == "" {
			authType = defaultAuthType
		}
		req.Header.Add("Authorization", fmt.Sprintf("%s %s", authType, r.auth))
	}

	// add custom headers
	for h, v := range r.headers {
		req.Header.Add(h, v)
	}

	resp, err := http.DefaultClient.Do(req)
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
