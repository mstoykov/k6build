package k6build

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"testing"
)

type object struct {
	id      string
	content []byte
}

func setupCache(path string, preload []object) (Cache, error) {
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

func TestStoreObject(t *testing.T) {
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
			expectErr: ErrCreatingObject,
		},
		{
			title:     "store invalid id (dot slash)",
			id:        "./invalid",
			content:   []byte("content"),
			expectErr: ErrCreatingObject,
		},
		{
			title:     "store invalid id (trailing slash)",
			id:        "invalid/",
			content:   []byte("content"),
			expectErr: ErrCreatingObject,
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

func TestGetObjectCache(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		title     string
		preload   []object
		id        string
		expected  []byte
		expectErr error
	}{
		{
			title: "retrieve existing object",
			preload: []object{
				{
					id:      "object",
					content: []byte("content"),
				},
			},
			id:        "object",
			expected:  []byte("content"),
			expectErr: nil,
		},
		{
			title: "retrieve non existing object",
			preload: []object{
				{
					id:      "object",
					content: []byte("content"),
				},
			},
			id:        "another object",
			expectErr: ErrObjectNotFound,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.title, func(t *testing.T) {
			t.Parallel()

			cache, err := setupCache(t.TempDir(), tc.preload)
			if err != nil {
				t.Fatalf("test setup: %v", err)
			}

			obj, err := cache.Get(context.TODO(), tc.id)
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
}
