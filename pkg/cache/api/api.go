// Package api defines the interface to a cache server
package api

import (
	"github.com/grafana/k6build"
	"github.com/grafana/k6build/pkg/cache"
)

// CacheResponse is the response to a cache server request
type CacheResponse struct {
	Error  *k6build.Error
	Object cache.Object
}
