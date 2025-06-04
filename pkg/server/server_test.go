package server

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/grafana/k6build"
	"github.com/grafana/k6build/pkg/api"
)

type mockBuilder struct {
	err     error
	patform string
	deps    map[string]string
}

func (m mockBuilder) Build(
	ctx context.Context,
	platform string,
	k6Constrains string,
	deps []k6build.Dependency,
) (k6build.Artifact, error) {
	if m.err != nil {
		return k6build.Artifact{}, m.err
	}

	return k6build.Artifact{
		Platform:     m.patform,
		Dependencies: m.deps,
	}, nil
}

func (m mockBuilder) Resolve(
	ctx context.Context,
	k6Constrains string,
	deps []k6build.Dependency,
) (map[string]string, error) {
	if m.err != nil {
		return nil, m.err
	}

	return m.deps, nil
}

func TestAPIServer(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		title    string
		build    k6build.BuildService
		req      []byte
		status   int
		err      error
		artifact k6build.Artifact
	}{
		{
			title: "build ok",
			build: mockBuilder{
				deps: map[string]string{"k6": "v0.1.0"},
			},
			req:      []byte("{\"Platform\": \"linux/amd64\", \"K6Constrains\": \"v0.1.0\", \"Dependencies\": []}"),
			status:   http.StatusOK,
			artifact: k6build.Artifact{},
			err:      nil,
		},
		{
			title: "build error",
			build: mockBuilder{
				err: k6build.ErrBuildFailed,
			},
			req:      []byte("{\"Platform\": \"linux/amd64\", \"K6Constrains\": \"v0.1.0\", \"Dependencies\": []}"),
			status:   http.StatusOK,
			artifact: k6build.Artifact{},
			err:      api.ErrBuildFailed,
		},
		{
			title: "invalid request",
			build: mockBuilder{
				deps: map[string]string{"k6": "v0.1.0"},
			}, req: []byte(""),
			status:   http.StatusBadRequest,
			artifact: k6build.Artifact{},
			err:      api.ErrInvalidRequest,
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

			if tc.err != nil && !errors.Is(buildResponse.Error, tc.err) {
				t.Fatalf("expected error: %q got %q", tc.err, buildResponse.Error)
			}
		})
	}
}
