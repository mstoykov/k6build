package k6build

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

// MemoryCache defines the state of a memory backed cache
type MemoryCache struct {
	objects map[string]Object
	content map[string][]byte
}

func NewMemoryCache() *MemoryCache {
	return &MemoryCache{
		objects: map[string]Object{},
		content: map[string][]byte{},
	}
}

func (f *MemoryCache) Get(_ context.Context, id string) (Object, error) {
	object, found := f.objects[id]
	if !found {
		return Object{}, ErrObjectNotFound
	}

	return object, nil
}

func (f *MemoryCache) Store(_ context.Context, id string, content io.Reader) (Object, error) {
	buffer := bytes.Buffer{}
	_, err := buffer.ReadFrom(content)
	if err != nil {
		return Object{}, ErrCreatingObject
	}

	checksum := fmt.Sprintf("%x", sha256.Sum256(buffer.Bytes()))
	object := Object{
		ID:       id,
		Checksum: checksum,
		URL:      fmt.Sprintf("memory:///%s", id),
	}

	f.objects[id] = object
	f.content[id] = buffer.Bytes()

	return object, nil
}

// Download implements Cache.
func (f *MemoryCache) Download(_ context.Context, object Object) (io.ReadCloser, error) {
	url, err := url.Parse(object.URL)
	if err != nil {
		return nil, err
	}

	id, _ := strings.CutPrefix(url.Path, "/")
	content, found := f.content[id]
	if !found {
		return nil, ErrObjectNotFound
	}

	return io.NopCloser(bytes.NewBuffer(content)), nil
}

func TestCacheServerGet(t *testing.T) {
	t.Parallel()

	cache := NewMemoryCache()
	objects := map[string][]byte{
		"object1": []byte("content object 1"),
	}

	for id, content := range objects {
		buffer := bytes.NewBuffer(content)
		if _, err := cache.Store(context.TODO(), id, buffer); err != nil {
			t.Fatalf("test setup: %v", err)
		}
	}

	config := CacheServerConfig{
		Cache: cache,
	}
	cacheSrv := NewCacheServer(config)

	srv := httptest.NewServer(cacheSrv)

	testCases := []struct {
		title    string
		id       string
		status   int
		epectErr string
	}{
		{
			title:  "return object",
			id:     "object1",
			status: http.StatusOK,
		},
		{
			title:  "object not found",
			id:     "not_found",
			status: http.StatusNotFound,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.title, func(t *testing.T) {
			t.Parallel()

			url := fmt.Sprintf("%s/%s", srv.URL, tc.id)
			resp, err := http.Get(url)
			if err != nil {
				t.Fatalf("accessing server %v", err)
			}
			defer func() {
				_ = resp.Body.Close()
			}()

			if resp.StatusCode != tc.status {
				t.Fatalf("expected %s got %s", http.StatusText(tc.status), resp.Status)
			}

			cacheResponse := CacheServerResponse{}
			err = json.NewDecoder(resp.Body).Decode(&cacheResponse)
			if err != nil {
				t.Fatalf("reading response content %v", err)
			}

			if tc.status != http.StatusOK {
				if cacheResponse.Error == "" {
					t.Fatalf("expected error message not none")
				}
				return
			}

			if cacheResponse.Object.ID != tc.id {
				t.Fatalf("expected object id %s got %s", tc.id, cacheResponse.Object.ID)
			}
		})
	}
}

func TestCacheServerStore(t *testing.T) {
	t.Parallel()

	cache := NewMemoryCache()

	config := CacheServerConfig{
		Cache: cache,
	}
	cacheSrv := NewCacheServer(config)

	srv := httptest.NewServer(cacheSrv)

	testCases := []struct {
		title   string
		id      string
		content string
		status  int
	}{
		{
			title:   "create object",
			id:      "object1",
			content: "object 1 content",
			status:  http.StatusOK,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.title, func(t *testing.T) {
			t.Parallel()

			url := fmt.Sprintf("%s/%s", srv.URL, tc.id)
			resp, err := http.Post(
				url,
				"application/octet-stream",
				bytes.NewBufferString(tc.content),
			)
			if err != nil {
				t.Fatalf("accessing server %v", err)
			}
			defer func() {
				_ = resp.Body.Close()
			}()

			if resp.StatusCode != tc.status {
				t.Fatalf("expected %s got %s", http.StatusText(tc.status), resp.Status)
			}

			cacheResponse := CacheServerResponse{}
			err = json.NewDecoder(resp.Body).Decode(&cacheResponse)
			if err != nil {
				t.Fatalf("reading response content %v", err)
			}

			if tc.status != http.StatusOK {
				if cacheResponse.Error == "" {
					t.Fatalf("expected error message not none")
				}
				return
			}

			if cacheResponse.Object.ID != tc.id {
				t.Fatalf("expected object id %s got %s", tc.id, cacheResponse.Object.ID)
			}
		})
	}
}

func TestCacheServerDownload(t *testing.T) {
	t.Parallel()

	cache := NewMemoryCache()
	objects := map[string][]byte{
		"object1": []byte("content object 1"),
	}

	for id, content := range objects {
		buffer := bytes.NewBuffer(content)
		if _, err := cache.Store(context.TODO(), id, buffer); err != nil {
			t.Fatalf("test setup: %v", err)
		}
	}

	config := CacheServerConfig{
		Cache: cache,
	}
	cacheSrv := NewCacheServer(config)

	srv := httptest.NewServer(cacheSrv)

	testCases := []struct {
		title   string
		id      string
		status  int
		content []byte
	}{
		{
			title:   "return object",
			id:      "object1",
			status:  http.StatusOK,
			content: objects["object1"],
		},
		{
			title:  "object not found",
			id:     "not_found",
			status: http.StatusNotFound,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.title, func(t *testing.T) {
			t.Parallel()

			url := fmt.Sprintf("%s/%s/download", srv.URL, tc.id)
			resp, err := http.Get(url)
			if err != nil {
				t.Fatalf("accessing server %v", err)
			}
			defer func() {
				_ = resp.Body.Close()
			}()

			if resp.StatusCode != tc.status {
				t.Fatalf("expected %s got %s", http.StatusText(tc.status), resp.Status)
			}

			if tc.status != http.StatusOK {
				return
			}

			content := bytes.Buffer{}
			_, err = content.ReadFrom(resp.Body)
			if err != nil {
				t.Fatalf("reading content %v", err)
			}

			if !bytes.Equal(content.Bytes(), tc.content) {
				t.Fatalf("expected got")
			}
		})
	}
}
