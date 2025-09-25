package common

import (
	"github.com/Peripli/service-manager/pkg/web"
)

type OverrideFilter struct {
	ClientIP string
	UserName string
}

func (f OverrideFilter) Name() string {
	return "UserContextFilter"
}

func (f OverrideFilter) Run(request *web.Request, next web.Handler) (*web.Response, error) {
	if f.ClientIP != "" {
		request.Header.Set("X-Forwarded-For", f.ClientIP)
	}
	user, _ := web.UserFromContext(request.Context())

	if user != nil && f.UserName != "" {
		user.Name = f.UserName
	}

	return next.Handle(request)
}

func (f OverrideFilter) FilterMatchers() []web.FilterMatcher {
	return []web.FilterMatcher{
		{
			Matchers: []web.Matcher{
				web.Path("**/*"),
			},
		},
	}
}
