package api

import (
	"context"
	"encoding/json"
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

var AVAILABLE_METHODS map[string]byte = map[string]byte{
	http.MethodDelete:  1,
	http.MethodGet:     1,
	http.MethodPut:     1,
	http.MethodPost:    1,
	http.MethodOptions: 1,
	http.MethodPatch:   1,
}

func (c *BlockedClientsController) AddBlockedClient(r *web.Request) (*web.Response, error) {

	var blockedClient types.BlockedClient
	err := json.Unmarshal(r.Body, &blockedClient)
	if err != nil {
		return nil, &util.HTTPError{
			ErrorType:  "BlockedClientError",
			StatusCode: http.StatusBadRequest,
		}
	}

	if len(blockedClient.BlockedMethods) > 0 {
		for _, method := range blockedClient.BlockedMethods {
			if _, ok := AVAILABLE_METHODS[method]; !ok {
				return nil, &util.HTTPError{
					ErrorType:   "BlockedClientError",
					Description: fmt.Sprintf("Invalid value for a blocked method. Allowed methods to block are: GET, PUT, POST, PATCH, DELETE, OPTIONS."),
					StatusCode:  http.StatusBadRequest,
				}
			}
		}
	}
	return c.CreateObject(r)
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
			Handler: c.AddBlockedClient,
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
