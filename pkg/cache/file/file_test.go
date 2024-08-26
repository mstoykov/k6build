package file

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"testing"

	"github.com/grafana/k6build/pkg/cache"
)

type object struct {
	id      string
	content []byte
}

func setupCache(path string, preload []object) (cache.Cache, error) {
	cache, err := NewFileCache(path)
	if err != nil {
		return nil, fmt.Errorf("test setup %w", err)
	}

	for _, o := range preload {
		_, err = cache.Store(context.TODO(), o.id, bytes.NewBuffer(o.content))
		if err != nil {
			return nil, fmt.Errorf("test setup %w", err)
		}
	}

	return cache, nil
}

func TestFileCacheStoreObject(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		title     string
		preload   []object
		id        string
		content   []byte
		expectErr error
	}{
		{
			title:   "store object",
			id:      "object",
			content: []byte("content"),
		},
		{
			title: "store existing object",
			preload: []object{
				{
					id:      "object",
					content: []byte("content"),
				},
			},
			id:      "object",
			content: []byte("new content"),
		},
		{
			title:   "store empty object",
			id:      "empty",
			content: nil,
		},
		{
			title:     "store empty id",
			id:        "",
			content:   []byte("content"),
			expectErr: cache.ErrCreatingObject,
		},
		{
			title:     "store invalid id (dot slash)",
			id:        "./invalid",
			content:   []byte("content"),
			expectErr: cache.ErrCreatingObject,
		},
		{
			title:     "store invalid id (trailing slash)",
			id:        "invalid/",
			content:   []byte("content"),
			expectErr: cache.ErrCreatingObject,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.title, func(t *testing.T) {
			t.Parallel()
			cache, err := setupCache(t.TempDir(), tc.preload)
			if err != nil {
				t.Fatalf("test setup: %v", err)
			}

			obj, err := cache.Store(context.TODO(), tc.id, bytes.NewBuffer(tc.content))
			if !errors.Is(err, tc.expectErr) {
				t.Fatalf("expected %v got %v", tc.expectErr, err)
			}

			// if expected error, don't validate object
			if tc.expectErr != nil {
				return
			}

			fileURL, err := url.Parse(obj.URL)
			if err != nil {
				t.Fatalf("invalid url %v", err)
			}

			content, err := os.ReadFile(fileURL.Path)
			if err != nil {
				t.Fatalf("reading object url %v", err)
			}

			if !bytes.Equal(tc.content, content) {
				t.Fatalf("expected %v got %v", tc.content, content)
			}
		})
	}
}

func TestFileCacheRetrieval(t *testing.T) {
	t.Parallel()

	preload := []object{
		{
			id:      "object",
			content: []byte("content"),
		},
	}

	cacheDir := t.TempDir()
	fileCache, err := setupCache(cacheDir, preload)
	if err != nil {
		t.Fatalf("test setup: %v", err)
	}

	t.Run("TestFileCacheGet", func(t *testing.T) {
		testCases := []struct {
			title     string
			id        string
			expected  []byte
			expectErr error
		}{
			{
				title:     "retrieve existing object",
				id:        "object",
				expected:  []byte("content"),
				expectErr: nil,
			},
			{
				title:     "retrieve non existing object",
				id:        "another object",
				expectErr: cache.ErrObjectNotFound,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.title, func(t *testing.T) {
				t.Parallel()

				obj, err := fileCache.Get(context.TODO(), tc.id)
				if !errors.Is(err, tc.expectErr) {
					t.Fatalf("expected %v got %v", tc.expectErr, err)
				}

				// if expected error, don't check returned object
				if tc.expectErr != nil {
					return
				}

				fileURL, err := url.Parse(obj.URL)
				if err != nil {
					t.Fatalf("invalid url %v", err)
				}

				data, err := os.ReadFile(fileURL.Path)
				if err != nil {
					t.Fatalf("reading object url %v", err)
				}

				if !bytes.Equal(data, tc.expected) {
					t.Fatalf("expected %v got %v", tc.expected, data)
				}
			})
		}
	})

	// FIXME: This test is leaking how the file cache creates the URLs for the objects
	t.Run("TestFileCacheDownload", func(t *testing.T) {
		t.Parallel()

		testCases := []struct {
			title     string
			object    cache.Object
			expected  []byte
			expectErr error
		}{
			{
				title: "download existing object",
				object: cache.Object{
					ID:  "object",
					URL: fmt.Sprintf("file://%s/object/data", cacheDir),
				},
				expected:  []byte("content"),
				expectErr: nil,
			},
			{
				title: "download non existing object",
				object: cache.Object{
					ID:  "object",
					URL: fmt.Sprintf("file://%s/another_object/data", cacheDir),
				},
				expectErr: cache.ErrObjectNotFound,
			},
			{
				title: "download malformed url",
				object: cache.Object{
					ID:  "object",
					URL: fmt.Sprintf("file://%s/invalid&path/data", cacheDir),
				},
				// FIXME: this should be an ErrInvalidURL
				expectErr: cache.ErrObjectNotFound,
			},
			{
				title: "download malicious url",
				object: cache.Object{
					ID:  "object",
					URL: fmt.Sprintf("file://%s/../../data", cacheDir),
				},
				expectErr: cache.ErrInvalidURL,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.title, func(t *testing.T) {
				t.Parallel()

				content, err := fileCache.Download(context.TODO(), tc.object)
				if !errors.Is(err, tc.expectErr) {
					t.Fatalf("expected %v got %v", tc.expectErr, err)
				}

				// if expected error, don't check returned object
				if tc.expectErr != nil {
					return
				}

				data := bytes.Buffer{}
				_, err = data.ReadFrom(content)
				if err != nil {
					t.Fatalf("reading content: %v", err)
				}

				if !bytes.Equal(data.Bytes(), tc.expected) {
					t.Fatalf("expected %v got %v", tc.expected, data)
				}
			})
		}
	})
}
