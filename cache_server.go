package k6build

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
)

// CacheServerResponse is the response to a cache server request
type CacheServerResponse struct {
	Error  string
	Object Object
}

// CacheServer implements an http server that handles cache requests
type CacheServer struct {
	baseURL string
	cache   Cache
	log     *slog.Logger
}

// CacheServerConfig defines the configuration for the APIServer
type CacheServerConfig struct {
	BaseURL string
	Cache   Cache
	Log     *slog.Logger
}

// NewCacheServer returns a CacheServer backed by a cache
func NewCacheServer(config CacheServerConfig) http.Handler {
	log := config.Log

	if log == nil {
		log = slog.New(
			slog.NewTextHandler(
				io.Discard,
				&slog.HandlerOptions{},
			),
		)
	}
	cacheSrv := &CacheServer{
		baseURL: config.BaseURL,
		cache:   config.Cache,
		log:     log,
	}

	handler := http.NewServeMux()
	// FIXME: this should be PUT (used POST as http client doesn't have PUT method)
	handler.HandleFunc("POST /{id}", cacheSrv.Store)
	handler.HandleFunc("GET /{id}", cacheSrv.Get)
	handler.HandleFunc("GET /{id}/download", cacheSrv.Download)

	return handler
}

// Get retrieves an objects if exists in the cache or an error otherwise
func (s *CacheServer) Get(w http.ResponseWriter, r *http.Request) {
	resp := CacheServerResponse{}

	w.Header().Add("Content-Type", "application/json")

	// ensure errors are reported and logged
	defer func() {
		if resp.Error != "" {
			s.log.Error(resp.Error)
			_ = json.NewEncoder(w).Encode(resp) //nolint:errchkjson
		}
	}()

	id := r.PathValue("id")
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		resp.Error = ErrInvalidRequest.Error()
		return
	}

	object, err := s.cache.Get(context.Background(), id) //nolint:contextcheck
	if err != nil {
		if errors.Is(err, ErrObjectNotFound) {
			w.WriteHeader(http.StatusNotFound)
		} else {
			w.WriteHeader(http.StatusInternalServerError)
		}
		resp.Error = err.Error()

		return
	}

	// overwrite URL with own
	baseURL := s.baseURL
	if baseURL == "" {
		baseURL = fmt.Sprintf("http://%s%s", r.Host, r.RequestURI)
	}
	downloadURL := fmt.Sprintf("%s/%s/download", baseURL, id)

	resp.Object = Object{
		ID:       id,
		Checksum: object.Checksum,
		URL:      downloadURL,
	}

	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resp) //nolint:errchkjson
}

// Store stores the object and returns the metadata
func (s *CacheServer) Store(w http.ResponseWriter, r *http.Request) {
	resp := CacheServerResponse{}

	w.Header().Add("Content-Type", "application/json")

	// ensure errors are reported and logged
	defer func() {
		if resp.Error != "" {
			s.log.Error(resp.Error)
			_ = json.NewEncoder(w).Encode(resp) //nolint:errchkjson
		}
	}()

	id := r.PathValue("id")
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		resp.Error = ErrInvalidRequest.Error()
		return
	}

	object, err := s.cache.Store(context.Background(), id, r.Body) //nolint:contextcheck
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		resp.Error = err.Error()
		return
	}

	// overwrite URL with own
	baseURL := s.baseURL
	if baseURL == "" {
		baseURL = fmt.Sprintf("http://%s%s", r.Host, r.RequestURI)
	}
	downloadURL := fmt.Sprintf("%s/%s/download", baseURL, id)

	resp.Object = Object{
		ID:       id,
		Checksum: object.Checksum,
		URL:      downloadURL,
	}

	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resp) //nolint:errchkjson
}

// Download returns an object's content given its id
func (s *CacheServer) Download(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
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
