package k6build

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

// CacheServerResponse is the response to a cache server request
type CacheServerResponse struct {
	Error  string
	Object Object
}

// CacheServer implements an http server that handles cache requests
type CacheServer struct {
	cache   Cache
	baseURL string
}

// NewCacheServer returns a CacheServer backed by a cache
func NewCacheServer(baseURL string, cache Cache) http.Handler {
	cacheSrv := &CacheServer{
		baseURL: baseURL,
		cache:   cache,
	}

	handler := http.NewServeMux()
	handler.HandleFunc("/store", cacheSrv.Store)
	handler.HandleFunc("/get", cacheSrv.Get)
	handler.HandleFunc("/download", cacheSrv.Download)

	return handler
}

// Get retrieves an objects if exists in the cache or an error otherwise
func (s *CacheServer) Get(w http.ResponseWriter, r *http.Request) {
	resp := CacheServerResponse{}

	id := r.URL.Query().Get("id")
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	object, err := s.cache.Get(context.Background(), id) //nolint:contextcheck
	if err != nil {
		if errors.Is(err, ErrObjectNotFound) {
			w.WriteHeader(http.StatusNotFound)
		} else {
			w.WriteHeader(http.StatusInternalServerError)
		}
		return
	}

	// overwrite URL with own
	resp.Object = Object{
		ID:       id,
		Checksum: object.Checksum,
		URL:      fmt.Sprintf(url.JoinPath(s.baseURL, object.ID)),
	}

	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resp) //nolint:errchkjson
}

// Store stores the object and returns the metadata
func (s *CacheServer) Store(w http.ResponseWriter, r *http.Request) {
	resp := CacheServerResponse{}

	id := r.URL.Query().Get("id")
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	object, err := s.cache.Store(context.Background(), id, r.Body) //nolint:contextcheck
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// overwrite URL with own
	resp.Object = Object{
		ID:       id,
		Checksum: object.Checksum,
		URL:      fmt.Sprintf(url.JoinPath(s.baseURL, object.ID)),
	}

	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resp) //nolint:errchkjson
}

// Download returns an object's content given its id
func (s *CacheServer) Download(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	object, err := s.cache.Get(context.Background(), id) //nolint:contextcheck
	if err != nil {
		if errors.Is(err, ErrObjectNotFound) {
			w.WriteHeader(http.StatusNotFound)
		} else {
			w.WriteHeader(http.StatusInternalServerError)
		}
		return
	}

	objectContent, err := s.cache.Download(context.Background(), object) //nolint:contextcheck
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer func() {
		_ = objectContent.Close()
	}()

	w.WriteHeader(http.StatusOK)
	w.Header().Add("Content-Type", "application/octet-stream")
	w.Header().Add("ETag", object.ID)
	_, _ = io.Copy(w, objectContent)
}
