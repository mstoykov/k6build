package k6build

import (
	"bytes"
	"context"
	"errors"
	"net/url"
	"os"
	"testing"
)

func TestFileCache(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		title     string
		content   []byte
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
				t.Fatalf("expected %v got %v", tc, err)
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
