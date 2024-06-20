package k6build

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

var (
	ErrObjectNotFound    = errors.New("object not found")   //nolint:revive
	ErrAccessingObject   = errors.New("accessing object")   //nolint:revive
	ErrCreatingObject    = errors.New("creating object")    //nolint:revive
	ErrInitializingCache = errors.New("initializing cache") //nolint:revive
)

// Object represents an object stored in the Cache
// TODO: add metadata (e.g creation data, size)
type Object struct {
	ID       string
	Checksum string
	// an url for downloading the object's content
	URL string
}

// Cache defines an interface for storing blobs
type Cache interface {
	// Get retrieves an objects if exists in the cache or an error otherwise
	Get(ctx context.Context, id string) (Object, error)
	// Store stores the object and returns the metadata
	Store(ctx context.Context, id string, content io.Reader) (Object, error)
}

// a Cache backed by a file system
type fileCache struct {
	path string
}

// NewTempFileCache creates a file cache using a temporary file
func NewTempFileCache() (Cache, error) {
	return NewFileCache(filepath.Join(os.TempDir(), "buildcache"))
}

// NewFileCache creates an cached backed by a directory
func NewFileCache(path string) (Cache, error) {
	err := os.MkdirAll(path, 0o750)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInitializingCache, err)
	}

	return &fileCache{
		path: path,
	}, nil
}

// Store stores the object and returns the metadata
func (f *fileCache) Store(_ context.Context, id string, content io.Reader) (Object, error) {
	if id == "" {
		return Object{}, fmt.Errorf("%w id cannot be empty", ErrCreatingObject)
	}

	if strings.Contains(id, "/") {
		return Object{}, fmt.Errorf("%w id cannot contain '/'", ErrCreatingObject)
	}

	objectDir := filepath.Join(f.path, id)
	// TODO: check permissions
	err := os.MkdirAll(objectDir, 0o750)
	if err != nil {
		return Object{}, fmt.Errorf("%w: %w", ErrCreatingObject, err)
	}

	objectFile, err := os.Create(filepath.Join(objectDir, "data")) //nolint:gosec
	if err != nil {
		return Object{}, fmt.Errorf("%w: %w", ErrCreatingObject, err)
	}

	// write content to object file and copy to buffer to calculate checksum
	// TODO: optimize memory by copying content in blocks
	buff := bytes.Buffer{}
	_, err = io.Copy(objectFile, io.TeeReader(content, &buff))
	if err != nil {
		return Object{}, fmt.Errorf("%w: %w", ErrCreatingObject, err)
	}

	// calculate checksum
	checksum := fmt.Sprintf("%x", sha256.Sum256(buff.Bytes()))

	// write metadata
	err = os.WriteFile(filepath.Join(objectDir, "checksum"), []byte(checksum), 0o644) //nolint:gosec
	if err != nil {
		return Object{}, fmt.Errorf("%w: %w", ErrCreatingObject, err)
	}

	return Object{
		ID:       id,
		Checksum: checksum,
		URL:      fmt.Sprintf("file://%s", objectFile.Name()),
	}, nil
}

// Get retrieves an objects if exists in the cache or an error otherwise
func (f *fileCache) Get(_ context.Context, id string) (Object, error) {
	objectDir := filepath.Join(f.path, id)
	_, err := os.Stat(objectDir)

	if errors.Is(err, os.ErrNotExist) {
		return Object{}, fmt.Errorf("%w: %s", ErrObjectNotFound, id)
	}

	if err != nil {
		return Object{}, fmt.Errorf("%w: %w", ErrAccessingObject, err)
	}

	checksum, err := os.ReadFile(filepath.Join(objectDir, "checksum")) //nolint:gosec
	if err != nil {
		return Object{}, fmt.Errorf("%w: %w", ErrAccessingObject, err)
	}

	return Object{
		ID:       id,
		Checksum: string(checksum),
		URL:      fmt.Sprintf("file://%s", filepath.Join(objectDir, "data")),
	}, nil
}
