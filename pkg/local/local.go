// Package local implements a local build service
package local

import (
	"bytes"
	"context"
	"crypto/sha1" //nolint:gosec
	"errors"
	"fmt"
	"os"
	"regexp"
	"sort"
	"sync"

	"github.com/grafana/k6build"
	"github.com/grafana/k6build/pkg/cache"
	"github.com/grafana/k6build/pkg/cache/client"
	"github.com/grafana/k6build/pkg/cache/file"
	"github.com/grafana/k6catalog"
	"github.com/grafana/k6foundry"
)

const (
	k6Dep  = "k6"
	k6Path = "go.k6.io/k6"

	opRe  = `(?<operator>[=|~|>|<|\^|>=|<=|!=]){0,1}(?:\s*)`
	verRe = `(?P<version>[v|V](?:0|[1-9]\d*)\.(?:0|[1-9]\d*)\.(?:0|[1-9]\d*))`
	preRe = `(?:[+|-|])(?P<prerelease>(?:[0-9a-zA-Z-]+(?:\.[0-9a-zA-Z-]+)*))`
)

var (
	ErrAccessingArtifact    = errors.New("accessing artifact")           //nolint:revive
	ErrBuildingArtifact     = errors.New("building artifact")            //nolint:revive
	ErrInitializingBuilder  = errors.New("initializing builder")         //nolint:revive
	ErrInvalidParameters    = errors.New("invalid build parameters")     //nolint:revive
	ErrPrereleaseNotAllowed = errors.New("pre-releases are not allowed") //nolint:revive

	constrainRe = regexp.MustCompile(opRe + verRe + preRe)
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
	// set verbose build mode
	Verbose bool
	// Allow prerelease versions
	AllowPrereleases bool
}

// buildSrv implements the BuildService interface
type localBuildSrv struct {
	allowPrereleases bool
	catalog          k6catalog.Catalog
	builder          k6foundry.Builder
	cache            cache.Cache
	mutexes          sync.Map
}

// NewBuildService creates a local build service using the given configuration
func NewBuildService(ctx context.Context, config BuildServiceConfig) (k6build.BuildService, error) {
	catalog, err := k6catalog.NewCatalog(ctx, config.Catalog)
	if err != nil {
		return nil, k6build.NewWrappedError(ErrInitializingBuilder, err)
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
		return nil, k6build.NewWrappedError(ErrInitializingBuilder, err)
	}

	var cache cache.Cache

	if config.CacheURL != "" {
		cache, err = client.NewCacheClient(
			client.CacheClientConfig{
				Server: config.CacheURL,
			},
		)
		if err != nil {
			return nil, k6build.NewWrappedError(ErrInitializingBuilder, err)
		}
	} else {
		cache, err = file.NewFileCache(config.CacheDir)
		if err != nil {
			return nil, k6build.NewWrappedError(ErrInitializingBuilder, err)
		}
	}

	return &localBuildSrv{
		allowPrereleases: config.AllowPrereleases,
		catalog:          catalog,
		builder:          builder,
		cache:            cache,
	}, nil
}

// DefaultLocalBuildService creates a local build service with default configuration
func DefaultLocalBuildService() (k6build.BuildService, error) {
	catalog, err := k6catalog.DefaultCatalog()
	if err != nil {
		return nil, k6build.NewWrappedError(ErrInitializingBuilder, err)
	}

	builder, err := k6foundry.NewDefaultNativeBuilder()
	if err != nil {
		return nil, k6build.NewWrappedError(ErrInitializingBuilder, err)
	}

	cache, err := file.NewTempFileCache()
	if err != nil {
		return nil, k6build.NewWrappedError(ErrInitializingBuilder, err)
	}

	return &localBuildSrv{
		catalog: catalog,
		builder: builder,
		cache:   cache,
	}, nil
}

