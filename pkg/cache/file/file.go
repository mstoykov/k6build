// Package file implements a file-backed cache
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
	"github.com/grafana/k6build/pkg/cache"
	"github.com/grafana/k6build/pkg/util"
)

// Cache a Cache backed by a file system
type Cache struct {
	dir     string
	mutexes sync.Map
}

// NewTempFileCache creates a file cache using a temporary file
func NewTempFileCache() (cache.Cache, error) {
	return NewFileCache(filepath.Join(os.TempDir(), "buildcache"))
}

// NewFileCache creates an cached backed by a directory
func NewFileCache(dir string) (cache.Cache, error) {
	err := os.MkdirAll(dir, 0o750)
	if err != nil {
		return nil, k6build.NewWrappedError(cache.ErrInitializingCache, err)
	}

	return &Cache{
		dir: dir,
	}, nil
}

// Store stores the object and returns the metadata
// Fails if the object already exists
func (f *Cache) Store(_ context.Context, id string, content io.Reader) (cache.Object, error) {
	if id == "" {
		return cache.Object{}, fmt.Errorf("%w: id cannot be empty", cache.ErrCreatingObject)
	}

	if strings.Contains(id, "/") {
		return cache.Object{}, fmt.Errorf("%w id cannot contain '/'", cache.ErrCreatingObject)
	}

	// prevent concurrent modification of an object
	unlock := f.lockObject(id)
	defer unlock()

	objectDir := filepath.Join(f.dir, id)

	if _, err := os.Stat(objectDir); !errors.Is(err, os.ErrNotExist) {
		return cache.Object{}, fmt.Errorf("%w: object already exists %q", cache.ErrCreatingObject, id)
	}

	// TODO: check permissions
	err := os.MkdirAll(objectDir, 0o750)
	if err != nil {
		return cache.Object{}, k6build.NewWrappedError(cache.ErrCreatingObject, err)
	}

	objectFile, err := os.Create(filepath.Join(objectDir, "data")) //nolint:gosec
	if err != nil {
		return cache.Object{}, k6build.NewWrappedError(cache.ErrCreatingObject, err)
	}
	defer objectFile.Close() //nolint:errcheck

	// write content to object file and copy to buffer to calculate checksum
	// TODO: optimize memory by copying content in blocks
	buff := bytes.Buffer{}
	_, err = io.Copy(objectFile, io.TeeReader(content, &buff))
	if err != nil {
		return cache.Object{}, k6build.NewWrappedError(cache.ErrCreatingObject, err)
	}

	// calculate checksum
	checksum := fmt.Sprintf("%x", sha256.Sum256(buff.Bytes()))

	// write metadata
	err = os.WriteFile(filepath.Join(objectDir, "checksum"), []byte(checksum), 0o644) //nolint:gosec
	if err != nil {
		return cache.Object{}, k6build.NewWrappedError(cache.ErrCreatingObject, err)
	}

	objectURL, _ := util.URLFromFilePath(objectFile.Name())
	return cache.Object{
		ID:       id,
		Checksum: checksum,
		URL:      objectURL.String(),
	}, nil
}

// Get retrieves an objects if exists in the cache or an error otherwise
func (f *Cache) Get(_ context.Context, id string) (cache.Object, error) {
	objectDir := filepath.Join(f.dir, id)
	_, err := os.Stat(objectDir)

	if errors.Is(err, os.ErrNotExist) {
		return cache.Object{}, fmt.Errorf("%w (%s)", cache.ErrObjectNotFound, id)
	}

	if err != nil {
		return cache.Object{}, k6build.NewWrappedError(cache.ErrAccessingObject, err)
	}

	checksum, err := os.ReadFile(filepath.Join(objectDir, "checksum")) //nolint:gosec
	if err != nil {
		return cache.Object{}, k6build.NewWrappedError(cache.ErrAccessingObject, err)
	}

	objectURL, _ := util.URLFromFilePath(filepath.Join(objectDir, "data"))
	return cache.Object{
		ID:       id,
		Checksum: string(checksum),
		URL:      objectURL.String(),
	}, nil
}

// Download returns the content of the object given its url
func (f *Cache) Download(_ context.Context, object cache.Object) (io.ReadCloser, error) {
	url, err := url.Parse(object.URL)
	if err != nil {
		return nil, k6build.NewWrappedError(cache.ErrAccessingObject, err)
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
				return nil, cache.ErrObjectNotFound
			}
			return nil, k6build.NewWrappedError(cache.ErrAccessingObject, err)
		}

		return objectFile, nil
	default:
		return nil, fmt.Errorf("%w unsupported schema: %s", cache.ErrInvalidURL, url.Scheme)
	}
}

func (f *Cache) sanitizePath(path string) (string, error) {
	path = filepath.Clean(path)

	if !filepath.IsAbs(path) || !strings.HasPrefix(path, f.dir) {
		return "", fmt.Errorf("%w : invalid path %s", cache.ErrInvalidURL, path)
	}

	return path, nil
}

// lockObject obtains a mutex used to prevent concurrent builds of the same artifact and
// returns a function that will unlock the mutex associated to the given id in the cache.
// The lock is also removed from the map. Subsequent calls will get another lock on the same
// id but this is safe as the object should already be in the cache and no further builds are needed.
func (f *Cache) lockObject(id string) func() {
	value, _ := f.mutexes.LoadOrStore(id, &sync.Mutex{})
	mtx, _ := value.(*sync.Mutex)
	mtx.Lock()

	return func() {
		f.mutexes.Delete(id)
		mtx.Unlock()
	}
}
