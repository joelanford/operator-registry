package action_test

import (
	"context"
	"embed"
	"io/fs"
	"testing"

	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/util/yaml"

	action2 "github.com/operator-framework/operator-registry/alpha/action"
	declcfg2 "github.com/operator-framework/operator-registry/alpha/declcfg"
	property2 "github.com/operator-framework/operator-registry/alpha/property"
	"github.com/operator-framework/operator-registry/pkg/containertools"
	"github.com/operator-framework/operator-registry/pkg/image"
	"github.com/operator-framework/operator-registry/pkg/lib/bundle"
)

func TestRender(t *testing.T) {
	type spec struct {
		name      string
		render    action2.Render
		expectCfg *declcfg2.DeclarativeConfig
		assertion require.ErrorAssertionFunc
	}

	registry, err := newRegistry()
	require.NoError(t, err)
	foov1csv, err := bundleImageV1.ReadFile("testdata/foo-bundle-v0.1.0/manifests/foo.v0.1.0.csv.yaml")
	require.NoError(t, err)
	foov1crd, err := bundleImageV1.ReadFile("testdata/foo-bundle-v0.1.0/manifests/foos.test.foo.crd.yaml")
	require.NoError(t, err)
	foov2csv, err := bundleImageV2.ReadFile("testdata/foo-bundle-v0.2.0/manifests/foo.v0.2.0.csv.yaml")
	require.NoError(t, err)
	foov2crd, err := bundleImageV2.ReadFile("testdata/foo-bundle-v0.2.0/manifests/foos.test.foo.crd.yaml")
	require.NoError(t, err)

	foov1csv, err = yaml.ToJSON(foov1csv)
	require.NoError(t, err)
	foov1crd, err = yaml.ToJSON(foov1crd)
	require.NoError(t, err)
	foov2csv, err = yaml.ToJSON(foov2csv)
	require.NoError(t, err)
	foov2crd, err = yaml.ToJSON(foov2crd)
	require.NoError(t, err)

	specs := []spec{
		{
			name: "Success/SqliteIndexImage",
			render: action2.Render{
				Refs:     []string{"test.registry/foo-operator/foo-index-sqlite:v0.2.0"},
				Registry: registry,
			},
			expectCfg: &declcfg2.DeclarativeConfig{
				Packages: []declcfg2.Package{
					{
						Schema:         "olm.package",
						Name:           "foo",
						DefaultChannel: "beta",
					},
				},
				Bundles: []declcfg2.Bundle{
					{
						Schema:  "olm.bundle",
						Name:    "foo.v0.1.0",
						Package: "foo",
						Image:   "test.registry/foo-operator/foo-bundle:v0.1.0",
						Properties: []property2.Property{
							property2.MustBuildChannel("beta", ""),
							property2.MustBuildGVK("test.foo", "v1", "Foo"),
							property2.MustBuildGVKRequired("test.bar", "v1alpha1", "Bar"),
							property2.MustBuildPackage("foo", "0.1.0"),
							property2.MustBuildPackageRequired("bar", "v0.1.0"),
							property2.MustBuildSkipRange("<0.1.0"),
							property2.MustBuildBundleObjectData(foov1csv),
							property2.MustBuildBundleObjectData(foov1crd),
						},
						RelatedImages: []declcfg2.RelatedImage{
							{
								Name:  "operator",
								Image: "test.registry/foo-operator/foo:v0.1.0",
							},
						},
						CsvJSON: string(foov1csv),
						Objects: []string{string(foov1csv), string(foov1crd)},
					},
					{
						Schema:  "olm.bundle",
						Name:    "foo.v0.2.0",
						Package: "foo",
						Image:   "test.registry/foo-operator/foo-bundle:v0.2.0",
						Properties: []property2.Property{
							property2.MustBuildChannel("beta", "foo.v0.1.0"),
							property2.MustBuildGVK("test.foo", "v1", "Foo"),
							property2.MustBuildGVKRequired("test.bar", "v1alpha1", "Bar"),
							property2.MustBuildPackage("foo", "0.2.0"),
							property2.MustBuildPackageRequired("bar", "v0.1.0"),
							property2.MustBuildSkipRange("<0.2.0"),
							property2.MustBuildSkips("foo.v0.1.1"),
							property2.MustBuildSkips("foo.v0.1.2"),
							property2.MustBuildBundleObjectData(foov2csv),
							property2.MustBuildBundleObjectData(foov2crd),
						},
						RelatedImages: []declcfg2.RelatedImage{
							{
								Name:  "operator",
								Image: "test.registry/foo-operator/foo:v0.2.0",
							},
						},
						CsvJSON: string(foov2csv),
						Objects: []string{string(foov2csv), string(foov2crd)},
					},
				},
			},
			assertion: require.NoError,
		},
		{
			name: "Success/DeclcfgIndexImage",
			render: action2.Render{
				Refs:     []string{"test.registry/foo-operator/foo-index-declcfg:v0.2.0"},
				Registry: registry,
			},
			expectCfg: &declcfg2.DeclarativeConfig{
				Packages: []declcfg2.Package{
					{
						Schema:         "olm.package",
						Name:           "foo",
						DefaultChannel: "beta",
					},
				},
				Bundles: []declcfg2.Bundle{
					{
						Schema:  "olm.bundle",
						Name:    "foo.v0.1.0",
						Package: "foo",
						Image:   "test.registry/foo-operator/foo-bundle:v0.1.0",
						Properties: []property2.Property{
							property2.MustBuildChannel("beta", ""),
							property2.MustBuildGVK("test.foo", "v1", "Foo"),
							property2.MustBuildGVKRequired("test.bar", "v1alpha1", "Bar"),
							property2.MustBuildPackage("foo", "0.1.0"),
							property2.MustBuildPackageRequired("bar", "v0.1.0"),
							property2.MustBuildSkipRange("<0.1.0"),
							property2.MustBuildBundleObjectData(foov1csv),
							property2.MustBuildBundleObjectData(foov1crd),
						},
						RelatedImages: []declcfg2.RelatedImage{
							{
								Name:  "operator",
								Image: "test.registry/foo-operator/foo:v0.1.0",
							},
						},
						CsvJSON: string(foov1csv),
						Objects: []string{string(foov1csv), string(foov1crd)},
					},
					{
						Schema:  "olm.bundle",
						Name:    "foo.v0.2.0",
						Package: "foo",
						Image:   "test.registry/foo-operator/foo-bundle:v0.2.0",
						Properties: []property2.Property{
							property2.MustBuildChannel("beta", "foo.v0.1.0"),
							property2.MustBuildGVK("test.foo", "v1", "Foo"),
							property2.MustBuildGVKRequired("test.bar", "v1alpha1", "Bar"),
							property2.MustBuildPackage("foo", "0.2.0"),
							property2.MustBuildPackageRequired("bar", "v0.1.0"),
							property2.MustBuildSkipRange("<0.2.0"),
							property2.MustBuildSkips("foo.v0.1.1"),
							property2.MustBuildSkips("foo.v0.1.2"),
							property2.MustBuildBundleObjectData(foov2csv),
							property2.MustBuildBundleObjectData(foov2crd),
						},
						RelatedImages: []declcfg2.RelatedImage{
							{
								Name:  "operator",
								Image: "test.registry/foo-operator/foo:v0.2.0",
							},
						},
						CsvJSON: string(foov2csv),
						Objects: []string{string(foov2csv), string(foov2crd)},
					},
				},
			},
			assertion: require.NoError,
		},
		{
			name: "Success/BundleImage",
			render: action2.Render{
				Refs:     []string{"test.registry/foo-operator/foo-bundle:v0.2.0"},
				Registry: registry,
			},
			expectCfg: &declcfg2.DeclarativeConfig{
				Bundles: []declcfg2.Bundle{
					{
						Schema:  "olm.bundle",
						Name:    "foo.v0.2.0",
						Package: "foo",
						Image:   "test.registry/foo-operator/foo-bundle:v0.2.0",
						Properties: []property2.Property{
							property2.MustBuildChannel("beta", "foo.v0.1.0"),
							property2.MustBuildGVK("test.foo", "v1", "Foo"),
							property2.MustBuildGVKRequired("test.bar", "v1alpha1", "Bar"),
							property2.MustBuildPackage("foo", "0.2.0"),
							property2.MustBuildPackageRequired("bar", "v0.1.0"),
							property2.MustBuildSkipRange("<0.2.0"),
							property2.MustBuildSkips("foo.v0.1.1"),
							property2.MustBuildSkips("foo.v0.1.2"),
						},
						RelatedImages: []declcfg2.RelatedImage{
							{
								Name:  "operator",
								Image: "test.registry/foo-operator/foo:v0.2.0",
							},
						},
					},
				},
			},
			assertion: require.NoError,
		},
	}

	for _, s := range specs {
		t.Run(s.name, func(t *testing.T) {
			actualCfg, actualErr := s.render.Run(context.Background())
			s.assertion(t, actualErr)
			require.Equal(t, s.expectCfg, actualCfg)
		})
	}
}

