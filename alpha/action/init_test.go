package action_test

import (
	"bytes"
	"errors"
	"testing"
	"testing/iotest"

	"github.com/stretchr/testify/require"

	action2 "github.com/operator-framework/operator-registry/alpha/action"
	declcfg2 "github.com/operator-framework/operator-registry/alpha/declcfg"
)

const (
	svgIcon = `<svg viewBox="0 0 100 100"><circle cx="25" cy="25" r="25"/></svg>`
)

func TestInit(t *testing.T) {
	type spec struct {
		name      string
		init      action2.Init
		expectPkg *declcfg2.Package
		assertion require.ErrorAssertionFunc
	}

	specs := []spec{
		{
			name: "Success/Empty",
			init: action2.Init{},
			expectPkg: &declcfg2.Package{
				Schema: "olm.package",
			},
			assertion: require.NoError,
		},
		{
			name: "Success/SetPackage",
			init: action2.Init{
				Package: "foo",
			},
			expectPkg: &declcfg2.Package{
				Schema: "olm.package",
				Name:   "foo",
			},
			assertion: require.NoError,
		},
		{
			name: "Success/SetDefaultChannel",
			init: action2.Init{
				DefaultChannel: "foo",
			},
			expectPkg: &declcfg2.Package{
				Schema:         "olm.package",
				DefaultChannel: "foo",
			},
			assertion: require.NoError,
		},
		{
			name: "Success/SetDescription",
			init: action2.Init{
				DescriptionReader: bytes.NewBufferString("foo"),
			},
			expectPkg: &declcfg2.Package{
				Schema:      "olm.package",
				Description: "foo",
			},
			assertion: require.NoError,
		},
		{
			name: "Success/SetIcon",
			init: action2.Init{
				IconReader: bytes.NewBufferString(svgIcon),
			},
			expectPkg: &declcfg2.Package{
				Schema: "olm.package",
				Icon: &declcfg2.Icon{
					Data:      bytes.NewBufferString(svgIcon).Bytes(),
					MediaType: "image/svg+xml",
				},
			},
			assertion: require.NoError,
		},
		{
			name: "Success/SetAll",
			init: action2.Init{
				Package:           "a",
				DefaultChannel:    "b",
				DescriptionReader: bytes.NewBufferString("c"),
				IconReader:        bytes.NewBufferString(svgIcon),
			},
			expectPkg: &declcfg2.Package{
				Schema:         "olm.package",
				Name:           "a",
				DefaultChannel: "b",
				Description:    "c",
				Icon: &declcfg2.Icon{
					Data:      bytes.NewBufferString(svgIcon).Bytes(),
					MediaType: "image/svg+xml",
				},
			},
			assertion: require.NoError,
		},
		{
			name: "Fail/ReadDescription",
			init: action2.Init{
				DescriptionReader: iotest.ErrReader(errors.New("fail")),
			},
			assertion: require.Error,
		},
		{
			name: "Fail/ReadIcon",
			init: action2.Init{
				IconReader: iotest.ErrReader(errors.New("fail")),
			},
			assertion: require.Error,
		},
		{
			name: "Fail/IconNotImage",
			init: action2.Init{
				IconReader: bytes.NewBufferString("foo"),
			},
			assertion: require.Error,
		},
		{
			name: "Fail/EmptyIcon",
			init: action2.Init{
				IconReader: bytes.NewBuffer(nil),
			},
			assertion: require.Error,
		},
	}
	for _, s := range specs {
		t.Run(s.name, func(t *testing.T) {
			actualPkg, actualErr := s.init.Run()
			s.assertion(t, actualErr)
			require.Equal(t, s.expectPkg, actualPkg)
		})
	}
}
