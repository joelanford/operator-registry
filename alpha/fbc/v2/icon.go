package v2

type Icon struct {
	// Schema is the FBC schema version of the icon. Always "olm.icon.v2".
	Schema string `json:"schema"`

	// Package is the name of the package to which this icon belongs.
	Package string `json:"package"`

	// MediaType is the media type of the icon.
	MediaType string `json:"mediaType"`

	// Data is the icon data
	Data []byte `json:"data"`
}
