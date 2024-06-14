package k6build

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
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
	// Store stores the object and returns the metadata
	Store(ctx context.Context, content io.Reader) (Object, error)
}

// an ObjectStore backed by a file
type fileObjectStore struct {
	path string
}

func NewTempFileCache() (Cache, error) {
	cacheDir, err := os.MkdirTemp(os.TempDir(), "buildcache*")
	if err != nil {
		return nil, fmt.Errorf("creating cache directory %w", err)
	}

	return NewFileCache(cacheDir)
}

// NewFileCache creates an cached backed by a directory
func NewFileCache(path string) (Cache, error) {
	fileInfo, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("invalid file path: %w", err)
	}

	if !fileInfo.IsDir() {
		return nil, fmt.Errorf("must be a directory: %s", path)
	}

	return &fileObjectStore{
		path: path,
	}, nil
}

func (f *fileObjectStore) Store(ctx context.Context, content io.Reader) (Object, error) {
	objectFile, err := os.CreateTemp(f.path, "*")
	if err != nil {
		return Object{}, fmt.Errorf("creating object %w", err)
	}

	// write content to object file and copy to buffer to calculate checksum
	// TODO: optimize memory by copying content in blocks
	buff := bytes.Buffer{}
	_, err = io.Copy(objectFile, io.TeeReader(content, &buff))
	if err != nil {
		return Object{}, fmt.Errorf("creating object %w", err)
	}

	// calculate checksum
	checksum := sha256.New()
	checksum.Sum(buff.Bytes())

	return Object{
		ID:       filepath.Base(objectFile.Name()),
		Checksum: string(fmt.Sprintf("%x", checksum.Sum(nil))),
		URL:      fmt.Sprintf("file://%s", objectFile.Name()),
	}, nil
}
