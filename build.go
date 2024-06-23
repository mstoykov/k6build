// Package k6build defines a service for building k8 binaries
package k6build

import (
	"bytes"
	"context"
	"fmt"

	"github.com/grafana/k6catalog"
	"github.com/grafana/k6foundry"
)

const (
	k6Dep = "k6"
)

// Dependency contains the properties of a k6 dependency.
type Dependency struct {
	// Name is the name of the dependency.
	Name string `json:"name,omitempty"`
	// Constraints contains the version constraints of the dependency.
	Constraints string `json:"constraints,omitempty"`
}

// Module defines an artifact dependency
type Module struct {
	Path    string `json:"path,omitempty"`
	Version string `json:"vesion,omitempty"`
}

// Artifact defines a binary that can be downloaded
// TODO: add metadata (e.g. list of dependencies, checksum, date compiled)
type Artifact struct {
	ID string `json:"id,omitempty"`
	// URL to fetch the artifact's binary
	URL string `json:"url,omitempty"`
	// list of dependencies
	Dependencies map[string]string `json:"dependencies,omitempty"`
	// platform
	Platform string `json:"platform,omitempty"`
	// binary checksum (sha256)
	Checksum string `json:"checksum,omitempty"`
}

// String returns a text serialization of the Artifact
func (a Artifact) String() string {
	buffer := &bytes.Buffer{}
	buffer.WriteString(fmt.Sprintf(" id: %s", a.ID))
	buffer.WriteString(fmt.Sprintf("platform: %s", a.Platform))
	for dep, version := range a.Dependencies {
		buffer.WriteString(fmt.Sprintf(" %s:%q", dep, version))
	}
	buffer.WriteString(fmt.Sprintf(" checksum: %s", a.Checksum))
	buffer.WriteString(fmt.Sprintf(" url: %s", a.URL))
	return buffer.String()
}

// BuildService defines the interface of a build service
type BuildService interface {
	// Build returns a k6 Artifact given its dependencies and version constrain
	Build(ctx context.Context, platform string, k6Constrains string, deps []Dependency) (Artifact, error)
}

// implements the BuildService interface
type localBuildSrv struct {
	catalog k6catalog.Catalog
	builder k6foundry.Builder
	cache   Cache
}

// NewBuildService creates a build service
func NewBuildService(
	catalog k6catalog.Catalog,
	builder k6foundry.Builder,
	cache Cache,
) BuildService {
	return &localBuildSrv{
		catalog: catalog,
		builder: builder,
		cache:   cache,
	}
}
