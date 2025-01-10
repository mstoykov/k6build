package local

import (
	"context"
	"fmt"
	"net/http/httptest"
	"testing"

	"github.com/grafana/k6build"
	"github.com/grafana/k6build/pkg/builder"
	"github.com/grafana/k6build/pkg/catalog"
	"github.com/grafana/k6build/pkg/store/file"
	"github.com/grafana/k6foundry"
	"github.com/grafana/k6foundry/pkg/testutils/goproxy"
)

// DependencyComp compares two dependencies for ordering
func DependencyComp(a, b catalog.Module) bool { return a.Path < b.Path }

// SetupTestLocalBuildService setups a local build service for testing
func SetupTestLocalBuildService(t *testing.T) (k6build.BuildService, error) {
	modules := []struct {
		path    string
		version string
		source  string
	}{
		{
			path:    "go.k6.io/k6",
			version: "v0.1.0",
			source:  "testdata/deps/k6",
		},
		{
			path:    "go.k6.io/k6",
			version: "v0.2.0",
			source:  "testdata/deps/k6",
		},
		{
			path:    "go.k6.io/k6ext",
			version: "v0.1.0",
			source:  "testdata/deps/k6ext",
		},
		{
			path:    "go.k6.io/k6ext",
			version: "v0.2.0",
			source:  "testdata/deps/k6ext",
		},
		{
			path:    "go.k6.io/k6ext2",
			version: "v0.1.0",
			source:  "testdata/deps/k6ext2",
		},
	}

	// creates a goproxy that serves the given modules
	proxy := goproxy.NewGoProxy()
	for _, m := range modules {
		err := proxy.AddModVersion(m.path, m.version, m.source)
		if err != nil {
			return nil, fmt.Errorf("setup %w", err)
		}
	}

	goproxySrv := httptest.NewServer(proxy)

	catalog, err := catalog.NewCatalogFromFile("testdata/catalog.json")
	if err != nil {
		return nil, fmt.Errorf("setting up test builder %w", err)
	}

	store, err := file.NewFileStore(t.TempDir())
	if err != nil {
		return nil, fmt.Errorf("creating temporary object store %w", err)
	}

	return builder.New(context.Background(), builder.Config{
		Opts: builder.Opts{
			GoOpts: k6foundry.GoOpts{
				CopyGoEnv: true,
				Env: map[string]string{
					"GOPROXY":   goproxySrv.URL,
					"GONOPROXY": "none",
					"GOPRIVATE": "go.k6.io",
					"GONOSUMDB": "go.k6.io",
				},
				TmpCache: true,
			},
		},
		Catalog: catalog,
		Store:   store,
	})
}
