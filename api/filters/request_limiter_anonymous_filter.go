package filters

import (
	"github.com/Peripli/service-manager/pkg/web"
	"github.com/ulule/limiter"
	"github.com/ulule/limiter/drivers/middleware/stdlib"
	"net/http"
)

type AnonymousRequestLimiterFilter struct {
	middleware *stdlib.Middleware
	nodes      int64
}

func (rl *AnonymousRequestLimiterFilter) Name() string {
	return "request_limiter_anonymous"
}

func NewAnonymousRequestLimiterFilter(middleware *stdlib.Middleware, nodes int64) *AnonymousRequestLimiterFilter {
	return &AnonymousRequestLimiterFilter{
		middleware: middleware,
		nodes:      nodes,
	}
}

func (rl *AnonymousRequestLimiterFilter) Run(request *web.Request, next web.Handler) (*web.Response, error) {
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

func (rl *AnonymousRequestLimiterFilter) FilterMatchers() []web.FilterMatcher {
	return []web.FilterMatcher{
		{
			Matchers: []web.Matcher{
				web.Path("/**"),
				web.Methods(http.MethodPost, http.MethodPatch, http.MethodGet, http.MethodDelete, http.MethodOptions),
			},
		},
	}
}
