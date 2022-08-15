package api

import (
	"context"
	"fmt"
	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/types"
	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/web"
	"net/http"
)

// OperationsController implements api.Controller by providing operations API logic
type OperationsController struct {
	*BaseController
}

// NewOperationsController returns a new controller for operations api
func NewOperationsController(ctx context.Context, options *Options) *OperationsController {
	return &OperationsController{
		BaseController: NewController(ctx, options, web.OperationsURL, types.OperationType, func() types.Object {
			return &types.Operation{}
		}, false),
	}
}

func (c *OperationsController) Routes() []web.Route {
	return []web.Route{
		{
			Endpoint: web.Endpoint{
				Method: http.MethodGet,
				Path:   web.OperationsURL,
			},
			Handler: c.ListObjects,
		},
		{
			Endpoint: web.Endpoint{
				Method: http.MethodDelete,
				Path:   fmt.Sprintf("%s/{%s}", c.resourceBaseURL, web.PathParamResourceID),
			},
			Handler: c.DeleteSingleObject,
		},
	}
}
