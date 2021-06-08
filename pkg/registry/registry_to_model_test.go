package registry

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	model2 "github.com/operator-framework/operator-registry/alpha/model"
	property2 "github.com/operator-framework/operator-registry/alpha/property"
	"github.com/operator-framework/operator-registry/pkg/image"
)

func TestConvertRegistryBundleToModelBundle(t *testing.T) {
	registryBundle, err := testRegistryBundle()
	require.NoError(t, err)
	expected := testModelBundle()

	actual, err := registryBundleToModelBundle(registryBundle)
	require.NoError(t, err)
	assertEqualsModelBundle(t, expected, *actual)

	registryBundles, err := ConvertRegistryBundleToModelBundles(registryBundle)
	assert.Equal(t, len(registryBundles), 2)
}

func testModelBundle() model2.Bundle {
	b := model2.Bundle{
		Name:     "etcdoperator.v0.9.2",
		Image:    "quay.io/operatorhubio/etcd:v0.9.2",
		Replaces: "etcdoperator.v0.9.0",
		Skips:    []string{"etcdoperator.v0.9.1"},
		Properties: []property2.Property{
			property2.MustBuildChannel("alpha", "etcdoperator.v0.9.0"),
			property2.MustBuildChannel("stable", "etcdoperator.v0.9.0"),
			property2.MustBuildPackage("etcd", "0.9.2"),
			property2.MustBuildSkips("etcdoperator.v0.9.1"),
			property2.MustBuildGVKRequired("etcd.database.coreos.com", "v1beta2", "EtcdCluster"),
			property2.MustBuildGVKRequired("testapi.coreos.com", "v1", "testapi"),
			property2.MustBuildGVK("etcd.database.coreos.com", "v1beta2", "EtcdCluster"),
			property2.MustBuildGVK("etcd.database.coreos.com", "v1beta2", "EtcdBackup"),
			property2.MustBuildGVK("etcd.database.coreos.com", "v1beta2", "EtcdRestore"),
		},
	}
	return b
}

func testRegistryBundle() (*Bundle, error) {
	input, err := NewImageInput(image.SimpleReference("quay.io/operatorhubio/etcd:v0.9.2"), "../../bundles/etcd.0.9.2")
	if err != nil {
		return nil, err
	}
	return input.Bundle, nil
}

func assertEqualsModelBundle(t *testing.T, a, b model2.Bundle) bool {
	assert.ElementsMatch(t, a.Properties, b.Properties)
	assert.ElementsMatch(t, a.Skips, b.Skips)
	assert.ElementsMatch(t, a.RelatedImages, b.RelatedImages)

	a.Properties, b.Properties = nil, nil
	a.Objects, b.Objects = nil, nil
	a.Skips, b.Skips = nil, nil
	a.RelatedImages, b.RelatedImages = nil, nil

	return assert.Equal(t, a, b)
}
