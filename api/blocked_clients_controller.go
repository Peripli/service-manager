package api

import (
	"context"
	"fmt"
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/pkg/web"
	"net/http"
)

// BlockedClientsController configuration controller
type BlockedClientsController struct {
	*BaseController
}

func NewBlockedClientsController(ctx context.Context, options *Options) *BlockedClientsController {

	return &BlockedClientsController{
		BaseController: NewController(ctx, options, web.BlockedClientsConfigURL, types.BlockedClientsType, func() types.Object {
			return &types.BlockedClient{}
		}, false),
	}

}

// Routes provides endpoints for modifying and obtaining the logging configuration
func (c *BlockedClientsController) Routes() []web.Route {
	return []web.Route{
		{
			Endpoint: web.Endpoint{
				Method: http.MethodGet,
				Path:   web.BlockedClientsConfigURL,
			},
			Handler: c.ListObjects,
		},
		{
			Endpoint: web.Endpoint{
				Method: http.MethodPost,
				Path:   web.BlockedClientsConfigURL,
			},
			Handler: c.CreateObject,
		},
		{
			Endpoint: web.Endpoint{
				Method: http.MethodDelete,
				Path:   fmt.Sprintf("%s/{%s}", web.BlockedClientsConfigURL, web.PathParamResourceID),
			},
			Handler: c.DeleteSingleObject,
		},
	}
}
