package filters

import (
	"errors"
	"fmt"
	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/log"
	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/web"
)

// sm-dev adopt and add new config properties in the dev en
const LabelName = "Tenant"

// NewMultitenancyFilters returns set of filters which applies multitenancy rules
func NewMultitenancyFilters(labelKey string, extractTenantFunc func(request *web.Request) (string, error)) ([]web.Filter, error) {
	if extractTenantFunc == nil {
		return nil, errors.New("extractTenantFunc should be provided")
	}

	return NewLabelingFilters(LabelName, labelKey, []string{web.PlatformsURL, web.ServiceBrokersURL, web.ServiceInstancesURL, web.ServiceBindingsURL}, func(request *web.Request) (string, error) {
		ctx := request.Context()

		userContext, found := web.UserFromContext(ctx)
		if !found {
			log.C(ctx).Debug("No user found in user context. Proceeding with empty tenant ID value...")
			return "", nil
		}
		if userContext.AccessLevel == web.GlobalAccess {
			log.C(ctx).Debug("Access level is Global. Proceeding with empty tenant ID value...")
			return "", nil
		}

		return extractTenantFunc(request)
	}), nil
}

//TenantLabelingFilterName returns the name of the filter that is adding the tenant label to tenant-scoped resources
func TenantLabelingFilterName() string {
	return fmt.Sprintf("%s%s", LabelName, ResourceLabelingFilterNameSuffix)
}
