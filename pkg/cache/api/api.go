// Package api defines the interface to a cache server
package api

import (
	"github.com/grafana/k6build/pkg/cache"
)

// CacheResponse is the response to a cache server request
type CacheResponse struct {
	Error  string
	Object cache.Object
}
