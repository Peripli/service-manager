package filters

import (
	"context"
	"fmt"
	"github.com/Peripli/service-manager/pkg/log"
	"github.com/Peripli/service-manager/pkg/util"
	"github.com/Peripli/service-manager/pkg/util/slice"
	"github.com/Peripli/service-manager/pkg/web"
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

func handleLimitIsReached(resetTime int64, limitByKey string, context context.Context) (*web.Response, error) {
	log.C(context).Info("Request limit has been exceeded for client with key", limitByKey)
	restAsTime := time.Unix(resetTime, 0)
	return nil, &util.HTTPError{
		ErrorType:   "BadRequest",
		Description: fmt.Sprintf("The allowed request limit has been reached, please try again in %s", time.Until(restAsTime)),
		StatusCode:  http.StatusTooManyRequests,
	}
}

func getLimiterKey(request *web.Request, excludeList []string) (string, bool) {
	user, ok := web.UserFromContext(request.Context())

	//don't restrict global users
	if user.AccessLevel == web.GlobalAccess || user.AccessLevel == web.AllTenantAccess {
		return "", false
	}

	if !ok {
		//don't restrict public endpoints
		return "", false
	}

	if slice.StringsAnyEquals(excludeList, user.Name) {
		return "", false
	}

	return user.Name, true
}

func (rl *RateLimiterFilter) Name() string {
	return "RateLimiterFilter"
}

func (rl *RateLimiterFilter) Run(request *web.Request, next web.Handler) (*web.Response, error) {
	limitByKey, shouldRateLimit := getLimiterKey(request, rl.excludeList)

	//we skip exclude list, public endpoints and global or sub-account all scopes
	if !shouldRateLimit {
		return next.Handle(request)
	}

	limiterContext, err := rl.middleware.Limiter.Get(request.Context(), limitByKey)

	if err != nil {
		return nil, err
	}

	if limiterContext.Reached {
		return handleLimitIsReached(limiterContext.Reset, limitByKey, request.Context())
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

	limit := strconv.FormatInt(limiterContext.Limit, 10)
	reset := strconv.FormatInt(limiterContext.Reset, 10)

	log.C(request.Context()).Debugf("client key:%s, X-RateLimit-Limit=%s,X-RateLimit-Remaining=%s,X-RateLimit-Reset=%n", limitByKey, limit, reset)

	resp.Header.Add("X-RateLimit-Limit", limit)
	resp.Header.Add("X-RateLimit-Reset", reset)
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
