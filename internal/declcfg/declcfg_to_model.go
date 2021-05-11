package declcfg

import (
	"fmt"

	"k8s.io/apimachinery/pkg/util/sets"

	"github.com/operator-framework/operator-registry/internal/model"
	"github.com/operator-framework/operator-registry/internal/property"
)

func convertToModelPackage(m model.Model, p Package) error {
	if p.Name == "" {
		return fmt.Errorf("package name empty")
	}
	if _, ok := m[p.Name]; ok {
		return fmt.Errorf("duplicate package %q", p.Name)
	}
	pkg := &model.Package{
		Name:        p.Name,
		Description: p.Description,
		Channels:    map[string]*model.Channel{},
	}
	if p.Icon != nil {
		pkg.Icon = &model.Icon{
			Data:      p.Icon.Data,
			MediaType: p.Icon.MediaType,
		}
	}
	m[p.Name] = pkg
	return nil
}

func convertToModelChannel(m model.Model, c Channel, bundles map[string]Bundle) error {
	pkg, ok := m[c.Package]
	if !ok {
		return fmt.Errorf("unknown package %q", c.Package)
	}
	if c.Name == "" {
		return fmt.Errorf("channel name empty in package %q", pkg.Name)
	}
	if _, ok := pkg.Channels[c.Name]; ok {
		return fmt.Errorf("duplicate channel %q in package %q", c.Name, pkg.Name)
	}
	ch := &model.Channel{
		Package: pkg,
		Name:    c.Name,
		Bundles: make(map[string]*model.Bundle),
	}

	nodes, skipped := buildNodes(c.Edges, c.BlockedEdges)
	replacesList, skipsList := buildEdges(c.Edges, skipped)

	for name := range nodes {
		b, ok := bundles[name]
		if !ok {
			if skipped.Has(name) {
				continue
			}
			return fmt.Errorf("bundle with name %q not found", name)
		}
		replaces := replacesList[name]
		replace := ""
		if replaces.Len() > 1 {
			return fmt.Errorf("bundle %q replaces %v, but only one replace edge is allowed", b.Name, replaces)
		} else if len(replaces) == 1 {
			replace = replaces.List()[0]
		}
		skips := []string{}
		for _, skip := range skipsList[name].List() {
			skips = append(skips, skip)
		}
		props := b.Properties[:0]
		for _, p := range b.Properties {
			if p.Type != property.TypeChannel && p.Type != property.TypeSkips {
				props = append(props, p)
			}
		}
		props = append(props, property.MustBuildChannel(ch.Name, replace))
		for _, s := range skips {
			props = append(props, property.MustBuildSkips(s))
		}
		mb := &model.Bundle{
			Package:       pkg,
			Channel:       ch,
			Name:          b.Name,
			Image:         b.Image,
			Replaces:      replace,
			Skips:         skips,
			Properties:    props,
			RelatedImages: relatedImagesToModelRelatedImages(b.RelatedImages),
			CsvJSON:       b.CsvJSON,
			Objects:       b.Objects,
		}
		ch.Bundles[mb.Name] = mb
	}
	pkg.Channels[c.Name] = ch
	return nil
}

func buildNodes(edges []Edge, blockedEdges []Edge) (sets.String, sets.String) {
	nodes := sets.NewString()
	skipped := sets.NewString()
	for _, e := range edges {
		nodes.Insert(e.From, e.To)
	}
	for _, e := range blockedEdges {
		nodes.Insert(e.From, e.To)
		skipped.Insert(e.To)
	}
	nodes.Delete("")
	skipped.Delete("")
	return nodes, skipped
}

func buildEdges(edges []Edge, skipped sets.String) (map[string]sets.String, map[string]sets.String) {
	addEdge := func(l map[string]sets.String, from string, to string) {
		froms, ok := l[to]
		if !ok {
			froms = sets.NewString()
		}
		if from != "" {
			froms.Insert(from)
		}
		l[to] = froms
	}
	replaces := map[string]sets.String{}
	skips := map[string]sets.String{}
	for _, e := range edges {
		if skipped.Has(e.From) {
			addEdge(skips, e.From, e.To)
		} else {
			addEdge(replaces, e.From, e.To)
		}
	}
	return replaces, skips
}

func mapBundles(bundles []Bundle) (map[string]Bundle, error) {
	bundlesMap := make(map[string]Bundle, len(bundles))
	for _, b := range bundles {
		bundlesMap[b.Name] = b
	}
	return bundlesMap, nil
}

func ConvertToModel(cfg DeclarativeConfig) (model.Model, error) {
	m := model.Model{}

	// 1. Create model packages
	// 2. Organize bundles by package and version.
	// 2. Create model channels

	for _, pkg := range cfg.Packages {
		if err := convertToModelPackage(m, pkg); err != nil {
			return nil, fmt.Errorf("convert package: %v", err)
		}
	}
	bundlesMap, err := mapBundles(cfg.Bundles)
	if err != nil {
		return nil, fmt.Errorf("map bundles: %v", err)
	}

	for _, ch := range cfg.Channels {
		if err := convertToModelChannel(m, ch, bundlesMap); err != nil {
			return nil, fmt.Errorf("convert channel: %v", err)
		}
	}

	for _, pkg := range cfg.Packages {
		mpkg := m[pkg.Name]
		defaultChannelName := pkg.DefaultChannel
		defaultChannel, ok := mpkg.Channels[defaultChannelName]
		if !ok {
			return nil, fmt.Errorf("default channel %q for package %q not found", defaultChannelName, pkg.Name)
		}
		mpkg.DefaultChannel = defaultChannel
	}

	if err := m.Validate(); err != nil {
		return nil, err
	}
	m.Normalize()
	return m, nil
}

func relatedImagesToModelRelatedImages(in []RelatedImage) []model.RelatedImage {
	var out []model.RelatedImage
	for _, p := range in {
		out = append(out, model.RelatedImage{
			Name:  p.Name,
			Image: p.Image,
		})
	}
	return out
}
