// Package client implements a cache client
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
	"github.com/grafana/k6build/pkg/cache"
	"github.com/grafana/k6build/pkg/cache/api"
)

var (
	ErrAccessingServer = errors.New("making request")        //nolint:revive
	ErrInvalidConfig   = errors.New("invalid configuration") //nolint:revive
	ErrInvalidRequest  = errors.New("invalid request")       //nolint:revive
	ErrInvalidResponse = errors.New("invalid response")      //nolint:revive
	ErrRequestFailed   = errors.New("request failed")        //nolint:revive

)

// CacheClientConfig defines the configuration for accessing a remote cache service
type CacheClientConfig struct {
	Server string
}

// CacheClient access blobs in a CacheServer
type CacheClient struct {
	server string
}

// NewCacheClient returns a client for a cache server
func NewCacheClient(config CacheClientConfig) (*CacheClient, error) {
	if _, err := url.Parse(config.Server); err != nil {
		return nil, k6build.NewError(ErrInvalidConfig, err)
	}

	return &CacheClient{
		server: config.Server,
	}, nil
}

// Get retrieves an objects if exists in the cache or an error otherwise
func (c *CacheClient) Get(_ context.Context, id string) (cache.Object, error) {
	url := fmt.Sprintf("%s/%s", c.server, id)

	// TODO: use http.Request
	resp, err := http.Get(url) //nolint:gosec,noctx
	if err != nil {
		return cache.Object{}, k6build.NewError(ErrAccessingServer, err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusNotFound {
			return cache.Object{}, cache.ErrObjectNotFound
		}
		return cache.Object{}, k6build.NewError(ErrRequestFailed, fmt.Errorf("status %s", resp.Status))
	}

	cacheResponse := api.CacheResponse{}
	err = json.NewDecoder(resp.Body).Decode(&cacheResponse)
	if err != nil {
		return cache.Object{}, k6build.NewError(ErrInvalidResponse, err)
	}

	if cacheResponse.Error != nil {
		return cache.Object{}, k6build.NewError(ErrRequestFailed, cacheResponse.Error)
	}

	return cacheResponse.Object, nil
}

// Store stores the object and returns the metadata
func (c *CacheClient) Store(_ context.Context, id string, content io.Reader) (cache.Object, error) {
	url := fmt.Sprintf("%s/%s", c.server, id)
	resp, err := http.Post( //nolint:gosec,noctx
		url,
		"application/octet-stream",
		content,
	)
	if err != nil {
		return cache.Object{}, k6build.NewError(ErrAccessingServer, err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return cache.Object{}, k6build.NewError(ErrRequestFailed, fmt.Errorf("status %s", resp.Status))
	}
	cacheResponse := api.CacheResponse{}
	err = json.NewDecoder(resp.Body).Decode(&cacheResponse)
	if err != nil {
		return cache.Object{}, k6build.NewError(ErrInvalidResponse, err)
	}

	if cacheResponse.Error != nil {
		return cache.Object{}, k6build.NewError(ErrRequestFailed, cacheResponse.Error)
	}

	return cacheResponse.Object, nil
}

// Download returns the content of the object given its url
func (c *CacheClient) Download(_ context.Context, object cache.Object) (io.ReadCloser, error) {
	resp, err := http.Get(object.URL) //nolint:noctx,bodyclose
	if err != nil {
		return nil, k6build.NewError(ErrAccessingServer, err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, k6build.NewError(ErrRequestFailed, fmt.Errorf("status %s", resp.Status))
	}

	return resp.Request.Body, nil
}
