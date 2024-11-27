package server

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

	"github.com/grafana/k6build/pkg/store"
	"github.com/grafana/k6build/pkg/store/api"
)

// MemoryStore implements a memory backed object store
type MemoryStore struct {
	objects map[string]store.Object
	content map[string][]byte
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		objects: map[string]store.Object{},
		content: map[string][]byte{},
	}
}

func (f *MemoryStore) Get(_ context.Context, id string) (store.Object, error) {
	object, found := f.objects[id]
	if !found {
		return store.Object{}, store.ErrObjectNotFound
	}

	return object, nil
}

func (f *MemoryStore) Put(_ context.Context, id string, content io.Reader) (store.Object, error) {
	buffer := bytes.Buffer{}
	_, err := buffer.ReadFrom(content)
	if err != nil {
		return store.Object{}, store.ErrCreatingObject
	}

	checksum := fmt.Sprintf("%x", sha256.Sum256(buffer.Bytes()))
	object := store.Object{
		ID:       id,
		Checksum: checksum,
		URL:      fmt.Sprintf("memory:///%s", id),
	}

	f.objects[id] = object
	f.content[id] = buffer.Bytes()

	return object, nil
}

func (f *MemoryStore) Download(_ context.Context, object store.Object) (io.ReadCloser, error) {
	url, err := url.Parse(object.URL)
	if err != nil {
		return nil, err
	}

	id, _ := strings.CutPrefix(url.Path, "/")
	content, found := f.content[id]
	if !found {
		return nil, store.ErrObjectNotFound
	}

	return io.NopCloser(bytes.NewBuffer(content)), nil
}

func TestStoreServerGet(t *testing.T) {
	t.Parallel()

	store := NewMemoryStore()
	objects := map[string][]byte{
		"object1": []byte("content object 1"),
	}

	for id, content := range objects {
		buffer := bytes.NewBuffer(content)
		if _, err := store.Put(context.TODO(), id, buffer); err != nil {
			t.Fatalf("test setup: %v", err)
		}
	}

	config := StoreServerConfig{
		Store: store,
	}
	storeSrv := NewStoreServer(config)

	srv := httptest.NewServer(storeSrv)

	testCases := []struct {
		title    string
		id       string
		status   int
		epectErr error
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

			storeResponse := api.StoreResponse{}
			err = json.NewDecoder(resp.Body).Decode(&storeResponse)
			if err != nil {
				t.Fatalf("reading response content %v", err)
			}

			if tc.status != http.StatusOK {
				if storeResponse.Error == nil {
					t.Fatalf("expected error message not none")
				}
				return
			}

			if storeResponse.Object.ID != tc.id {
				t.Fatalf("expected object id %s got %s", tc.id, storeResponse.Object.ID)
			}
		})
	}
}

func TestStoreServerPut(t *testing.T) {
	t.Parallel()

	store := NewMemoryStore()

	config := StoreServerConfig{
		Store: store,
	}
	storeSrv := NewStoreServer(config)

	srv := httptest.NewServer(storeSrv)

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

			storeResponse := api.StoreResponse{}
			err = json.NewDecoder(resp.Body).Decode(&storeResponse)
			if err != nil {
				t.Fatalf("reading response content %v", err)
			}

			if tc.status != http.StatusOK {
				if storeResponse.Error == nil {
					t.Fatalf("expected error message not none")
				}
				return
			}

			if storeResponse.Object.ID != tc.id {
				t.Fatalf("expected object id %s got %s", tc.id, storeResponse.Object.ID)
			}
		})
	}
}

func TestStoreServerDownload(t *testing.T) {
	t.Parallel()

	store := NewMemoryStore()
	objects := map[string][]byte{
		"object1": []byte("content object 1"),
	}

	for id, content := range objects {
		buffer := bytes.NewBuffer(content)
		if _, err := store.Put(context.TODO(), id, buffer); err != nil {
			t.Fatalf("test setup: %v", err)
		}
	}

	config := StoreServerConfig{
		Store: store,
	}
	storeSrv := NewStoreServer(config)

	srv := httptest.NewServer(storeSrv)

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
