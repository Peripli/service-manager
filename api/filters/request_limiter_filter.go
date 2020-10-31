package filters

import (
	"errors"
	"fmt"
	"github.com/Peripli/service-manager/pkg/util"
	"github.com/Peripli/service-manager/pkg/web"
	"github.com/ulule/limiter/drivers/middleware/stdlib"
	"net/http"
	"strconv"
	"time"
)

type RequestLimiterFilter struct {
	middleware *stdlib.Middleware
	nodes      int64
}

func NewRequestLimiterFilter(middleware *stdlib.Middleware, nodes int64) *RequestLimiterFilter {
	return &RequestLimiterFilter{
		middleware: middleware,
		nodes:      nodes,
	}
}

func handleLimitIsReached(resetTime int64) (*web.Response, error) {
	restAsTime := time.Unix(resetTime, 0)
	return nil, &util.HTTPError{
		ErrorType:   "BadRequest",
		Description: fmt.Sprintf("The allowed request limit has been reached, please try again in %s", time.Until(restAsTime)),
		StatusCode:  http.StatusTooManyRequests,
	}
}

func getLimiterKey(webReq *web.Request) (string, error) {
	user, ok := web.UserFromContext(webReq.Context())
	if !ok {
		return "", errors.New("unable to find user context")
	}

	return user.Name, nil
}

func (rl *RequestLimiterFilter) Name() string {
	return "request_limiter"
}

func (rl *RequestLimiterFilter) Run(request *web.Request, next web.Handler) (*web.Response, error) {
	limitByKey, err := getLimiterKey(request)

	if err != nil {
		return nil, err
	}

	limiterContext, err := rl.middleware.Limiter.Get(request.Context(), limitByKey)

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

	if request.IsResponseWriterHijacked() {
		return resp, err
	}

	if resp.Header == nil {
		resp.Header = http.Header{}
	}

	resp.Header.Add("X-RateLimit-Limit", strconv.FormatInt(limiterContext.Limit, 10))
	resp.Header.Add("X-RateLimit-Remaining", strconv.FormatInt(limiterContext.Remaining*rl.nodes, 10))
	resp.Header.Add("X-RateLimit-Reset", strconv.FormatInt(limiterContext.Reset, 10))
	return resp, err
}

func (rl *RequestLimiterFilter) FilterMatchers() []web.FilterMatcher {
	return []web.FilterMatcher{
		{
			Matchers: []web.Matcher{
				web.Path("/**"),
				web.Methods(http.MethodPost, http.MethodPatch, http.MethodGet, http.MethodDelete, http.MethodOptions),
			},
		},
	}
}
