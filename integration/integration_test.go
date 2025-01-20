//go:build integration
// +build integration

package k6provider

import (
	"context"
	"fmt"
	"net/http/httptest"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/grafana/k6build"
	"github.com/grafana/k6build/pkg/builder"
	"github.com/grafana/k6build/pkg/catalog"
	"github.com/grafana/k6build/pkg/client"
	"github.com/grafana/k6build/pkg/server"
	strclient "github.com/grafana/k6build/pkg/store/client"
	filestore "github.com/grafana/k6build/pkg/store/file"
	storesrv "github.com/grafana/k6build/pkg/store/server"
)

func Test_BuildServer(t *testing.T) {
	t.Parallel()

	// 1. setup a file store
	store, err := filestore.NewFileStore(filepath.Join(t.TempDir(), "store"))
	if err != nil {
		t.Fatalf("store setup %v", err)
	}
	storeConfig := storesrv.StoreServerConfig{
		Store: store,
	}

	// 2. start an object store server
	storeHandler, err := storesrv.NewStoreServer(storeConfig)
	if err != nil {
		t.Fatalf("store setup %v", err)
	}
	storeSrv := httptest.NewServer(storeHandler)

	// 3. configure a builder
	storeClient, err := strclient.NewStoreClient(strclient.StoreClientConfig{Server: storeSrv.URL})
	if err != nil {
		t.Fatalf("store client setup %v", err)
	}
	catalog, err := catalog.DefaultCatalog()
	if err != nil {
		t.Fatalf("build server setup %v", err)
	}
	buildConfig := builder.Config{
		Opts: builder.Opts{
			GoOpts: builder.GoOpts{
				CopyGoEnv: true,
			},
		},
		Catalog: catalog,
		Store:   storeClient,
	}
	builder, err := builder.New(context.TODO(), buildConfig)
	if err != nil {
		t.Fatalf("setup %v", err)
	}

	// 4. start a builder server
	srvConfig := server.APIServerConfig{
		BuildService: builder,
	}
	buildSrv := httptest.NewServer(server.NewAPIServer(srvConfig))

	// 5. test building k6 with different options
	testCases := []struct {
		title       string
		platform    string
		k6Constrain string
		deps        []k6build.Dependency
	}{
		{
			title:       "build latest k6",
			platform:    fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
			k6Constrain: "*",
		},
	}

	for _, tc := range testCases { //nolint:paralleltest
		t.Run(tc.title, func(t *testing.T) {
			client, err := client.NewBuildServiceClient(
				client.BuildServiceClientConfig{
					URL: buildSrv.URL,
				},
			)
			if err != nil {
				t.Fatalf("client setup %v", err)
			}

			_, err = client.Build(context.TODO(), tc.platform, tc.k6Constrain, tc.deps)
			if err != nil {
				t.Fatalf("building artifact  %v", err)
			}
		})
	}
}
