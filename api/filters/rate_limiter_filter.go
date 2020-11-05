package filters

import (
	"context"
	"fmt"
	"github.com/Peripli/service-manager/pkg/log"
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/pkg/util"
	"github.com/Peripli/service-manager/pkg/util/slice"
	"github.com/Peripli/service-manager/pkg/web"
	"github.com/ulule/limiter"
	"github.com/ulule/limiter/drivers/middleware/stdlib"
	"net/http"
	"strconv"
	"time"
)

type RateLimiterFilter struct {
	middleware  *stdlib.Middleware
	excludeList []string
}

func NewRateLimiterFilter(middleware *stdlib.Middleware, excludeList []string) *RateLimiterFilter {
	return &RateLimiterFilter{
		middleware:  middleware,
		excludeList: excludeList,
	}
}

func handleLimitIsReached(limiterContext limiter.Context, username string, isLimitedClient bool, context context.Context) error {
	if !isLimitedClient {
		return nil
	}

	log.C(context).Debugf("Request limit has been exceeded for client with key", username)
	restAsTime := time.Unix(limiterContext.Reset, 0)
	return &util.HTTPError{
		ErrorType:   "BadRequest",
		Description: fmt.Sprintf("The allowed request limit has been reached, please try again in %s", time.Until(restAsTime)),
		StatusCode:  http.StatusTooManyRequests,
	}
}

func isRateLimitedClient(userContext *web.UserContext, excludeList []string) (bool, error) {
	//don't restrict global users
	if userContext.AccessLevel == web.GlobalAccess || userContext.AccessLevel == web.AllTenantAccess {
		return false, nil
	}

	if userContext.AuthenticationType == web.Basic {
		platform := types.Platform{}
		err := userContext.Data(&platform)
		if err != nil {
			return false, err
		}

		//Skip global platforms
		if platform.Labels == nil {
			return false, nil
		}
	}

	if slice.StringsAnyEquals(excludeList, userContext.Name) {
		return false, nil
	}

	return true, nil
}

func (rl *RateLimiterFilter) Name() string {
	return "RateLimiterFilter"
}

func (rl *RateLimiterFilter) Run(request *web.Request, next web.Handler) (*web.Response, error) {
	userContext, isProtectedEndpoint := web.UserFromContext(request.Context())

	if !isProtectedEndpoint {
		//skip public endpoints - no user context found
		return next.Handle(request)
	}

	isLimitedClient, err := isRateLimitedClient(userContext, rl.excludeList)
	if err != nil {
		log.C(request.Context()).WithError(err).Errorf("unable to determine if client should be rate limited")
		return nil, err
	}

	limiterContext, err := rl.middleware.Limiter.Get(request.Context(), userContext.Name)
	if err != nil {
		return nil, err
	}

	if limiterContext.Remaining == 1 {
		log.C(request.Context()).Infof("is_limited_client:%s,client key:%s, X-RateLimit-Limit=%s,X-o-Remaining=%s,X-RateLimit-Reset=%s", userContext.Name, limiterContext.Limit, limiterContext.Reset, isLimitedClient)
	}

	if limiterContext.Reached {
		err := handleLimitIsReached(limiterContext, userContext.Name, isLimitedClient, request.Context())
		if err != nil {
			return nil, err
		}
	}

	resp, err := next.Handle(request)
	if err != nil {
		return nil, err
	}

	if request.IsResponseWriterHijacked() {
		return resp, err
	}

	if resp.Header == nil {
		resp.Header = http.Header{}
	}

	if isLimitedClient {
		limit := strconv.FormatInt(limiterContext.Limit, 10)
		reset := strconv.FormatInt(limiterContext.Reset, 10)
		remaining := strconv.FormatInt(limiterContext.Remaining, 10)
		log.C(request.Context()).Debugf("client key:%s, X-RateLimit-Limit=%s,X-o-Remaining=%s,X-RateLimit-Reset=%s", userContext.Name, limit, remaining, reset)
		resp.Header.Add("X-RateLimit-Limit", limit)
		resp.Header.Add("X-RateLimit-Remaining", remaining)
		resp.Header.Add("X-RateLimit-Reset", reset)
	}

	return resp, err
}

func (rl *RateLimiterFilter) FilterMatchers() []web.FilterMatcher {
	return []web.FilterMatcher{
		{
			Matchers: []web.Matcher{
				web.Path("/**"),
				web.Methods(http.MethodPost, http.MethodPatch, http.MethodGet, http.MethodDelete, http.MethodOptions),
			},
		},
	}
}
