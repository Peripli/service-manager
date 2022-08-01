package filters

import (
	"context"
	"errors"
	"fmt"
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/pkg/util"
	"github.com/Peripli/service-manager/pkg/web"
	"github.com/Peripli/service-manager/storage"
	"github.com/lib/pq"
	"net/http"
	"sync"
	"time"
)

type BlockedClientsFilter struct {
	repository               storage.Repository
	ctx                      context.Context
	cache                    sync.Map
	storageURI               string
	updateBlockedClientsList func(ctx context.Context) []*types.BlockedClient
}

// NewBlockedClientsFilter creates a new BlockedClientsFilter filter
func NewBlockedClientsFilter(ctx context.Context, repository storage.Repository, storageURI string) *BlockedClientsFilter {
	blockedClientsFilter := &BlockedClientsFilter{
		repository: repository,
		ctx:        ctx,
		cache:      sync.Map{},
		storageURI: storageURI,
	}
	blockedClientsFilter.initializeBlockedClients()
	return blockedClientsFilter

}

func (b *BlockedClientsFilter) connectDBForBlockedClientsEvent() error {
	reportProblem := func(et pq.ListenerEventType, err error) {
		if err != nil {
			//add login
			fmt.Println(err)
		}
	}
	listener := pq.NewListener(b.storageURI, 30*time.Second, time.Minute, reportProblem)
	err := listener.Listen("new-blocked-client")
	if err != nil {
		return err
	}
	return nil

}

func (b *BlockedClientsFilter) initializeBlockedClients() (err error) {

	blockedClientsList, err := b.repository.List(b.ctx, types.BlockedClientsType)
	if err != nil {
		return err
	}
	for i := 0; i < blockedClientsList.Len(); i++ {
		blockedClient := blockedClientsList.ItemAt(i).(*types.BlockedClient)
		b.cache.Store(blockedClient.ClientID, blockedClient)
	}
	return nil
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

func (b *BlockedClientsFilter) Name() string {
	return "BlockedClientsFilter"
}

func (b *BlockedClientsFilter) Run(request *web.Request, next web.Handler) (*web.Response, error) {
	reqCtx := request.Context()
	method := request.Method
	userContext, ok := web.UserFromContext(reqCtx)
	if !ok {
		return nil, errors.New("no client found")
	}
	blockedClient, isBlockedClient := b.isClientBlocked(userContext, method)
	if isBlockedClient {
		errorResponse := &util.HTTPError{
			ErrorType:   "RequestNotAllowed",
			Description: fmt.Sprintf("You're blocked to execute this request. Client: %d ", blockedClient.ClientID),
			StatusCode:  http.StatusMethodNotAllowed,
		}
		return nil, errorResponse

	}
	// if not - next.Handle(request)
	// if it is - return an error (what is the error message?)
	//if err != nil {
	//	log.C(request.Context()).WithError(err).Errorf("client is blocked - validate with Avi regarding this string")
	//	return nil, err
	//}

	return next.Handle(request)
}

func (bc *BlockedClientsFilter) isClientBlocked(userContext *web.UserContext, method string) (*types.BlockedClient, bool) {
	//don't restrict global users
	if userContext.AccessLevel == web.GlobalAccess || userContext.AccessLevel == web.AllTenantAccess {
		return nil, false
	}
	blockedClientCache, ok := bc.cache.Load(userContext.Name)
	if !ok {
		return nil, true
	}
	blockedClient := blockedClientCache.(*types.BlockedClient)
	// add to retrieved from db
	return blockedClient, contains(blockedClient.BlockedMethods, method)

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
