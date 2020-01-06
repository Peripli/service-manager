package api

import (
	"fmt"
	"net/http"

	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/pkg/web"
	"github.com/Peripli/service-manager/storage"
)

// OperationsController implements api.Controller by providing operations API logic
type OperationsController struct {
	*BaseController
}

func NewOperationsController(repository storage.Repository, defaultPageSize, maxPageSize int) *OperationsController {
	return &OperationsController{
		BaseController: NewController(repository, web.OperationsURL, types.OperationType, func() types.Object {
			return &types.Operation{}
		}, defaultPageSize, maxPageSize),
	}
}

func (c *OperationsController) Routes() []web.Route {
	return []web.Route{
		{
			Endpoint: web.Endpoint{
				Method: http.MethodGet,
				Path:   fmt.Sprintf("%s/{%s}", web.OperationsURL, PathParamID),
			},
			Handler: c.GetSingleObject,
		},
	}
}
