package filters

import (
	"github.com/Peripli/service-manager/pkg/util"
	"github.com/Peripli/service-manager/pkg/web"
	"net/http"
	"strings"
)

const (
	ForceDeleteFilterName                = "ForceDeleteValidationFilter"
	FailOnNotAllowedUrlDescription       = "Force deletion can only be executed for instances or bindings"
	FailOnBadFlagsCombinationDescription = "Combination of cascade 'false' and force 'true' is not allowed"
)

type ForceDeleteValidationFilter struct {
}

func (s *ForceDeleteValidationFilter) Name() string {
	return ForceDeleteFilterName
}
func (s *ForceDeleteValidationFilter) FilterMatchers() []web.FilterMatcher {
	return []web.FilterMatcher{
		{
			Matchers: []web.Matcher{
				web.Methods(http.MethodDelete),
			},
		},
	}
}
func (s *ForceDeleteValidationFilter) Run(request *web.Request, next web.Handler) (*web.Response, error) {
	forceDelete := request.URL.Query().Get(web.QueryParamForce) == "true"
	cascadeDelete := request.URL.Query().Get(web.QueryParamCascade) == "true"

	if !forceDelete {
		return next.Handle(request)
	}

	if !s.validateAllowedUrls(request.URL.Path) {
		return nil, &util.HTTPError{
			ErrorType:   "BadRequest",
			Description: FailOnNotAllowedUrlDescription,
			StatusCode:  http.StatusBadRequest,
		}
	}

	if forceDelete && !cascadeDelete {
		return nil, &util.HTTPError{
			ErrorType:   "BadRequest",
			Description: FailOnBadFlagsCombinationDescription,
			StatusCode:  http.StatusBadRequest,
		}
	}

	return next.Handle(request)
}

func (s *ForceDeleteValidationFilter) validateAllowedUrls(path string) bool {
	if strings.Contains(path, web.ServiceInstancesURL) || strings.Contains(path, web.ServiceBindingsURL) {
		return true
	}
	return false
}
