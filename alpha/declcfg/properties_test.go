package declcfg

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"

	property2 "github.com/operator-framework/operator-registry/alpha/property"
)

func TestParseProperties(t *testing.T) {
	type spec struct {
		name          string
		properties    []property2.Property
		expectErrType error
		expectProps   *property2.Properties
	}

	specs := []spec{
		{
			name: "Error/InvalidChannel",
			properties: []property2.Property{
				{Type: property2.TypeChannel, Value: json.RawMessage(`""`)},
			},
			expectErrType: property2.ParseError{},
		},
		{
			name: "Error/InvalidSkips",
			properties: []property2.Property{
				{Type: property2.TypeSkips, Value: json.RawMessage(`{}`)},
			},
			expectErrType: property2.ParseError{},
		},
		{
			name: "Error/DuplicateChannels",
			properties: []property2.Property{
				property2.MustBuildChannel("alpha", "foo.v0.0.3"),
				property2.MustBuildChannel("beta", "foo.v0.0.3"),
				property2.MustBuildChannel("alpha", "foo.v0.0.4"),
			},
			expectErrType: propertyDuplicateError{},
		},
		{
			name: "Success/Valid",
			properties: []property2.Property{
				property2.MustBuildChannel("alpha", "foo.v0.0.3"),
				property2.MustBuildChannel("beta", "foo.v0.0.4"),
				property2.MustBuildSkips("foo.v0.0.1"),
				property2.MustBuildSkips("foo.v0.0.2"),
			},
			expectProps: &property2.Properties{
				Channels: []property2.Channel{
					{Name: "alpha", Replaces: "foo.v0.0.3"},
					{Name: "beta", Replaces: "foo.v0.0.4"},
				},
				Skips: []property2.Skips{"foo.v0.0.1", "foo.v0.0.2"},
			},
		},
	}

	for _, s := range specs {
		t.Run(s.name, func(t *testing.T) {
			props, err := parseProperties(s.properties)
			if s.expectErrType != nil {
				assert.IsType(t, s.expectErrType, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, s.expectProps, props)
			}
		})
	}
}
