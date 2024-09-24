package migrations

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"github.com/Masterminds/semver/v3"
	"k8s.io/apimachinery/pkg/util/sets"

	"github.com/operator-framework/operator-registry/alpha/declcfg"
	fbcv2 "github.com/operator-framework/operator-registry/alpha/fbc/v2"
	"github.com/operator-framework/operator-registry/alpha/property"
)

func v2(cfg *declcfg.DeclarativeConfig) error {
	slices.DeleteFunc(cfg.Packages, func(p declcfg.Package) bool {
		pkgV2 := fbcv2.Package{
			Schema:           "olm.package.v2",
			Package:          p.Name,
			ShortDescription: ellipsesDescription(p.Description),
			LongDescription:  p.Description,
		}
		cfg.PackagesV2 = append(cfg.PackagesV2, pkgV2)

		if p.Icon != nil {
			iconV2 := fbcv2.Icon{
				Schema:    "olm.icon.v2",
				Package:   p.Name,
				MediaType: p.Icon.MediaType,
				Data:      p.Icon.Data,
			}
			cfg.IconsV2 = append(cfg.IconsV2, iconV2)
		}
		return true
	})

	nameToVersion := map[string]semver.Version{}
	var errs []error
	slices.DeleteFunc(cfg.Bundles, func(b declcfg.Bundle) bool {
		var (
			vers *semver.Version
			err  error

			properties  = map[string]json.RawMessage{}
			constraints = map[string]json.RawMessage{}

			rawListProperties  = map[string][]json.RawMessage{}
			rawListConstraints = map[string][]json.RawMessage{}
		)

		for _, p := range b.Properties {
			switch p.Type {
			case property.TypePackage:
				var pkgProp property.Package
				if err := json.Unmarshal(p.Value, &pkgProp); err != nil {
					errs = append(errs, fmt.Errorf("could not migrate bundle %q: %v", b.Name, err))
					return true
				}
				vers, err = semver.NewVersion(pkgProp.Version)
				if err != nil {
					errs = append(errs, fmt.Errorf("could not migrate bundle %q: %v", b.Name, err))
					return true
				}
			case property.TypeCSVMetadata:
				properties[p.Type] = p.Value
			case property.TypeGVK, property.TypeBundleObject:
				rawListProperties[p.Type] = append(rawListProperties[p.Type], p.Value)
			case property.TypeGVKRequired, property.TypePackageRequired, property.TypeConstraint:
				rawListConstraints[p.Type] = append(rawListConstraints[p.Type], p.Value)
			default:
				errs = append(errs, fmt.Errorf("could not migrate bundle %q: unknown property type %q cannot be translated to v2 properties format", b.Name, p.Type))
				return true
			}
		}
		for k, v := range rawListProperties {
			properties[k], _ = json.Marshal(v)
		}

		for k, v := range rawListConstraints {
			constraints[k], _ = json.Marshal(v)
		}

		nameToVersion[b.Name] = *vers

		relatedURIs := sets.New[string]()
		for _, r := range b.RelatedImages {
			relatedURIs.Insert(fmt.Sprintf("docker://%s", r.Image))
		}
		bundleV2 := fbcv2.Bundle{
			Schema:      "olm.bundle.v2",
			Package:     b.Package,
			Name:        fmt.Sprintf("%s-%s-%d", b.Package, vers, 1),
			Version:     *vers,
			Release:     1,
			URI:         fmt.Sprintf("docker://%s", b.Image),
			RelatedURIs: sets.List(relatedURIs),
			Properties:  properties,
			Constraints: constraints,
		}
		cfg.BundlesV2 = append(cfg.BundlesV2, bundleV2)
		return true
	})
	if len(errs) > 0 {
		return fmt.Errorf("error migrating bundles: %v", errs)
	}

	slices.DeleteFunc(cfg.Channels, func(c declcfg.Channel) bool {
		entries := []fbcv2.ChannelEntry{}
		for _, e := range c.Entries {
			var upgradesFromBuilder strings.Builder
			if e.Replaces != "" {
				upgradesFromBuilder.WriteString(fmt.Sprintf("%s", nameToVersion[e.Replaces]))
			}
			for _, s := range e.Skips {
				upgradesFromBuilder.WriteString(fmt.Sprintf(" %s", nameToVersion[s]))
			}
			if e.SkipRange != "" {
				upgradesFromBuilder.WriteString(fmt.Sprintf(" %s", e.SkipRange))
			}
			upgradesFromStr := upgradesFromBuilder.String()

			// If the original channel entry has no upgrade edges, make
			// sure to explicitly configure it to an impossible range
			// equivalent to "no upgrades".
			if upgradesFromStr == "" {
				upgradesFromStr = fmt.Sprintf("<0.0.0 >0.0.0")
			}
			upgradesFrom, err := semver.NewConstraint(upgradesFromStr)
			if err != nil {
				errs = append(errs, fmt.Errorf("count not migrate channel %q: %v", c.Name, err))
				return true
			}

			entries = append(entries, fbcv2.ChannelEntry{
				Version:      nameToVersion[e.Name],
				UpgradesFrom: *upgradesFrom,
			})
		}

		rawListProperties := map[string][]json.RawMessage{}
		for _, p := range c.Properties {
			rawListProperties[p.Type] = append(rawListProperties[p.Type], p.Value)
		}

		properties := map[string]json.RawMessage{}
		for k, v := range rawListProperties {
			properties[k], _ = json.Marshal(v)
		}

		channelV2 := fbcv2.Channel{
			Schema:     "olm.channel.v2",
			Package:    c.Package,
			Name:       c.Name,
			Entries:    entries,
			Properties: properties,
		}
		cfg.ChannelsV2 = append(cfg.ChannelsV2, channelV2)
		return true
	})
	if len(errs) > 0 {
		return fmt.Errorf("error migrating channels: %v", errs)
	}

	return nil
}

func ellipsesDescription(s string) string {
	if len(s) <= 60 {
		return s
	}
	lastSpace := strings.LastIndex(s[:57], " ")
	if lastSpace == -1 {
		lastSpace = 57
	}
	return s[:lastSpace] + "..."
}
