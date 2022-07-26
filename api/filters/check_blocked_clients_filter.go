package filters

import (
	"context"
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/pkg/web"
	"github.com/Peripli/service-manager/storage"
	"net/http"
)

type BlockedClientsFilter struct {
	repository               storage.Repository
	updateBlockedClientsList func(ctx context.Context) []*types.BlockedClient
	cachedBlockedClientsList []*types.BlockedClient
}

// NewBlockedClientsFilter creates a new BlockedClientsFilter filter
func NewBlockedClientsFilter(ctx context.Context, repository storage.Repository) *BlockedClientsFilter {
	return &BlockedClientsFilter{
		repository:               repository,
		updateBlockedClientsList: getBlockedClientsList(repository),
		cachedBlockedClientsList: initializeBlockedClientsList(ctx, repository),
	}
}

func initializeBlockedClientsList(ctx context.Context, repo storage.Repository) []*types.BlockedClient {
	clientsList, err := repo.List(ctx, types.BlockedClientsType)
	if err != nil {
		return nil
	}

	blockedClients := clientsList.(*types.BlockedClients).BlockedClients
	return blockedClients
}

func getBlockedClientsList(repository storage.Repository) func(ctx context.Context) []*types.BlockedClient {
	return func(ctx context.Context) []*types.BlockedClient {
		clientsList, err := repository.List(ctx, types.BlockedClientsType)
		if err != nil {
			return nil
		}

		blockedClients := clientsList.(*types.BlockedClients).BlockedClients
		return blockedClients

	}

}

func (bc *BlockedClientsFilter) Name() string {
	return "BlockedClientsFilter"
}

func (bc *BlockedClientsFilter) Run(request *web.Request, next web.Handler) (*web.Response, error) {

	// get clientID from request - web.UserFromContext(request.Context())
	// call isClientBlocked function
	// if not - next.Handle(request)
	// if it is - return an error (what is the error message?)
	//if err != nil {
	//	log.C(request.Context()).WithError(err).Errorf("client is blocked - validate with Avi regarding this string")
	//	return nil, err
	//}

	return next.Handle(request)
}

func (bc *BlockedClientsFilter) isClientBlocked(client string) bool {
	// check if this ID is in cachedBlockedClientsList and whether he can use the request method (methods column)

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
