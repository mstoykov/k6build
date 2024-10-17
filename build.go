// Package k6build defines a service for building k6 binaries
package k6build

import (
	"bytes"
	"context"
	"errors"
	"fmt"
)

var ErrBuildFailed = errors.New("build failed") //nolint:revive

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
	Version string `json:"version,omitempty"`
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
	return a.toString(true, " ")
}

// Print returns a string with a pretty print of the artifact
func (a Artifact) Print() string {
	return a.toString(true, "\n")
}

// PrintSummary returns a string with a pretty print of the artifact
func (a Artifact) PrintSummary() string {
	return a.toString(false, "\n")
}

// Print returns a text serialization of the Artifact
func (a Artifact) toString(details bool, sep string) string {
	buffer := &bytes.Buffer{}
	if details {
		buffer.WriteString(fmt.Sprintf("id: %s%s", a.ID, sep))
	}
	buffer.WriteString(fmt.Sprintf("platform: %s%s", a.Platform, sep))
	for dep, version := range a.Dependencies {
		buffer.WriteString(fmt.Sprintf("%s:%q%s", dep, version, sep))
	}
	buffer.WriteString(fmt.Sprintf("checksum: %s%s", a.Checksum, sep))
	if details {
		buffer.WriteString(fmt.Sprintf("url: %s%s", a.URL, sep))
	}
	return buffer.String()
}

// BuildService defines the interface of a build service
type BuildService interface {
	// Build returns a k6 Artifact given its dependencies and version constrain
	Build(ctx context.Context, platform string, k6Constrains string, deps []Dependency) (Artifact, error)
}
