package filters

import (
	"fmt"
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/pkg/util"
	"github.com/Peripli/service-manager/pkg/web"
	"github.com/Peripli/service-manager/storage"
	"net/http"
)

const BlockedClientsFilterName = "BlockedClientsFilter"

type BlockedClientsFilter struct {
	blockedClientsCache *storage.Cache
	tenantLabelKey      string
}

func NewBlockedClientsFilter(cache *storage.Cache, tenantLabelKey string) *BlockedClientsFilter {
	b := &BlockedClientsFilter{blockedClientsCache: cache, tenantLabelKey: tenantLabelKey}
	return b
}
func (b *BlockedClientsFilter) Name() string {
	return BlockedClientsFilterName
}

func (b *BlockedClientsFilter) Run(request *web.Request, next web.Handler) (*web.Response, error) {
	reqCtx := request.Context()
	method := request.Method
	userContext, ok := web.UserFromContext(reqCtx)
	if !ok {
		//there is no context on the endpoint
		return next.Handle(request)
	}
	blockedClient, isBlockedClient := b.isClientBlocked(userContext, method)
	if isBlockedClient {
		errorResponse := &util.HTTPError{
			ErrorType:  "RequestNotAllowed",
			StatusCode: http.StatusMethodNotAllowed,
		}

		errorResponse.Description = fmt.Sprintf("You're blocked to execute this request. Client: %d ", blockedClient.ClientID)

		return nil, errorResponse

	}

	return next.Handle(request)
}

func (bc *BlockedClientsFilter) isClientBlocked(userContext *web.UserContext, method string) (*types.BlockedClient, bool) {
	//don't restrict global users
	if userContext.AccessLevel == web.GlobalAccess || userContext.AccessLevel == web.AllTenantAccess {
		return nil, false
	}

	if userContext.AuthenticationType == web.Basic {
		platform := types.Platform{}
		err := userContext.Data(&platform)
		if err != nil {
			return nil, false
		}

		if _, isTenantScopedPlatform := platform.Labels[bc.tenantLabelKey]; !isTenantScopedPlatform {
			return nil, false
		}

	}

	blockedClientCache, ok := bc.blockedClientsCache.Get(userContext.Name)
	if !ok {
		return nil, false
	}
	blockedClient := blockedClientCache.(types.BlockedClient)
	return &blockedClient, contains(blockedClient.BlockedMethods, method)

}
func contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

func (bc *BlockedClientsFilter) FilterMatchers() []web.FilterMatcher {
	return []web.FilterMatcher{
		{
			Matchers: []web.Matcher{
				web.Path("/**"),
				web.Methods(http.MethodPost, http.MethodPatch, http.MethodGet, http.MethodDelete, http.MethodOptions),
			},
		},
	}
}
