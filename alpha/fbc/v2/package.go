package v2

import (
	"encoding/json"
)

type Package struct {
	// Schema is the FBC schema version of the package. Always "olm.package.v2".
	Schema string `json:"schema"`

	// Package is the name of this package.
	Package string `json:"package"`

	// ShortDescription is a short description of the package.
	ShortDescription string `json:"shortDescription,omitempty"`

	// LongDescription is a long description of the package.
	LongDescription string `json:"longDescription,omitempty"`

	// Annotations is a map of string keys to string values. Annotations are
	// used to store simple arbitrary metadata about the package.
	Annotations map[string]string `json:"annotations,omitempty"`

	// Properties is a map of string keys to arbitrary JSON-encoded values.
	// Properties are used to store complex metadata about the package. A
	// property's "type" key is used to determine how to interpret the
	// JSON-encoded value. Unrecognized properties MUST be ignored.
	Properties map[string]json.RawMessage `json:"properties,omitempty"`
}
