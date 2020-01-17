package filters

import (
	"encoding/json"
	"net/http"

	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/pkg/visibility"

	"github.com/Peripli/service-manager/pkg/query"
	"github.com/Peripli/service-manager/pkg/util"

	"github.com/Peripli/service-manager/pkg/web"
	"github.com/Peripli/service-manager/storage"
)

const CheckVisibilityFilterName = "CheckVisibilityFilter"

type checkVisibilityFilter struct {
	repository storage.Repository
	checker    *visibility.Checker
}

// NewCheckVisibilityFilter creates new filter that checks if a plan is visible to the user on provision request
func NewCheckVisibilityFilter(repository storage.Repository, checker *visibility.Checker) *checkVisibilityFilter {
	return &checkVisibilityFilter{
		repository: repository,
		checker:    checker,
	}
}

// Name returns the name of the plugin
func (*checkVisibilityFilter) Name() string {
	return CheckVisibilityFilterName
}

func (f *checkVisibilityFilter) Run(req *web.Request, next web.Handler) (*web.Response, error) {
	var instance struct {
		*types.ServiceInstance
		RawContext json.RawMessage `json:"context"`
	}
	err := util.BytesToObject(req.Body, instance)
	if err != nil {
		return nil, err
	}
	byID := query.ByField(query.EqualsOperator, "id", instance.PlatformID)
	platform, err := f.repository.Get(req.Context(), types.PlatformType, byID)
	if err != nil {
		return nil, util.HandleStorageError(err, string(types.PlatformType))
	}

	if err := f.checker.CheckVisibility(req, platform.(*types.Platform), instance.ServicePlanID, instance.RawContext); err != nil {
		return nil, err
	}
	return next.Handle(req)
}

func (f *checkVisibilityFilter) FilterMatchers() []web.FilterMatcher {
	return []web.FilterMatcher{
		web.FilterMatcher{
			Matchers: []web.Matcher{
				web.Path(web.ServiceInstancesURL),
				web.Methods(http.MethodPost),
			},
		},
	}
}
