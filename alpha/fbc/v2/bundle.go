package v2

import (
	"encoding/json"

	"github.com/Masterminds/semver/v3"
)

type Bundle struct {
	// Schema is the FBC schema version of the bundle. Always "olm.bundle.v2".
	Schema string `json:"schema"`

	// Package is the name of the package to which this bundle belongs.
	Package string `json:"package"`

	// Name is the name of the bundle. It is required to be in the format of
	// "package-version-release", and mnust be unique within a catalog.
	Name string `json:"name"`

	// Version is the version of the software packaged in this bundle.
	Version semver.Version `json:"version"`

	// Release is the number of times the Version of this bundle has been
	// released.
	Release uint32 `json:"release"`

	// URI is the location of the bundle.
	URI string `json:"uri"`

	// RelatedURIs is a list of related URIs that are associated with the
	// bundle. URIs should be included here if they are referenced or used by
	// the bundle.
	RelatedURIs []string `json:"relatedURIs,omitempty"`

	// Annotations is a map of string keys to string values. Annotations are
	// used to store simple arbitrary metadata about the bundle.
	Annotations map[string]string `json:"annotations,omitempty"`

	// Properties is a map of string keys to arbitrary JSON-encoded values.
	// Properties are used to store complex metadata about the bundle. A
	// property's "type" key is used to determine how to interpret the
	// JSON-encoded value. Unrecognized properties MUST be ignored.
	Properties map[string]json.RawMessage `json:"properties,omitempty"`

	// Constraints is a map of string keys to arbitrary JSON-encoded values.
	// Constraints are used to store complex constraints that the bundle
	// requires. A constraint's "type" key is used to determine how to
	// interpret the JSON-encoded value. Unrecognized constraints MUST be
	// treated as unsatisfiable.
	Constraints map[string]json.RawMessage `json:"constraints,omitempty"`
}
