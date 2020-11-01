package filters

import (
	"github.com/Peripli/service-manager/pkg/web"
	"github.com/ulule/limiter"
	"github.com/ulule/limiter/drivers/middleware/stdlib"
	"net/http"
)

type RateLimiterAnonymousFilter struct {
	middleware *stdlib.Middleware
	nodes      int64
}

func (rl *RateLimiterAnonymousFilter) Name() string {
	return "RequestLimiterAnonymousFilter"
}

func NewRateLimiterAnonymousFilter(middleware *stdlib.Middleware, nodes int64) *RateLimiterAnonymousFilter {
	return &RateLimiterAnonymousFilter{
		middleware: middleware,
		nodes:      nodes,
	}
}

func (rl *RateLimiterAnonymousFilter) Run(request *web.Request, next web.Handler) (*web.Response, error) {
	limiterContext, err := rl.middleware.Limiter.Peek(request.Context(), limiter.GetIPKey(request.Request, true))

	if err != nil {
		return nil, err
	}

	if limiterContext.Reached {
		return handleLimitIsReached(limiterContext.Reset)
	}

	resp, err := next.Handle(request)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode == http.StatusForbidden {
		_, err := rl.middleware.Limiter.Get(request.Context(), limiter.GetIPKey(request.Request, true))
		if err != nil {
			return nil, err
		}
	}

	return resp, err
}

func (rl *RateLimiterAnonymousFilter) FilterMatchers() []web.FilterMatcher {
	return []web.FilterMatcher{
		{
			Matchers: []web.Matcher{
				web.Path("/**"),
				web.Methods(http.MethodPost, http.MethodPatch, http.MethodGet, http.MethodDelete, http.MethodOptions),
			},
		},
	}
}
