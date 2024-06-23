package k6build

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

// returns a HandleFunc that returns a canned status and response
func handlerMock(status int, resp *CacheServerResponse) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Add("Content-Type", "application/json")

		// send canned response
		respBuffer := &bytes.Buffer{}
		if resp != nil {
			err := json.NewEncoder(respBuffer).Encode(resp)
			if err != nil {
				// set uncommon status code to signal something unexpected happened
				w.WriteHeader(http.StatusTeapot)
				return
			}
		}

		w.WriteHeader(status)
		_, _ = w.Write(respBuffer.Bytes())
	}
}

// returns a HandleFunc that returns a canned status and content for a download
func downloadMock(status int, content []byte) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Add("Content-Type", "application/octet-stream")
		w.WriteHeader(status)
		if content != nil {
			_, _ = w.Write(content)
		}
	}
}

func TestCacheClientGet(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		title     string
		status    int
		resp      *CacheServerResponse
		expectErr error
	}{
		{
			title:  "normal get",
			status: http.StatusOK,
			resp: &CacheServerResponse{
				Error:  "",
				Object: Object{},
			},
		},
		{
			title:     "object not found",
			status:    http.StatusNotFound,
			resp:      nil,
			expectErr: ErrObjectNotFound,
		},
		{
			title:  "error accessing object",
			status: http.StatusInternalServerError,
			resp: &CacheServerResponse{
				Error:  "Error accessing object",
				Object: Object{},
			},
			expectErr: ErrRequestFailed,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.title, func(t *testing.T) {
			t.Parallel()

			srv := httptest.NewServer(handlerMock(tc.status, tc.resp))

			client, err := NewCacheClient(CacheClientConfig{Server: srv.URL})
			if err != nil {
				t.Fatalf("test setup %v", err)
			}

			_, err = client.Get(context.TODO(), "object")
			if !errors.Is(err, tc.expectErr) {
				t.Fatalf("expected %v got %v", tc.expectErr, err)
			}
		})
	}
}

func TestCacheClientStore(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		title     string
		status    int
		resp      *CacheServerResponse
		expectErr error
	}{
		{
			title:  "normal response",
			status: http.StatusOK,
			resp: &CacheServerResponse{
				Error:  "",
				Object: Object{},
			},
		},
		{
			title:  "error creating object",
			status: http.StatusInternalServerError,
			resp: &CacheServerResponse{
				Error:  "Error creating object",
				Object: Object{},
			},
			expectErr: ErrRequestFailed,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.title, func(t *testing.T) {
			t.Parallel()

			srv := httptest.NewServer(handlerMock(tc.status, tc.resp))

			client, err := NewCacheClient(CacheClientConfig{Server: srv.URL})
			if err != nil {
				t.Fatalf("test setup %v", err)
			}

			_, err = client.Store(context.TODO(), "object", bytes.NewBuffer(nil))
			if !errors.Is(err, tc.expectErr) {
				t.Fatalf("expected %v got %v", tc.expectErr, err)
			}
		})
	}
}

func TestCacheClientDownload(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		title     string
		status    int
		content   []byte
		expectErr error
	}{
		{
			title:   "normal response",
			status:  http.StatusOK,
			content: []byte("object content"),
		},
		{
			title:     "error creating object",
			status:    http.StatusInternalServerError,
			expectErr: ErrRequestFailed,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.title, func(t *testing.T) {
			t.Parallel()

			srv := httptest.NewServer(downloadMock(tc.status, tc.content))

			client, err := NewCacheClient(CacheClientConfig{Server: srv.URL})
			if err != nil {
				t.Fatalf("test setup %v", err)
			}

			obj := Object{
				ID:  "object",
				URL: srv.URL,
			}
			_, err = client.Download(context.TODO(), obj)
			if !errors.Is(err, tc.expectErr) {
				t.Fatalf("expected %v got %v", tc.expectErr, err)
			}
		})
	}
}
