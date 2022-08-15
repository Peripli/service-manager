package filters

import (
	"net/http"

	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/types"

	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/query"
	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/web"
)

const ProtectedSMPlatformFilterName = "ProtectedSMPlatformFilter"

// ProtectedSMPlatformFilter disallows patching and deleting of the service manager platform
type ProtectedSMPlatformFilter struct {
}

func (f *ProtectedSMPlatformFilter) Name() string {
	return ProtectedSMPlatformFilterName
}

func (f *ProtectedSMPlatformFilter) Run(req *web.Request, next web.Handler) (*web.Response, error) {
	ctx := req.Context()
	byName := query.ByField(query.NotEqualsOperator, "name", types.SMPlatform)
	ctx, err := query.AddCriteria(ctx, byName)
	if err != nil {
		return nil, err
	}
	req.Request = req.WithContext(ctx)

	return next.Handle(req)
}

func (f *ProtectedSMPlatformFilter) FilterMatchers() []web.FilterMatcher {
	return []web.FilterMatcher{
		{
			Matchers: []web.Matcher{
				web.Path(web.PlatformsURL + "/*"),
				web.Methods(http.MethodPatch, http.MethodDelete),
			},
		},
	}
}
