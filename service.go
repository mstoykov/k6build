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

// Artifact defines a binary that can be downloaded
// TODO: add metadata (e.g. list of dependencies, checksum, date compiled)
type Artifact struct {
	ID string `json:"id,omitempty"`
	// URL to fetch the artifact's binary
	URL string `json:"url,omitempty"`
}

// BuildService defines the interface of a build service
type BuildService interface {
	// Build returns a k6 Artifact given its dependencies and version constrain
	Build(ctx context.Context, platform string, k6Constrains string, deps []Dependency) (Artifact, error)
}

// implements the BuildService interface
type buildsrv struct {
	catalog k6catalog.Catalog
	builder k6foundry.Builder
	cache   Cache
}

func NewBuildService(
	catalog k6catalog.Catalog,
	builder k6foundry.Builder,
	cache Cache,
) BuildService {
	return &buildsrv{
		catalog: catalog,
		builder: builder,
		cache:   cache,
	}
}

// DefaultLocalBuildService creates a Local Build service with default configuration
func DefaultLocalBuildService() (BuildService, error) {
	registry, err := k6catalog.DefaultRegistry()
	if err != nil {
		return nil, fmt.Errorf("creating catalog %w", err)
	}

	catalog := k6catalog.NewCatalog(registry)

	builder, err := k6foundry.NewDefaultNativeBuilder()
	if err != nil {
		return nil, fmt.Errorf("creating builder %w", err)
	}

	cache, err := NewTempFileCache()
	if err != nil {
		return nil, fmt.Errorf("creating temp cache %w", err)
	}
	return &buildsrv{
		catalog: catalog,
		builder: builder,
		cache:   cache,
	}, nil
}

func (b *buildsrv) Build(ctx context.Context, platform string, k6Constrains string, deps []Dependency) (Artifact, error) {
	buildPlatform, err := k6foundry.ParsePlatform(platform)
	if err != nil {
		return Artifact{}, fmt.Errorf("invalid platform %w", err)
	}

	k6Mod, err := b.catalog.Resolve(ctx, k6catalog.Dependency{Name: k6Dep, Constrains: k6Constrains})
	if err != nil {
		return Artifact{}, err
	}

	mods := []k6foundry.Module{}
	for _, d := range deps {
		m, err := b.catalog.Resolve(ctx, k6catalog.Dependency{Name: d.Name, Constrains: d.Constraints})
		if err != nil {
			return Artifact{}, err
		}
		mods = append(mods, k6foundry.Module{Path: m.Path, Version: m.Version})
	}

	artifactBuffer := &bytes.Buffer{}
	err = b.builder.Build(ctx, buildPlatform, k6Mod.Version, mods, []string{}, artifactBuffer)
	if err != nil {
		return Artifact{}, fmt.Errorf("building artifact  %w", err)
	}

	artifactObject, err := b.cache.Store(ctx, artifactBuffer)
	if err != nil {
		return Artifact{}, fmt.Errorf("creating object  %w", err)
	}

	return Artifact{ID: artifactObject.ID, URL: artifactObject.URL}, nil
}
