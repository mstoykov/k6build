// Package file implements a file-backed object store
package file

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/grafana/k6build"
	"github.com/grafana/k6build/pkg/store"
	"github.com/grafana/k6build/pkg/util"
)

// Store a ObjectStore backed by a file system
type Store struct {
	dir     string
	mutexes sync.Map
}

// NewTempFileStore creates a file object store using a temporary file
func NewTempFileStore() (store.ObjectStore, error) {
	return NewFileStore(filepath.Join(os.TempDir(), "k6build", "objectstore"))
}

// NewFileStore creates an object store backed by a directory
func NewFileStore(dir string) (store.ObjectStore, error) {
	err := os.MkdirAll(dir, 0o750)
	if err != nil {
		return nil, k6build.NewWrappedError(store.ErrInitializingStore, err)
	}

	return &Store{
		dir: dir,
	}, nil
}

// Put stores the object and returns the metadata
// Fails if the object already exists
func (f *Store) Put(_ context.Context, id string, content io.Reader) (store.Object, error) {
	if id == "" {
		return store.Object{}, fmt.Errorf("%w: id cannot be empty", store.ErrCreatingObject)
	}

	if strings.Contains(id, "/") {
		return store.Object{}, fmt.Errorf("%w id cannot contain '/'", store.ErrCreatingObject)
	}

	// prevent concurrent modification of an object
	unlock := f.lockObject(id)
	defer unlock()

	objectDir := filepath.Join(f.dir, id)

	if _, err := os.Stat(objectDir); !errors.Is(err, os.ErrNotExist) {
		return store.Object{}, fmt.Errorf("%w: object already exists %q", store.ErrCreatingObject, id)
	}

	// TODO: check permissions
	err := os.MkdirAll(objectDir, 0o750)
	if err != nil {
		return store.Object{}, k6build.NewWrappedError(store.ErrCreatingObject, err)
	}

	objectFile, err := os.Create(filepath.Join(objectDir, "data")) //nolint:gosec
	if err != nil {
		return store.Object{}, k6build.NewWrappedError(store.ErrCreatingObject, err)
	}
	defer objectFile.Close() //nolint:errcheck

	// write content to object file and copy to buffer to calculate checksum
	// TODO: optimize memory by copying content in blocks
	buff := bytes.Buffer{}
	_, err = io.Copy(objectFile, io.TeeReader(content, &buff))
	if err != nil {
		return store.Object{}, k6build.NewWrappedError(store.ErrCreatingObject, err)
	}

	// calculate checksum
	checksum := fmt.Sprintf("%x", sha256.Sum256(buff.Bytes()))

	// write metadata
	err = os.WriteFile(filepath.Join(objectDir, "checksum"), []byte(checksum), 0o644) //nolint:gosec
	if err != nil {
		return store.Object{}, k6build.NewWrappedError(store.ErrCreatingObject, err)
	}

	objectURL, _ := util.URLFromFilePath(objectFile.Name())
	return store.Object{
		ID:       id,
		Checksum: checksum,
		URL:      objectURL.String(),
	}, nil
}

// Get retrieves an objects if exists in the object store or an error otherwise
func (f *Store) Get(_ context.Context, id string) (store.Object, error) {
	objectDir := filepath.Join(f.dir, id)
	_, err := os.Stat(objectDir)

	if errors.Is(err, os.ErrNotExist) {
		return store.Object{}, fmt.Errorf("%w (%s)", store.ErrObjectNotFound, id)
	}

	if err != nil {
		return store.Object{}, k6build.NewWrappedError(store.ErrAccessingObject, err)
	}

	checksum, err := os.ReadFile(filepath.Join(objectDir, "checksum")) //nolint:gosec
	if err != nil {
		return store.Object{}, k6build.NewWrappedError(store.ErrAccessingObject, err)
	}

	objectURL, _ := util.URLFromFilePath(filepath.Join(objectDir, "data"))
	return store.Object{
		ID:       id,
		Checksum: string(checksum),
		URL:      objectURL.String(),
	}, nil
}

// Download returns the content of the object given its url
func (f *Store) Download(_ context.Context, object store.Object) (io.ReadCloser, error) {
	url, err := url.Parse(object.URL)
	if err != nil {
		return nil, k6build.NewWrappedError(store.ErrAccessingObject, err)
	}

	switch url.Scheme {
	case "file":
		objectPath, err := util.URLToFilePath(url)
		if err != nil {
			return nil, err
		}

		// prevent malicious path
		objectPath, err = f.sanitizePath(objectPath)
		if err != nil {
			return nil, err
		}

		objectFile, err := os.Open(objectPath) //nolint:gosec // path is sanitized
		if err != nil {
			// FIXME: is the path has invalid characters, still will return ErrNotExists
			if errors.Is(err, os.ErrNotExist) {
				return nil, store.ErrObjectNotFound
			}
			return nil, k6build.NewWrappedError(store.ErrAccessingObject, err)
		}

		return objectFile, nil
	default:
		return nil, fmt.Errorf("%w unsupported schema: %s", store.ErrInvalidURL, url.Scheme)
	}
}

func (f *Store) sanitizePath(path string) (string, error) {
	path = filepath.Clean(path)

	if !filepath.IsAbs(path) || !strings.HasPrefix(path, f.dir) {
		return "", fmt.Errorf("%w : invalid path %s", store.ErrInvalidURL, path)
	}

	return path, nil
}

// lockObject obtains a mutex used to prevent concurrent builds of the same artifact and
// returns a function that will unlock the mutex associated to the given id in the object store.
// The lock is also removed from the map. Subsequent calls will get another lock on the same
// id but this is safe as the object should already be in the object store and no further
// builds are needed.
func (f *Store) lockObject(id string) func() {
	value, _ := f.mutexes.LoadOrStore(id, &sync.Mutex{})
	mtx, _ := value.(*sync.Mutex)
	mtx.Lock()

	return func() {
		f.mutexes.Delete(id)
		mtx.Unlock()
	}
}
