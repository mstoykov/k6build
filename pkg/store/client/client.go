// Package client implements an object store service client
package client

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/grafana/k6build"
	"github.com/grafana/k6build/pkg/store"
	"github.com/grafana/k6build/pkg/store/api"
)

// ErrInvalidConfig signals an error with the client configuration
var ErrInvalidConfig = errors.New("invalid configuration")

// StoreClientConfig defines the configuration for accessing a remote object store service
type StoreClientConfig struct {
	Server string
}

// StoreClient access blobs in a StoreServer
type StoreClient struct {
	server string
}

// NewStoreClient returns a client for an object store server
func NewStoreClient(config StoreClientConfig) (*StoreClient, error) {
	if _, err := url.Parse(config.Server); err != nil {
		return nil, k6build.NewWrappedError(ErrInvalidConfig, err)
	}

	return &StoreClient{
		server: config.Server,
	}, nil
}

// Get retrieves an objects if exists in the store or an error otherwise
func (c *StoreClient) Get(_ context.Context, id string) (store.Object, error) {
	url := fmt.Sprintf("%s/%s", c.server, id)

	// TODO: use http.Request
	resp, err := http.Get(url) //nolint:gosec,noctx
	if err != nil {
		return store.Object{}, k6build.NewWrappedError(api.ErrRequestFailed, err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusNotFound {
			return store.Object{}, store.ErrObjectNotFound
		}
		return store.Object{}, k6build.NewWrappedError(api.ErrRequestFailed, fmt.Errorf("status %s", resp.Status))
	}

	storeResponse := api.StoreResponse{}
	err = json.NewDecoder(resp.Body).Decode(&storeResponse)
	if err != nil {
		return store.Object{}, k6build.NewWrappedError(api.ErrRequestFailed, err)
	}

	if storeResponse.Error != nil {
		return store.Object{}, storeResponse.Error
	}

	return storeResponse.Object, nil
}

// Put stores the object and returns the metadata
func (c *StoreClient) Put(_ context.Context, id string, content io.Reader) (store.Object, error) {
	url := fmt.Sprintf("%s/%s", c.server, id)
	resp, err := http.Post( //nolint:gosec,noctx
		url,
		"application/octet-stream",
		content,
	)
	if err != nil {
		return store.Object{}, k6build.NewWrappedError(api.ErrRequestFailed, err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return store.Object{}, k6build.NewWrappedError(api.ErrRequestFailed, fmt.Errorf("status %s", resp.Status))
	}
	storeResponse := api.StoreResponse{}
	err = json.NewDecoder(resp.Body).Decode(&storeResponse)
	if err != nil {
		return store.Object{}, k6build.NewWrappedError(api.ErrRequestFailed, err)
	}

	if storeResponse.Error != nil {
		return store.Object{}, storeResponse.Error
	}

	return storeResponse.Object, nil
}

// Download returns the content of the object given its url
func (c *StoreClient) Download(_ context.Context, object store.Object) (io.ReadCloser, error) {
	resp, err := http.Get(object.URL) //nolint:noctx,bodyclose
	if err != nil {
		return nil, k6build.NewWrappedError(api.ErrRequestFailed, err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, k6build.NewWrappedError(api.ErrRequestFailed, fmt.Errorf("status %s", resp.Status))
	}

	return resp.Request.Body, nil
}
