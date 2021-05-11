package action

import (
	"fmt"
	"io/ioutil"

	"github.com/h2non/filetype"

	"github.com/operator-framework/operator-registry/internal/declcfg"
)

type Init struct {
	Package         string
	DefaultChannel  string
	DescriptionFile string
	IconFile        string
}

func (i Init) Run() (*declcfg.Package, error) {
	pkg := &declcfg.Package{
		Schema:         "olm.package",
		Name:           i.Package,
		DefaultChannel: i.DefaultChannel,
	}
	if i.DescriptionFile != "" {
		descriptionData, err := ioutil.ReadFile(i.DescriptionFile)
		if err != nil {
			return nil, fmt.Errorf("read description file %q: %v", i.DescriptionFile, err)
		}
		pkg.Description = string(descriptionData)
	}

	if i.IconFile != "" {
		iconData, err := ioutil.ReadFile(i.IconFile)
		if err != nil {
			return nil, fmt.Errorf("read icon file %q: %v", i.IconFile, err)
		}
		iconType, err := filetype.Match(iconData)
		if err != nil {
			return nil, fmt.Errorf("detect icon mediatype: %v", err)
		}
		if iconType.MIME.Type != "image" {
			return nil, fmt.Errorf("detected invalid type %q: not an image", iconType.MIME.Value)
		}
		pkg.Icon = &declcfg.Icon{
			Data:      iconData,
			MediaType: iconType.MIME.Value,
		}
	}
	return pkg, nil
}
