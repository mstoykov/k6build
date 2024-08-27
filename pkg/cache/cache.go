// Package cache defines the interface of a cache service
package cache

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
)

var (
	ErrAccessingObject   = errors.New("accessing object")   //nolint:revive
	ErrCreatingObject    = errors.New("creating object")    //nolint:revive
	ErrInitializingCache = errors.New("initializing cache") //nolint:revive
	ErrObjectNotFound    = errors.New("object not found")   //nolint:revive
	ErrInvalidURL        = errors.New("invalid object URL") //nolint:revive

)

// Object represents an object stored in the Cache
// TODO: add metadata (e.g creation data, size)
type Object struct {
	ID       string
	Checksum string
	// an url for downloading the object's content
	URL string
}

func (o Object) String() string {
	buffer := &bytes.Buffer{}
	buffer.WriteString(fmt.Sprintf("id: %s", o.ID))
	buffer.WriteString(fmt.Sprintf(" checksum: %s", o.Checksum))
	buffer.WriteString(fmt.Sprintf("url: %s", o.URL))

	return buffer.String()
}

// Cache defines an interface for storing blobs
type Cache interface {
	// Get retrieves an objects if exists in the cache or an error otherwise
	Get(ctx context.Context, id string) (Object, error)
	// Store stores the object and returns the metadata
	Store(ctx context.Context, id string, content io.Reader) (Object, error)
	// Download returns the content of the object
	Download(ctx context.Context, object Object) (io.ReadCloser, error)
}