//go:embed testdata/foo-bundle-v0.1.0/manifests/*
//go:embed testdata/foo-bundle-v0.1.0/metadata/*
var bundleImageV1 embed.FS

//go:embed testdata/foo-bundle-v0.2.0/manifests/*
//go:embed testdata/foo-bundle-v0.2.0/metadata/*
var bundleImageV2 embed.FS

//go:embed testdata/foo-index-v0.2.0-sqlite/database/*
var sqliteImage embed.FS

//go:embed testdata/foo-index-v0.2.0-declcfg/foo/*
var declcfgImage embed.FS

func newRegistry() (image.Registry, error) {
	subSqliteImage, err := fs.Sub(sqliteImage, "testdata/foo-index-v0.2.0-sqlite")
	if err != nil {
		return nil, err
	}
	subDeclcfgImage, err := fs.Sub(declcfgImage, "testdata/foo-index-v0.2.0-declcfg")
	if err != nil {
		return nil, err
	}
	subBundleImageV1, err := fs.Sub(bundleImageV2, "testdata/foo-bundle-v0.1.0")
	if err != nil {
		return nil, err
	}
	subBundleImageV2, err := fs.Sub(bundleImageV2, "testdata/foo-bundle-v0.2.0")
	if err != nil {
		return nil, err
	}
	return &image.MockRegistry{
		RemoteImages: map[image.Reference]*image.MockImage{
			image.SimpleReference("test.registry/foo-operator/foo-index-sqlite:v0.2.0"): &image.MockImage{
				Labels: map[string]string{
					containertools.DbLocationLabel: "/database/index.db",
				},
				FS: subSqliteImage,
			},
			image.SimpleReference("test.registry/foo-operator/foo-index-declcfg:v0.2.0"): &image.MockImage{
				Labels: map[string]string{
					"operators.operatorframework.io.index.configs.v1": "/foo",
				},
				FS: subDeclcfgImage,
			},
			image.SimpleReference("test.registry/foo-operator/foo-bundle:v0.1.0"): &image.MockImage{
				Labels: map[string]string{
					bundle.PackageLabel: "foo",
				},
				FS: subBundleImageV1,
			},
			image.SimpleReference("test.registry/foo-operator/foo-bundle:v0.2.0"): &image.MockImage{
				Labels: map[string]string{
					bundle.PackageLabel: "foo",
				},
				FS: subBundleImageV2,
			},
		},
	}, nil
}
