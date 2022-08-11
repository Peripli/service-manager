package filters

import (
	"fmt"
	"github.com/Peripli/service-manager/pkg/log"
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/pkg/util"
	"github.com/Peripli/service-manager/pkg/util/slice"
	"github.com/Peripli/service-manager/pkg/web"
	"github.com/ulule/limiter"
	"github.com/ulule/limiter/drivers/middleware/stdlib"
	"net/http"
	"strings"
)

type RateLimiterFilter struct {
	rateLimiters      []RateLimiterMiddleware
	excludeClients    []string
	excludePaths      []string
	tenantLabelKey    string
	usageLogThreshold int64
}

type RateLimiterMiddleware struct {
	middleware *stdlib.Middleware
	pathPrefix string
	method     string
	rate       limiter.Rate
}

func NewRateLimiterMiddleware(middleware *stdlib.Middleware, pathPrefix string, method string, rate limiter.Rate) RateLimiterMiddleware {
	return RateLimiterMiddleware{
		middleware,
		pathPrefix,
		method,
		rate,
	}
}

func NewRateLimiterFilter(middleware []RateLimiterMiddleware, excludeClients, excludePaths []string, usageLogThreshold int64, tenantLabelKey string) *RateLimiterFilter {
	return &RateLimiterFilter{
		rateLimiters:      middleware,
		excludeClients:    excludeClients,
		excludePaths:      excludePaths,
		usageLogThreshold: usageLogThreshold,
		tenantLabelKey:    tenantLabelKey,
	}
}

func (rl *RateLimiterFilter) handleLimitIsReached(limiterContext limiter.Context) error {
	return &util.HTTPError{
		ErrorType:   "BadRequest",
		Description: fmt.Sprintf("The allowed request limit of %d requests has been reached please try again later", limiterContext.Limit),
		StatusCode:  http.StatusTooManyRequests,
	}
}

func (rl *RateLimiterFilter) isRateLimitedClient(userContext *web.UserContext) (bool, error) {
	//don't restrict global users
	if userContext.AccessLevel == web.GlobalAccess || userContext.AccessLevel == web.AllTenantAccess {
		return false, nil
	}

	excludeByName := userContext.Name
	if userContext.AuthenticationType == web.Basic {
		platform := types.Platform{}
		err := userContext.Data(&platform)
		if err != nil {
			return false, err
		}

		if _, isTenantScopedPlatform := platform.Labels[rl.tenantLabelKey]; !isTenantScopedPlatform {
			return false, nil
		}

		excludeByName = platform.Name
	}

	if slice.StringsAnyEquals(rl.excludeClients, excludeByName) {
		return false, nil
	}

	return true, nil
}

func (rl *RateLimiterFilter) Name() string {
	return "RateLimiterFilter"
}

func (rl *RateLimiterFilter) isExcludedPath(path string) bool {
	for _, prefix := range rl.excludePaths {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}
	return false
}

func (rl *RateLimiterFilter) Run(request *web.Request, next web.Handler) (*web.Response, error) {
	userContext, isProtectedEndpoint := web.UserFromContext(request.Context())

	if !isProtectedEndpoint || rl.isExcludedPath(request.URL.Path) {
		//skip public endpoints or excluded prefix's
		return next.Handle(request)
	}

	isLimitedClient, err := rl.isRateLimitedClient(userContext)
	if err != nil {
		log.C(request.Context()).WithError(err).Errorf("unable to determine if client should be rate limited")
		return nil, err
	}

	if isLimitedClient {
		for _, rlm := range rl.rateLimiters {
			if !strings.HasPrefix(request.URL.Path, rlm.pathPrefix) {
				continue
			}
			if rlm.method != "" && strings.ToUpper(rlm.method) != strings.ToUpper(request.Method) {
				continue
			}
			method := "all-methods"
			if rlm.method != "" {
				method = rlm.method
			}
			key := method + ":" + rlm.pathPrefix + ":" + rlm.rate.Formatted + ":" + userContext.Name
			limiterContext, err := rlm.middleware.Limiter.Get(request.Context(), key)
			if err != nil {
				log.C(request.Context()).Errorf("failed to get limiter context with key %s: %v", key, err)
				return nil, err
			}

			// Log the clients that reach half of the allowed limit
			if limiterContext.Remaining == limiterContext.Limit-(limiterContext.Limit/rl.usageLogThreshold) {
				log.C(request.Context()).Infof("the client has already used %d percents of its rate limit quota, is_limited_client: %t, client key: %s, path prefix: %s, X-RateLimit-Limit=%d, X-o-Remaining=%d, X-RateLimit-Reset=%d", rl.usageLogThreshold, isLimitedClient, userContext.Name, rlm.pathPrefix, limiterContext.Limit, limiterContext.Remaining, limiterContext.Reset)
			}

			if limiterContext.Reached {
				log.C(request.Context()).Infof("Request limit has been exceeded for client with key: %s and path: %s", userContext.Name, rlm.pathPrefix)
				err := rl.handleLimitIsReached(limiterContext)
				if err != nil {
					return nil, err
				}
			}
		}
	}

	return next.Handle(request)
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
