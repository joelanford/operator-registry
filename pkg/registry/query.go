package registry

import (
	"github.com/operator-framework/operator-registry/pkg/api"
)

type SliceBundleSender []*api.Bundle

func (s *SliceBundleSender) Send(b *api.Bundle) error {
	*s = append(*s, b)
	return nil
}
