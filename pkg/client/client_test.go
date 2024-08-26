package client

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

type testSrv struct {
	status   int
	response api.BuildResponse
}

func (t testSrv) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "application/json")

	// validate request
	req := api.BuildRequest{}
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// send canned response
	resp := &bytes.Buffer{}
	err = json.NewEncoder(resp).Encode(t.response)
	if err != nil {
		// set uncommon status code to signal something unexpected happened
		w.WriteHeader(http.StatusTeapot)
		return
	}

	w.WriteHeader(t.status)
	_, _ = w.Write(resp.Bytes())
}

func TestRemote(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		title     string
		status    int
		resp      api.BuildResponse
		expectErr error
	}{
		{
			title:  "normal build",
			status: http.StatusOK,
			resp: api.BuildResponse{
				Error:    "",
				Artifact: k6build.Artifact{},
			},
		},
		{
			title:  "request failed",
			status: http.StatusInternalServerError,
			resp: api.BuildResponse{
				Error:    "request failed",
				Artifact: k6build.Artifact{},
			},
			expectErr: ErrRequestFailed,
		},
		{
			title:  "failed build",
			status: http.StatusOK,
			resp: api.BuildResponse{
				Error:    "failed build",
				Artifact: k6build.Artifact{},
			},
			expectErr: ErrBuildFailed,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.title, func(t *testing.T) {
			t.Parallel()

			srv := httptest.NewServer(testSrv{
				status:   tc.status,
				response: tc.resp,
			})

			client, err := NewBuildServiceClient(
				BuildServiceClientConfig{
					URL: srv.URL,
				},
			)
			if err != nil {
				t.Fatalf("unexpected %v", err)
			}

			_, err = client.Build(
				context.TODO(),
				"linux/amd64",
				"v0.1.0",
				[]k6build.Dependency{{Name: "k6/x/test", Constraints: "*"}},
			)

			if !errors.Is(err, tc.expectErr) {
				t.Fatalf("expected %v got %v", tc.expectErr, err)
			}
		})
	}
}
