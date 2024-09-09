// Package local implements a local build service
package local

import (
	"bytes"
	"context"
	"crypto/sha1" //nolint:gosec
	"errors"
	"fmt"
	"os"
	"sort"

	"github.com/grafana/k6build"
	"github.com/grafana/k6build/pkg/cache"
	"github.com/grafana/k6build/pkg/cache/client"
	"github.com/grafana/k6build/pkg/cache/file"
	"github.com/grafana/k6catalog"
	"github.com/grafana/k6foundry"
)

const (
	k6Dep = "k6"
)

// BuildServiceConfig defines the configuration for a Local build service
type BuildServiceConfig struct {
	// Set build environment variables
	// Can be used for setting (or overriding, if CopyGoEnv is true) go environment variables
	BuildEnv map[string]string
	// path to catalog's json file. Can be a file path or a URL
	Catalog string
	// url to remote cache service
	CacheURL string
	// path to cache dir
	CacheDir string
	// Copy go environment. BuildEnv can override the variables copied from go environment.
	CopyGoEnv bool
	// ser verbose build mode
	Verbose bool
}

// buildSrv implements the BuildService interface
type localBuildSrv struct {
	catalog k6catalog.Catalog
	builder k6foundry.Builder
	cache   cache.Cache
}

// NewBuildService creates a local build service using the given configuration
func NewBuildService(ctx context.Context, config BuildServiceConfig) (k6build.BuildService, error) {
	catalog, err := k6catalog.NewCatalog(ctx, config.Catalog)
	if err != nil {
		return nil, fmt.Errorf("getting catalog %w", err)
	}

	builderOpts := k6foundry.NativeBuilderOpts{
		GoOpts: k6foundry.GoOpts{
			Env:       config.BuildEnv,
			CopyGoEnv: config.CopyGoEnv,
		},
	}
	if config.Verbose {
		builderOpts.Stdout = os.Stdout
		builderOpts.Stderr = os.Stderr
	}

	builder, err := k6foundry.NewNativeBuilder(ctx, builderOpts)
	if err != nil {
		return nil, fmt.Errorf("creating builder %w", err)
	}

	var cache cache.Cache

	if config.CacheURL != "" {
		cache, err = client.NewCacheClient(
			client.CacheClientConfig{
				Server: config.CacheURL,
			},
		)
		if err != nil {
			return nil, fmt.Errorf("creating cache client %w", err)
		}
	} else {
		cache, err = file.NewFileCache(config.CacheDir)
		if err != nil {
			return nil, fmt.Errorf("creating cache %w", err)
		}
	}

	return &localBuildSrv{
		catalog: catalog,
		builder: builder,
		cache:   cache,
	}, nil
}

// DefaultLocalBuildService creates a local build service with default configuration
func DefaultLocalBuildService() (k6build.BuildService, error) {
	catalog, err := k6catalog.DefaultCatalog()
	if err != nil {
		return nil, fmt.Errorf("creating catalog %w", err)
	}

	builder, err := k6foundry.NewDefaultNativeBuilder()
	if err != nil {
		return nil, fmt.Errorf("creating builder %w", err)
	}

	cache, err := file.NewTempFileCache()
	if err != nil {
		return nil, fmt.Errorf("creating temp cache %w", err)
	}

	return &localBuildSrv{
		catalog: catalog,
		builder: builder,
		cache:   cache,
	}, nil
}

func (b *localBuildSrv) Build(
	ctx context.Context,
	platform string,
	k6Constrains string,
	deps []k6build.Dependency,
) (k6build.Artifact, error) {
	buildPlatform, err := k6foundry.ParsePlatform(platform)
	if err != nil {
		return k6build.Artifact{}, fmt.Errorf("invalid platform %w", err)
	}

	// sort dependencies to ensure idempotence of build
	sort.Slice(deps, func(i, j int) bool { return deps[i].Name < deps[j].Name })
	resolved := map[string]string{}

	k6Mod, err := b.catalog.Resolve(ctx, k6catalog.Dependency{Name: k6Dep, Constrains: k6Constrains})
	if err != nil {
		return k6build.Artifact{}, err
	}
	resolved[k6Dep] = k6Mod.Version

	mods := []k6foundry.Module{}
	for _, d := range deps {
		m, modErr := b.catalog.Resolve(ctx, k6catalog.Dependency{Name: d.Name, Constrains: d.Constraints})
		if modErr != nil {
			return k6build.Artifact{}, modErr
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
		return k6build.Artifact{
			ID:           id,
			Checksum:     artifactObject.Checksum,
			URL:          artifactObject.URL,
			Dependencies: resolved,
			Platform:     platform,
		}, nil
	}

	if !errors.Is(err, cache.ErrObjectNotFound) {
		return k6build.Artifact{}, fmt.Errorf("accessing artifact %w", err)
	}

	artifactBuffer := &bytes.Buffer{}
	err = b.builder.Build(ctx, buildPlatform, k6Mod.Version, mods, []string{}, artifactBuffer)
	if err != nil {
		return k6build.Artifact{}, fmt.Errorf("building artifact  %w", err)
	}

	artifactObject, err = b.cache.Store(ctx, id, artifactBuffer)
	if err != nil {
		return k6build.Artifact{}, fmt.Errorf("creating object  %w", err)
	}

	return k6build.Artifact{
		ID:           id,
		Checksum:     artifactObject.Checksum,
		URL:          artifactObject.URL,
		Dependencies: resolved,
		Platform:     platform,
	}, nil
}
