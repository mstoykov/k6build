package k6build

import (
	"context"
	"errors"
	"testing"

	"github.com/grafana/k6catalog"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func dependencyComp(a, b Module) bool { return a.Path < b.Path }

func TestDependencyResolution(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		title     string
		k6        string
		deps      []Dependency
		expectErr error
		expect    Artifact
	}{
		{
			title:     "build k6 v0.1.0 ",
			k6:        "v0.1.0",
			deps:      []Dependency{},
			expectErr: nil,
			expect: Artifact{
				Dependencies: map[string]string{"k6": "v0.1.0"},
			},
		},
		{
			title:     "build k6 >v0.1.0",
			k6:        ">v0.1.0",
			deps:      []Dependency{},
			expectErr: nil,
			expect: Artifact{
				Dependencies: map[string]string{"k6": "v0.2.0"},
			},
		},
		{
			title:     "build unsatisfied k6 constrain (>v0.2.0)",
			k6:        ">v0.2.0",
			deps:      []Dependency{},
			expectErr: k6catalog.ErrCannotSatisfy,
		},
		{
			title:     "build k6 v0.1.0 exact dependency constraint",
			k6:        "v0.1.0",
			deps:      []Dependency{{Name: "k6/x/ext", Constraints: "v0.1.0"}},
			expectErr: nil,
			expect: Artifact{
				Dependencies: map[string]string{
					"k6":       "v0.1.0",
					"k6/x/ext": "v0.1.0",
				},
			},
		},
		{
			title:     "build k6 v0.1.0 unsatisfied dependency constrain",
			k6:        "v0.1.0",
			deps:      []Dependency{{Name: "k6/x/ext", Constraints: ">v0.2.0"}},
			expectErr: k6catalog.ErrCannotSatisfy,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.title, func(t *testing.T) {
			t.Parallel()

			buildsrv, err := SetupTestLocalBuildService(
				LocalBuildServiceConfig{
					CacheDir: t.TempDir(),
					Catalog:  "testdata/catalog.json",
				},
			)
			if err != nil {
				t.Fatalf("test setup %v", err)
			}

			artifact, err := buildsrv.Build(
				context.TODO(),
				"linux/amd64",
				tc.k6,
				tc.deps,
			)

			if !errors.Is(err, tc.expectErr) {
				t.Fatalf("unexpected error wanted %v got %v", tc.expectErr, err)
			}

			// don't check artifact if error is expected
			if tc.expectErr != nil {
				return
			}

			diff := cmp.Diff(tc.expect.Dependencies, artifact.Dependencies, cmpopts.SortSlices(dependencyComp))
			if diff != "" {
				t.Fatalf("dependencies don't match: %s\n", diff)
			}
		})
	}
}

func TestIdempotentBuild(t *testing.T) {
	t.Parallel()
	buildsrv, err := SetupTestLocalBuildService(
		LocalBuildServiceConfig{
			CacheDir: t.TempDir(),
			Catalog:  "testdata/catalog.json",
		},
	)
	if err != nil {
		t.Fatalf("test setup %v", err)
	}

	artifact, err := buildsrv.Build(
		context.TODO(),
		"linux/amd64",
		"v0.1.0",
		[]Dependency{
			{Name: "k6/x/ext", Constraints: "v0.1.0"},
			{Name: "k6/x/ext2", Constraints: "v0.1.0"},
		},
	)
	if err != nil {
		t.Fatalf("test setup %v", err)
	}

	t.Run("should rebuild same artifact", func(t *testing.T) {
		t.Parallel()

		testCases := []struct {
			title    string
			platform string
			k6       string
			deps     []Dependency
		}{
			{
				title:    "same dependencies",
				platform: "linux/amd64",
				k6:       "v0.1.0",
				deps: []Dependency{
					{Name: "k6/x/ext", Constraints: "v0.1.0"},
					{Name: "k6/x/ext2", Constraints: "v0.1.0"},
				},
			},
			{
				title:    "different order of dependencies",
				platform: "linux/amd64",
				k6:       "v0.1.0",
				deps: []Dependency{
					{Name: "k6/x/ext2", Constraints: "v0.1.0"},
					{Name: "k6/x/ext", Constraints: "v0.1.0"},
				},
			},
		}

		for _, tc := range testCases {
			t.Run(tc.title, func(t *testing.T) {
				t.Parallel()

				rebuild, err := buildsrv.Build(
					context.TODO(),
					tc.platform,
					tc.k6,
					tc.deps,
				)
				if err != nil {
					t.Fatalf("unexpected %v", err)
				}

				if artifact.ID != rebuild.ID {
					t.Fatalf("artifact ID don't match")
				}

				diff := cmp.Diff(artifact.Dependencies, rebuild.Dependencies, cmpopts.SortSlices(dependencyComp))
				if diff != "" {
					t.Fatalf("dependencies don't match: %s\n", diff)
				}
			})
		}
	})

	t.Run("should build a different artifact", func(t *testing.T) {
		t.Parallel()

		testCases := []struct {
			title    string
			platform string
			k6       string
			deps     []Dependency
		}{
			{
				title:    "different k6 versions",
				platform: "linux/amd64",
				k6:       "v0.2.0",
				deps: []Dependency{
					{Name: "k6/x/ext", Constraints: "v0.1.0"},
					{Name: "k6/x/ext2", Constraints: "v0.1.0"},
				},
			},
			{
				title:    "different dependency versions",
				platform: "linux/amd64",
				k6:       "v0.1.0",
				deps: []Dependency{
					{Name: "k6/x/ext", Constraints: "v0.2.0"},
					{Name: "k6/x/ext2", Constraints: "v0.1.0"},
				},
			},
			{
				title:    "different dependencies",
				platform: "linux/amd64",
				k6:       "v0.1.0",
				deps: []Dependency{
					{Name: "k6/x/ext", Constraints: "v0.1.0"},
				},
			},
			{
				title:    "different platform",
				platform: "linux/arm64",
				k6:       "v0.1.0",
				deps: []Dependency{
					{Name: "k6/x/ext", Constraints: "v0.1.0"},
					{Name: "k6/x/ext2", Constraints: "v0.1.0"},
				},
			},
		}

		for _, tc := range testCases {
			t.Run(tc.title, func(t *testing.T) {
				t.Parallel()

				rebuild, err := buildsrv.Build(
					context.TODO(),
					tc.platform,
					tc.k6,
					tc.deps,
				)
				if err != nil {
					t.Fatalf("unexpected %v", err)
				}

				if artifact.ID == rebuild.ID {
					t.Fatalf("should had built a different artifact")
				}
			})
		}
	})
}
