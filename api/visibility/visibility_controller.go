package visibility

import (
	"github.com/Peripli/service-manager/api/base"

	"github.com/Peripli/service-manager/storage"

	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/pkg/web"
)

type Controller struct {
	*base.Controller
}

var _ web.Controller = &Controller{}

func NewController(repository storage.Repository) *Controller {
	return &Controller{
		Controller: base.NewController(repository, web.VisibilitiesURL, func() types.Object {
			return &types.Visibility{}
		}),
	}
}
