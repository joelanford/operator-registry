package v2

import (
	"encoding/json"
	"fmt"

	"github.com/Masterminds/semver/v3"
)

type Channel struct {
	// Schema is the FBC schema version of the channel. Always "olm.channel.v2".
	Schema string `json:"schema"`

	// Package is the name of the package to which this channel belongs.
	Package string `json:"package"`

	// Name is the name of the channel. It must be unique within a package.
	Name string `json:"name"`

	// Entries is a list of ChannelEntry objects that describe the bundles in
	// the channel.
	Entries []ChannelEntry `json:"entries"`

	// Annotations is a map of string keys to string values. Annotations are
	// used to store simple arbitrary metadata about the channel.
	Annotations map[string]string `json:"annotations,omitempty"`

	// Properties is a map of string keys to arbitrary JSON-encoded values.
	// Properties are used to store complex metadata. A property's "type" key
	// is used to determine how to interpret the JSON-encoded value.
	Properties map[string]json.RawMessage `json:"properties,omitempty"`
}

type ChannelEntry struct {
	// Version is the version of the bundle to be included in the channel.
	Version semver.Version `json:"version"`

	// UpgradesFrom is an optional constraint that specifies the range of
	// versions that can be upgraded to Version in the channel. If not
	// specified, semver semantics apply. That is to say the constraint
	// will be " >=X.0.0 <Version" where X is the major version of Version.
	UpgradesFrom semver.Constraints `json:"upgradesFrom,omitempty"`
}

var _ json.Unmarshaler = (*ChannelEntry)(nil)

func (c *ChannelEntry) UnmarshalJSON(bytes []byte) error {
	type entryUnmarshaler struct {
		Version      semver.Version `json:"version"`
		UpgradesFrom string         `json:"upgradesFrom,omitempty"`
	}
	var eu entryUnmarshaler
	if err := json.Unmarshal(bytes, &eu); err != nil {
		return err
	}
	c.Version = eu.Version
	if eu.UpgradesFrom == "" {
		eu.UpgradesFrom = fmt.Sprintf(">=%s <%s", minVersionFrom(eu.Version), eu.Version)
	}

	cf, err := semver.NewConstraint(eu.UpgradesFrom)
	if err != nil {
		return err
	}
	c.UpgradesFrom = *cf
	return nil
}

func minVersionFrom(v semver.Version) semver.Version {
	// 1.2.3 -> 1.0.0
	// 1.0.0 -> 1.0.0
	// 0.2.3 -> 0.2.0
	// 0.2.0 -> 0.2.0
	// 0.0.3 -> 0.0.3
	if v.Major() == 0 {
		if v.Minor() == 0 {
			return v
		}
		return *semver.New(0, v.Minor(), 0, "", "")
	}
	return *semver.New(v.Major(), 0, 0, "", "")
}
