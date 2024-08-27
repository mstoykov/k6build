package server

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/grafana/k6build"
	"github.com/grafana/k6build/pkg/api"
)

type buildFunction func(
	ctx context.Context,
	platform string,
	k6Constrains string,
	deps []k6build.Dependency,
) (k6build.Artifact, error)

func (f buildFunction) Build(
	ctx context.Context,
	platform string,
	k6Constrains string,
	deps []k6build.Dependency,
) (k6build.Artifact, error) {
	return f(ctx, platform, k6Constrains, deps)
}

//nolint:revive
func buildOk(
	ctx context.Context,
	platform string,
	k6Constrains string,
	deps []k6build.Dependency,
) (k6build.Artifact, error) {
	return k6build.Artifact{
		Dependencies: map[string]string{"k6": "v0.1.0"},
	}, nil
}

//nolint:revive
func buildErr(
	ctx context.Context,
	platform string,
	k6Constrains string,
	deps []k6build.Dependency,
) (k6build.Artifact, error) {
	return k6build.Artifact{}, k6build.ErrBuildFailed
}

func expectedError(expected string, actual string) bool {
	if expected != "" {
		return strings.Contains(actual, expected)
	}

	return actual == ""
}

func TestAPIServer(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		title    string
		build    buildFunction
		req      []byte
		status   int
		err      string
		artifact k6build.Artifact
	}{
		{
			title:    "build ok",
			build:    buildFunction(buildOk),
			req:      []byte("{\"Platform\": \"linux/amd64\", \"K6Constrains\": \"v0.1.0\", \"Dependencies\": []}"),
			status:   http.StatusOK,
			artifact: k6build.Artifact{},
			err:      "",
		},
		{
			title:    "build error",
			build:    buildFunction(buildErr),
			req:      []byte("{\"Platform\": \"linux/amd64\", \"K6Constrains\": \"v0.1.0\", \"Dependencies\": []}"),
			status:   http.StatusBadRequest,
			artifact: k6build.Artifact{},
			err:      k6build.ErrBuildFailed.Error(),
		},
		{
			title:    "invalid request",
			build:    buildFunction(buildOk),
			req:      []byte(""),
			status:   http.StatusBadRequest,
			artifact: k6build.Artifact{},
			err:      "invalid request",
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.title, func(t *testing.T) {
			t.Parallel()

			config := APIServerConfig{
				BuildService: tc.build,
			}
			apiserver := httptest.NewServer(NewAPIServer(config))

			req := bytes.Buffer{}
			req.Write(tc.req)

			resp, err := http.Post(apiserver.URL, "application/json", &req)
			if err != nil {
				t.Fatalf("making request %v", err)
			}
			defer func() {
				_ = resp.Body.Close()
			}()

			if resp.StatusCode != tc.status {
				t.Fatalf("expected status code: %d got %d", tc.status, resp.StatusCode)
			}

			buildResponse := api.BuildResponse{}
			err = json.NewDecoder(resp.Body).Decode(&buildResponse)
			if err != nil {
				t.Fatalf("decoding response %v", err)
			}

			if !expectedError(tc.err, buildResponse.Error) {
				t.Fatalf("expected error: %q got %q", tc.err, buildResponse.Error)
			}
		})
	}
}
