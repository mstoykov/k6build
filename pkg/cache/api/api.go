// Package api defines the interface to a cache server
package api

import (
	"errors"

	"github.com/grafana/k6build"
	"github.com/grafana/k6build/pkg/cache"
)

var (
	// ErrInvalidRequest signals the request could not be processed
	// due to erroneous parameters
	ErrInvalidRequest = errors.New("invalid request")
	// ErrRequestFailed signals the request failed, probably due to a network error
	ErrRequestFailed = errors.New("request failed")
	// ErrCacheAccess signals the access to the cache failed
	ErrCacheAccess = errors.New("cache access failed")
)

// CacheResponse is the response to a cache server request
type CacheResponse struct {
	Error  *k6build.WrappedError
	Object cache.Object
}
