package platform

import (
	"github.com/Peripli/service-manager/pkg/mutation"
	"github.com/Peripli/service-manager/pkg/security"
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/storage"
)

type Service struct {
	PlatformStorage storage.Platform
	Encrypter       security.Encrypter
	Mutator         mutation.Mutator
}

func (s *Service) Create(platform *types.Platform) error {
}
