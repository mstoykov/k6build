package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/grafana/k6build"
	"github.com/grafana/k6build/pkg/api"
)

type testSrv struct {
	status   int
	response api.BuildResponse
	auth     string
	authType string
	headers  map[string]string
}

func (t testSrv) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "application/json")

	// validate authorization
	if t.auth != "" {
		authHeader := fmt.Sprintf("%s %s", t.authType, t.auth)
		if r.Header.Get("Authorization") != authHeader {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
	}

	// validate headers
	for h, v := range t.headers {
		if r.Header.Get(h) != v {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
	}

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
		headers   map[string]string
		auth      string
		authType  string
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
		{
			title:     "auth header",
			auth:      "token",
			authType:  "Bearer",
			status:    http.StatusOK,
			resp:      api.BuildResponse{},
			expectErr: nil,
		},
		{
			title:     "failed auth",
			status:    http.StatusUnauthorized,
			resp:      api.BuildResponse{},
			expectErr: ErrRequestFailed,
		},
		{
			title: "custom headers",
			headers: map[string]string{
				"Custom-Header": "Custom-Value",
			},
			status:    http.StatusOK,
			resp:      api.BuildResponse{},
			expectErr: nil,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.title, func(t *testing.T) {
			t.Parallel()

			srv := httptest.NewServer(testSrv{
				status:   tc.status,
				response: tc.resp,
				auth:     tc.auth,
				authType: tc.authType,
				headers:  tc.headers,
			})

			client, err := NewBuildServiceClient(
				BuildServiceClientConfig{
					URL:               srv.URL,
					Headers:           tc.headers,
					Authorization:     tc.auth,
					AuthorizationType: tc.authType,
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