func (b *localBuildSrv) Build( //nolint:funlen
	ctx context.Context,
	platform string,
	k6Constrains string,
	deps []k6build.Dependency,
) (k6build.Artifact, error) {
	buildPlatform, err := k6foundry.ParsePlatform(platform)
	if err != nil {
		return k6build.Artifact{}, k6build.NewWrappedError(ErrInvalidParameters, err)
	}

	// sort dependencies to ensure idempotence of build
	sort.Slice(deps, func(i, j int) bool { return deps[i].Name < deps[j].Name })
	resolved := map[string]string{}

	// check if it is a pre-release version of the form v0.0.0-hash
	// if it is a prerelease we don't check with the catalog, but instead we use
	// the prerelease as version when building this module
	// the build process will return the actual version built in the build info
	// and we can check that version with the catalog
	var k6Mod k6catalog.Module
	prerelease, err := isPrerelease(k6Constrains)
	if err != nil {
		return k6build.Artifact{}, err
	}
	if prerelease != "" {
		if !b.allowPrereleases {
			return k6build.Artifact{}, k6build.NewWrappedError(ErrInvalidParameters, ErrPrereleaseNotAllowed)
		}
		k6Mod = k6catalog.Module{Path: k6Path, Version: prerelease}
	} else {
		k6Mod, err = b.catalog.Resolve(ctx, k6catalog.Dependency{Name: k6Dep, Constrains: k6Constrains})
		if err != nil {
			return k6build.Artifact{}, err
		}
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

	unlock := b.lockArtifact(id)
	defer unlock()

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
		return k6build.Artifact{}, k6build.NewWrappedError(ErrAccessingArtifact, err)
	}

	artifactBuffer := &bytes.Buffer{}
	buildInfo, err := b.builder.Build(ctx, buildPlatform, k6Mod.Version, mods, []string{}, artifactBuffer)
	if err != nil {
		return k6build.Artifact{}, k6build.NewWrappedError(ErrAccessingArtifact, err)
	}

	// if this is a prerelease, we must use the actual version built
	// TODO: check this version is supported
	if prerelease != "" {
		resolved[k6Dep] = buildInfo.ModVersions[k6Mod.Path]
	}

	artifactObject, err = b.cache.Store(ctx, id, artifactBuffer)
	if err != nil {
		return k6build.Artifact{}, k6build.NewWrappedError(ErrAccessingArtifact, err)
	}

	return k6build.Artifact{
		ID:           id,
		Checksum:     artifactObject.Checksum,
		URL:          artifactObject.URL,
		Dependencies: resolved,
		Platform:     platform,
	}, nil
}

// lockArtifact obtains a mutex used to prevent concurrent builds of the same artifact and
// returns a function that will unlock the mutex associated to the given id in the cache.
// The lock is also removed from the map. Subsequent calls will get another lock on the same
// id but this is safe as the object should already be in the cache and no further builds are needed.
func (b *localBuildSrv) lockArtifact(id string) func() {
	value, _ := b.mutexes.LoadOrStore(id, &sync.Mutex{})
	mtx, _ := value.(*sync.Mutex)
	mtx.Lock()

	return func() {
		b.mutexes.Delete(id)
		mtx.Unlock()
	}
}

// isPrerelease checks if the constrain references a pre-release version.
// E.g.  v0.1.0+build-effa45f
func isPrerelease(constrain string) (string, error) {
	opInx := constrainRe.SubexpIndex("operator")
	verIdx := constrainRe.SubexpIndex("version")
	preIdx := constrainRe.SubexpIndex("prerelease")
	matches := constrainRe.FindStringSubmatch(constrain)

	if matches == nil {
		return "", nil
	}

	op := matches[opInx]
	ver := matches[verIdx]
	prerelease := matches[preIdx]

	if op != "" && op != "=" {
		return "", k6build.NewWrappedError(
			ErrInvalidParameters,
			fmt.Errorf("only exact match is allowed for pre-release versions"),
		)
	}

	if ver != "v0.0.0" {
		return "", k6build.NewWrappedError(
			ErrInvalidParameters,
			fmt.Errorf("prerelease version must start with v0.0.0"),
		)
	}
	return prerelease, nil
}
