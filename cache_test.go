package k6build

import (
	"bytes"
	"context"
	"errors"
	"net/url"
	"os"
	"testing"
)

func TestCreateObject(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		title     string
		content   []byte
		id        string
		expectErr error
	}{
		{
			title:   "store object",
			content: []byte("content"),
		},
		{
			title:   "store empty object",
			content: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.title, func(t *testing.T) {
			t.Parallel()

			cache, err := NewFileCache(t.TempDir())
			if err != nil {
				t.Fatalf("test setup %v", err)
			}

			obj, err := cache.Store(context.TODO(), "object", bytes.NewBuffer(tc.content))
			if !errors.Is(err, tc.expectErr) {
				t.Fatalf("expected %v got %v", tc.expectErr, err)
			}

			fileUrl, err := url.Parse(obj.URL)
			if err != nil {
				t.Fatalf("invalid url %v", err)
			}

			content, err := os.ReadFile(fileUrl.Path)
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
		id        string
		expectErr error
	}{
		{
			title:     "retrieve existing",
			id:        "object",
			expectErr: nil,
		},
		{
			title:     "retrieve non existing object",
			id:        "object2",
			expectErr: ErrObjectNotFound,
		},
	}

	cache, err := NewFileCache(t.TempDir())
	if err != nil {
		t.Fatalf("test setup %v", err)
	}

	content := []byte("content")
	_, err = cache.Store(context.TODO(), "object", bytes.NewBuffer(content))
	if err != nil {
		t.Fatalf("test setup %v", err)
	}

	for _, tc := range testCases {
		t.Run(tc.title, func(t *testing.T) {
			t.Parallel()

			obj, err := cache.Get(context.TODO(), tc.id)
			if !errors.Is(err, tc.expectErr) {
				t.Fatalf("expected %v got %v", tc.expectErr, err)
			}

			// if expected error, don't check returned object
			if tc.expectErr != nil {
				return
			}

			fileUrl, err := url.Parse(obj.URL)
			if err != nil {
				t.Fatalf("invalid url %v", err)
			}

			data, err := os.ReadFile(fileUrl.Path)
			if err != nil {
				t.Fatalf("reading object url %v", err)
			}

			if !bytes.Equal(data, content) {
				t.Fatalf("expected %v got %v", data, content)
			}
		})
	}
}
