package declcfg

import (
	"fmt"

	model2 "github.com/operator-framework/operator-registry/alpha/model"
	property2 "github.com/operator-framework/operator-registry/alpha/property"
)

func ConvertToModel(cfg DeclarativeConfig) (model2.Model, error) {
	mpkgs := model2.Model{}
	defaultChannels := map[string]string{}
	for _, p := range cfg.Packages {
		mpkg := &model2.Package{
			Name:        p.Name,
			Description: p.Description,
			Channels:    map[string]*model2.Channel{},
		}
		if p.Icon != nil {
			mpkg.Icon = &model2.Icon{
				Data:      p.Icon.Data,
				MediaType: p.Icon.MediaType,
			}
		}
		defaultChannels[p.Name] = p.DefaultChannel
		mpkgs[p.Name] = mpkg
	}

	for _, b := range cfg.Bundles {
		defaultChannelName := defaultChannels[b.Package]
		if b.Package == "" {
			return nil, fmt.Errorf("package name must be set for bundle %q", b.Name)
		}
		mpkg, ok := mpkgs[b.Package]
		if !ok {
			return nil, fmt.Errorf("unknown package %q for bundle %q", b.Package, b.Name)
		}

		props, err := parseProperties(b.Properties)
		if err != nil {
			return nil, fmt.Errorf("parse properties for bundle %q: %v", b.Name, err)
		}

		if len(props.Packages) == 0 {
			return nil, fmt.Errorf("missing package property for bundle %q", b.Name)
		}

		if b.Package != props.Packages[0].PackageName {
			return nil, fmt.Errorf("package %q does not match %q property %q", b.Package, property2.TypePackage, props.Packages[0].PackageName)
		}

		if len(props.Channels) == 0 {
			return nil, fmt.Errorf("bundle %q is missing channel information", b.Name)
		}

		for _, bundleChannel := range props.Channels {
			pkgChannel, ok := mpkg.Channels[bundleChannel.Name]
			if !ok {
				pkgChannel = &model2.Channel{
					Package: mpkg,
					Name:    bundleChannel.Name,
					Bundles: map[string]*model2.Bundle{},
				}
				if bundleChannel.Name == defaultChannelName {
					mpkg.DefaultChannel = pkgChannel
				}
				mpkg.Channels[bundleChannel.Name] = pkgChannel
			}
			pkgChannel.Bundles[b.Name] = &model2.Bundle{
				Package:       mpkg,
				Channel:       pkgChannel,
				Name:          b.Name,
				Image:         b.Image,
				Replaces:      bundleChannel.Replaces,
				Skips:         skipsToStrings(props.Skips),
				Properties:    b.Properties,
				RelatedImages: relatedImagesToModelRelatedImages(b.RelatedImages),
				CsvJSON:       b.CsvJSON,
				Objects:       b.Objects,
			}
		}
	}

	for _, mpkg := range mpkgs {
		defaultChannelName := defaultChannels[mpkg.Name]
		if defaultChannelName != "" && mpkg.DefaultChannel == nil {
			dch := &model2.Channel{
				Package: mpkg,
				Name:    defaultChannelName,
				Bundles: map[string]*model2.Bundle{},
			}
			mpkg.DefaultChannel = dch
			mpkg.Channels[dch.Name] = dch
		}
	}

	if err := mpkgs.Validate(); err != nil {
		return nil, err
	}
	mpkgs.Normalize()
	return mpkgs, nil
}

func skipsToStrings(in []property2.Skips) []string {
	var out []string
	for _, s := range in {
		out = append(out, string(s))
	}
	return out
}

func relatedImagesToModelRelatedImages(in []RelatedImage) []model2.RelatedImage {
	var out []model2.RelatedImage
	for _, p := range in {
		out = append(out, model2.RelatedImage{
			Name:  p.Name,
			Image: p.Image,
		})
	}
	return out
}
