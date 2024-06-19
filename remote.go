package k6build

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
)

var (
	ErrBuildFailed   = errors.New("build failed")   //nolint:revive
	ErrRequestFailed = errors.New("request failed") //nolint:revive
)

// RemoteBuildServiceConfig defines the configuration for accessing a remote build service
type RemoteBuildServiceConfig struct {
	URL string
}

// NewRemoteBuildService returns a new client for a remote build service
func NewRemoteBuildService(config RemoteBuildServiceConfig) (BuildService, error) {
	return &remoteSrv{
		srv: config.URL,
	}, nil
}

type remoteSrv struct {
	srv string
}

func (r *remoteSrv) Build(
	ctx context.Context,
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

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		r.srv,
		marshaled,
	)
	if err != nil {
		return Artifact{}, fmt.Errorf("%w: %w", ErrRequestFailed, err)
	}

	resp, err := http.DefaultClient.Do(req)
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
