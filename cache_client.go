package k6build

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
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
		return nil, fmt.Errorf("%w: %w", ErrInvalidConfig, err)
	}

	return &CacheClient{
		server: config.Server,
	}, nil
}

// Get retrieves an objects if exists in the cache or an error otherwise
func (c *CacheClient) Get(_ context.Context, id string) (Object, error) {
	url := fmt.Sprintf("%s/get?id=%s", c.server, id)

	// TODO: use http.Request
	resp, err := http.Get(url) //nolint:gosec,noctx
	if err != nil {
		return Object{}, fmt.Errorf("%w %w", ErrAccessingServer, err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusNotFound {
			return Object{}, fmt.Errorf("%w with status", ErrObjectNotFound)
		}
		return Object{}, fmt.Errorf("%w with status %s", ErrRequestFailed, resp.Status)
	}

	cacheResponse := CacheServerResponse{}
	err = json.NewDecoder(resp.Body).Decode(&cacheResponse)
	if err != nil {
		return Object{}, fmt.Errorf("%w: %s", ErrInvalidResponse, err.Error())
	}

	if cacheResponse.Error != "" {
		return Object{}, fmt.Errorf("%w: %s", ErrRequestFailed, cacheResponse.Error)
	}

	return cacheResponse.Object, nil
}

// Store stores the object and returns the metadata
func (c *CacheClient) Store(_ context.Context, id string, content io.Reader) (Object, error) {
	url := fmt.Sprintf("%s/store?id=%s", c.server, id)
	resp, err := http.Post( //nolint:gosec,noctx
		url,
		"application/octet-stream",
		content,
	)
	if err != nil {
		return Object{}, fmt.Errorf("%w %w", ErrAccessingServer, err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return Object{}, fmt.Errorf("%w with status %s", ErrRequestFailed, resp.Status)
	}

	cacheResponse := CacheServerResponse{}
	err = json.NewDecoder(resp.Body).Decode(&cacheResponse)
	if err != nil {
		return Object{}, fmt.Errorf("%w: %s", ErrInvalidResponse, err.Error())
	}

	if cacheResponse.Error != "" {
		return Object{}, fmt.Errorf("%w: %s", ErrRequestFailed, cacheResponse.Error)
	}

	return cacheResponse.Object, nil
}

// Download returns the content of the object given its url
func (c *CacheClient) Download(_ context.Context, object Object) (io.ReadCloser, error) {
	resp, err := http.Get(object.URL) //nolint:noctx,bodyclose
	if err != nil {
		return nil, fmt.Errorf("%w %w", ErrAccessingServer, err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w with status %s", ErrRequestFailed, resp.Status)
	}

	return resp.Request.Body, nil
}
