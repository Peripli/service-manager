package filters

import (
	"github.com/Peripli/service-manager/pkg/web"
	"net/http"
)

const (
	RegeneratePlatformCredentialsFilterName = "RegeneratePlatformCredentialsFilter"
	RegenerateCredentialsQueryParam         = "regenerateCredentials"
	ActivateCredentialsQueryParam           = "activateCredentials"
)

// RegeneratePlatformCredentialsFilter checks if regenerate credentials for platform was required
type RegeneratePlatformCredentialsFilter struct {
}

func (f *RegeneratePlatformCredentialsFilter) Name() string {
	return RegeneratePlatformCredentialsFilterName
}

func (f *RegeneratePlatformCredentialsFilter) Run(req *web.Request, next web.Handler) (*web.Response, error) {

	if req.URL.Query().Get(RegenerateCredentialsQueryParam) == "true" {
		newCtx := web.ContextWithGeneratePlatformCredentialsFlag(req.Context(), true)
		req.Request = req.WithContext(newCtx)
	}

	if req.URL.Query().Get(ActivateCredentialsQueryParam) == "true" {
		newCtx := web.ContextWithActivatePlatformCredentialsFlag(req.Context(), true)
		req.Request = req.WithContext(newCtx)
	}
	return next.Handle(req)
}

func (f *RegeneratePlatformCredentialsFilter) FilterMatchers() []web.FilterMatcher {
	return []web.FilterMatcher{
		{
			Matchers: []web.Matcher{
				web.Path(web.PlatformsURL + "/*"),
				web.Methods(http.MethodPatch),
			},
		},
	}
}
