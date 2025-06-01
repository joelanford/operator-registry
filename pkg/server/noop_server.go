package server

import (
	"context"

	"github.com/operator-framework/operator-registry/pkg/api"
)

var _ api.RegistryServer = &NoopServer{}

type NoopServer struct {
	api.UnimplementedRegistryServer
}

func (n NoopServer) ListPackages(request *api.ListPackageRequest, server api.Registry_ListPackagesServer) error {
	return nil
}

func (n NoopServer) GetPackage(ctx context.Context, request *api.GetPackageRequest) (*api.Package, error) {
	return nil, nil
}

func (n NoopServer) GetBundle(ctx context.Context, request *api.GetBundleRequest) (*api.Bundle, error) {
	return nil, nil
}

func (n NoopServer) GetBundleForChannel(ctx context.Context, request *api.GetBundleInChannelRequest) (*api.Bundle, error) {
	return nil, nil
}

func (n NoopServer) GetChannelEntriesThatReplace(request *api.GetAllReplacementsRequest, server api.Registry_GetChannelEntriesThatReplaceServer) error {
	return nil
}

func (n NoopServer) GetBundleThatReplaces(ctx context.Context, request *api.GetReplacementRequest) (*api.Bundle, error) {
	return nil, nil
}

func (n NoopServer) GetChannelEntriesThatProvide(request *api.GetAllProvidersRequest, server api.Registry_GetChannelEntriesThatProvideServer) error {
	return nil
}

func (n NoopServer) GetLatestChannelEntriesThatProvide(request *api.GetLatestProvidersRequest, server api.Registry_GetLatestChannelEntriesThatProvideServer) error {
	return nil
}

func (n NoopServer) GetDefaultBundleThatProvides(ctx context.Context, request *api.GetDefaultProviderRequest) (*api.Bundle, error) {
	return nil, nil
}

func (n NoopServer) ListBundles(request *api.ListBundlesRequest, server api.Registry_ListBundlesServer) error {
	return nil
}
