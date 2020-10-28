package filters

import (
	"fmt"
	"github.com/Peripli/service-manager/pkg/util"
	"github.com/Peripli/service-manager/pkg/web"
	"github.com/ulule/limiter"
	"github.com/ulule/limiter/drivers/middleware/stdlib"
	"net/http"
	"strconv"
	"time"
)

type RequestLimiterFilter struct {
	middleware *stdlib.Middleware
	nodes      int
}

func NewRequestLimiterFilter(middleware *stdlib.Middleware, nodes int) *RequestLimiterFilter {
	return &RequestLimiterFilter{
		middleware: middleware,
		nodes:      nodes,
	}
}

func (rl *RequestLimiterFilter) getLimiterKey(webReq *web.Request) string {
	user, ok := web.UserFromContext(webReq.Context())
	if ok && len(user.Name) > 0 {
		return user.Name
	}

	return limiter.GetIPKey(webReq.Request, rl.middleware.TrustForwardHeader)
}

func (rl *RequestLimiterFilter) Name() string {
	return "request_limiter"
}

func (rl *RequestLimiterFilter) Run(request *web.Request, next web.Handler) (*web.Response, error) {

	key := rl.getLimiterKey(request)
	limiterContext, err := rl.middleware.Limiter.Get(request.Context(), key)
	if err != nil {
		return nil, err
	}

	resetTimestamp := strconv.FormatInt(limiterContext.Reset, 10)

	if limiterContext.Reached {
		resetTime, err := strconv.ParseInt(resetTimestamp, 10, 64)
		if err != nil {
			panic(err)
		}
		tm := time.Unix(resetTime, 0)

		return nil, &util.HTTPError{
			ErrorType:   "BadRequest",
			Description: fmt.Sprintf("The allowed request limit has been reached, please try again in %s", tm.Sub(time.Now())),
			StatusCode:  http.StatusTooManyRequests,
		}
	}

	resp, err := next.Handle(request)

	if err != nil {
		return nil, err
	}

	requestsLimit := strconv.FormatInt(limiterContext.Limit, 10)
	remainingRequests := strconv.FormatInt(limiterContext.Remaining/int64(rl.nodes), 10)

	resp.Header.Add("X-RateLimit-Limit", requestsLimit)
	resp.Header.Add("X-RateLimit-Remaining", remainingRequests)
	resp.Header.Add("X-RateLimit-Reset", resetTimestamp)

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
