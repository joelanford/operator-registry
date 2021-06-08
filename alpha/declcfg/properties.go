package declcfg

import (
	"fmt"

	property2 "github.com/operator-framework/operator-registry/alpha/property"
)

func parseProperties(props []property2.Property) (*property2.Properties, error) {
	out, err := property2.Parse(props)
	if err != nil {
		return nil, err
	}

	channels := map[string]struct{}{}
	for _, ch := range out.Channels {
		if _, ok := channels[ch.Name]; ok {
			return nil, propertyDuplicateError{typ: property2.TypeChannel, key: ch.Name}
		}
		channels[ch.Name] = struct{}{}
	}
	return out, nil
}

type propertyDuplicateError struct {
	typ string
	key string
}

func (e propertyDuplicateError) Error() string {
	return fmt.Sprintf("duplicate property of type %q found with key %q", e.typ, e.key)
}
