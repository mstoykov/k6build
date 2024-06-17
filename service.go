// Package k6build defines a service for building k8 binaries
package k6build

import (
	"bytes"
	"context"
	"crypto/sha1" //nolint:gosec
	"errors"
	"fmt"
	"sort"

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

// NewBuildService creates a build service
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
	catalog, err := k6catalog.DefaultCatalog()
	if err != nil {
		return nil, fmt.Errorf("creating catalog %w", err)
	}

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

func (b *buildsrv) Build(
	ctx context.Context,
	platform string,
	k6Constrains string,
	deps []Dependency,
) (Artifact, error) {
	buildPlatform, err := k6foundry.ParsePlatform(platform)
	if err != nil {
		return Artifact{}, fmt.Errorf("invalid platform %w", err)
	}

	// sort dependencies to ensure idempotence of build
	sort.Slice(deps, func(i, j int) bool { return deps[i].Name < deps[j].Name })
	resolved := map[string]string{}

	k6Mod, err := b.catalog.Resolve(ctx, k6catalog.Dependency{Name: k6Dep, Constrains: k6Constrains})
	if err != nil {
		return Artifact{}, err
	}
	resolved[k6Dep] = k6Mod.Version

	mods := []k6foundry.Module{}
	for _, d := range deps {
		m, modErr := b.catalog.Resolve(ctx, k6catalog.Dependency{Name: d.Name, Constrains: d.Constraints})
		if modErr != nil {
			return Artifact{}, modErr
		}
		mods = append(mods, k6foundry.Module{Path: m.Path, Version: m.Version})
		resolved[d.Name] = m.Version
	}

	// generate id form sorted list of dependencies
	hashData := bytes.Buffer{}
	hashData.WriteString(platform)
	hashData.WriteString(fmt.Sprintf(":k6%s", k6Mod.Version))
	for _, d := range deps {
		hashData.WriteString(fmt.Sprintf(":%s%s", d, resolved[d.Name]))
	}
	id := fmt.Sprintf("%x", sha1.Sum(hashData.Bytes())) //nolint:gosec

	artifactObject, err := b.cache.Get(ctx, id)
	if err == nil {
		return Artifact{
			ID:           id,
			Checksum:     artifactObject.Checksum,
			URL:          artifactObject.URL,
			Dependencies: resolved,
			Platform:     platform,
		}, nil
	}

	if !errors.Is(err, ErrObjectNotFound) {
		return Artifact{}, fmt.Errorf("accessing artifact %w", err)
	}

	artifactBuffer := &bytes.Buffer{}
	err = b.builder.Build(ctx, buildPlatform, k6Mod.Version, mods, []string{}, artifactBuffer)
	if err != nil {
		return Artifact{}, fmt.Errorf("building artifact  %w", err)
	}

	artifactObject, err = b.cache.Store(ctx, id, artifactBuffer)
	if err != nil {
		return Artifact{}, fmt.Errorf("creating object  %w", err)
	}

	return Artifact{
		ID:           id,
		Checksum:     artifactObject.Checksum,
		URL:          artifactObject.URL,
		Dependencies: resolved,
		Platform:     platform,
	}, nil
}
