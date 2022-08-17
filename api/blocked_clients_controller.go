package api

import (
	"context"
	"fmt"
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/pkg/util"
	"github.com/Peripli/service-manager/pkg/web"
	"github.com/Peripli/service-manager/storage"
	"net/http"
)

// BlockedClientsController configuration controller
type BlockedClientsController struct {
	*BaseController
	cache *storage.Cache
}

func NewBlockedClientsController(ctx context.Context, options *Options) *BlockedClientsController {
	return &BlockedClientsController{
		BaseController: NewController(ctx, options, web.BlockedClientsConfigURL, types.BlockedClientsType, func() types.Object {
			return &types.BlockedClient{}
		}, false),
		cache: options.BlockedClientsCache,
	}

}
func (c *BlockedClientsController) ResyncBlockedClientsCache(r *web.Request) (*web.Response, error) {
	err := c.cache.FlushL()
	if err != nil {
		return nil, &util.HTTPError{
			ErrorType:   "BlockedClientError",
			Description: fmt.Sprintf("failed to resync blocked_cleints cache"),
			StatusCode:  http.StatusInternalServerError,
		}
	}
	return util.NewJSONResponse(http.StatusOK, struct{}{})
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
				Method: http.MethodGet,
				Path:   web.ResyncBlockedClients,
			},
			Handler: c.ResyncBlockedClientsCache,
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
