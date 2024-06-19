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

// LocalBuildServiceConfig defines the configuration for a Local build service
type LocalBuildServiceConfig struct {
	// Set build environment variables
	// Can be used for setting (or overriding, if CopyGoEnv is true) go environment variables
	BuildEnv map[string]string
	// path to catalog's json file
	Catalog string
	// path to cache dir
	CacheDir string
	// Copy go environment. BuildEnv can override the variables copied from go environment.
	CopyGoEnv bool
	// ser verbose build mode
	Verbose bool
}

// NewLocalBuildService creates a local build service using the given configuration
func NewLocalBuildService(ctx context.Context, config LocalBuildServiceConfig) (BuildService, error) {
	catalog, err := k6catalog.NewCatalogFromJSON(config.Catalog)
	if err != nil {
		return nil, fmt.Errorf("creating catalog %w", err)
	}

	builderOpts := k6foundry.NativeBuilderOpts{
		Verbose: config.Verbose,
		GoOpts: k6foundry.GoOpts{
			Env:       config.BuildEnv,
			CopyGoEnv: config.CopyGoEnv,
		},
	}
	builder, err := k6foundry.NewNativeBuilder(ctx, builderOpts)
	if err != nil {
		return nil, fmt.Errorf("creating builder %w", err)
	}

	cache, err := NewFileCache(config.CacheDir)
	if err != nil {
		return nil, fmt.Errorf("creating cache %w", err)
	}

	return &buildsrv{
		catalog: catalog,
		builder: builder,
		cache:   cache,
	}, nil
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
