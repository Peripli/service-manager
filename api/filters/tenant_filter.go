package filters

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/tidwall/sjson"

	"github.com/Peripli/service-manager/pkg/query"

	"github.com/tidwall/gjson"

	"github.com/Peripli/service-manager/pkg/log"

	"github.com/Peripli/service-manager/pkg/web"
)

// sm-dev adopt and add new config properties in the dev en
const TenantCriteriaFilterName = "TenantCriteriaFilter"
const TenantLabelingFilterName = "TenantLabelingFilter"

// NewMultitenancyFilters returns set of filters which applies multitenancy rules
func NewMultitenancyFilters(labelKey string, extractTenantFunc func(request *web.Request) (string, error)) []web.Filter {
	return []web.Filter{
		newTenantCriteriaFilter(labelKey, extractTenantFunc),
		newTenantLabelingFilter(labelKey, extractTenantFunc),
	}
}

// newTenantCriteriaFilter creates a new TenantFilter from the specified settings that filters the returned resources based on a filtering label
func newTenantCriteriaFilter(labelKey string, extractTenantFunc func(request *web.Request) (string, error)) *TenantFilter {
	return &TenantFilter{
		LabelKey:      labelKey,
		FilterName:    TenantCriteriaFilterName,
		Methods:       []string{http.MethodGet, http.MethodPatch, http.MethodDelete},
		ExtractTenant: extractTenantFunc,
		LabelingFunc: func(request *web.Request, labelKey, labelValue string) error {
			ctx := request.Context()
			criterion := query.ByLabel(query.EqualsOperator, labelKey, labelValue)
			var err error
			ctx, err = query.AddCriteria(ctx, criterion)
			if err != nil {
				return fmt.Errorf("could not add label criteria with key %s and value %s: %s", labelKey, labelValue, err)
			}

			log.C(ctx).Infof("Successfully added label criteria with key %s and value %s to context", labelKey, labelValue)
			request.Request = request.WithContext(ctx)

			return nil
		},
	}
}

// newTenantLabelingFilter creates a new TenantFilter from the specified settings that adds a filtering label when creating resources
func newTenantLabelingFilter(labelKey string, extractTenantFunc func(request *web.Request) (string, error)) *TenantFilter {
	return &TenantFilter{
		LabelKey:      labelKey,
		FilterName:    TenantLabelingFilterName,
		Methods:       []string{http.MethodPost},
		ExtractTenant: extractTenantFunc,
		LabelingFunc: func(request *web.Request, labelKey, labelValue string) error {
			ctx := request.Context()
			currentLabelValues := gjson.GetBytes(request.Body, fmt.Sprintf("labels.%s", labelKey)).Raw
			var path string
			var obj interface{}
			if len(currentLabelValues) != 0 {
				path = fmt.Sprintf("labels.%s.-1", labelKey)
				obj = labelValue
			} else {
				path = fmt.Sprintf("labels.%s", labelKey)
				obj = []string{labelValue}
			}

			var err error
			request.Body, err = sjson.SetBytes(request.Body, path, obj)
			if err != nil {
				return fmt.Errorf("could not add label with key %s and value %s: %s to request body during resource creation", labelKey, labelValue, err)
			}

			log.C(ctx).Infof("Successfully added label with key %s and value %s to request body during resource creation", labelKey, labelValue)

			return nil
		},
	}
}

// TenantFilter applies multitenancy labeling on the resources based on extracted tenant
type TenantFilter struct {
	// LabelKey is the key of the label, the value of which will be used to apply multitenancy rules
	LabelKey string

	ExtractTenant func(request *web.Request) (string, error)
	LabelingFunc  func(request *web.Request, labelKey, labelValue string) error

	FilterName string
	Methods    []string
}

// Name implements web.Named and returns the filter name
func (f *TenantFilter) Name() string {
	return f.FilterName
}

// Run implements web.Middleware and attempts to extract tenant and apply labeling function
func (f *TenantFilter) Run(request *web.Request, next web.Handler) (*web.Response, error) {
	if f.ExtractTenant == nil {
		return nil, errors.New("ExtractTenant function should be provided")
	}

	if f.LabelingFunc == nil {
		return nil, errors.New("LabelingFunc function should be provided")
	}

	ctx := request.Context()

	userContext, found := web.UserFromContext(ctx)
	if !found {
		log.C(ctx).Infof("No user found in user context. Proceeding with empty tenant ID value...")
		return next.Handle(request)
	}
	if userContext.AccessLevel == web.GlobalAccess {
		log.C(ctx).Infof("Access level is Global. Proceeding with empty tenant ID value...")
		return next.Handle(request)
	}

	tenant, err := f.ExtractTenant(request)
	if err != nil {
		return nil, err
	}

	if len(tenant) == 0 {
		return next.Handle(request)
	}

	if err := f.LabelingFunc(request, f.LabelKey, tenant); err != nil {
		return nil, err
	}

	return next.Handle(request)
}

// FilterMatchers implements web.Filter.FilterMatchers and specifies that the filter should run on configured method
func (f *TenantFilter) FilterMatchers() []web.FilterMatcher {
	return []web.FilterMatcher{
		{
			Matchers: []web.Matcher{
				web.Path(web.ServiceBrokersURL + "/**"),
				web.Methods(f.Methods...),
			},
		},
		{
			Matchers: []web.Matcher{
				web.Path(web.PlatformsURL + "/**"),
				web.Methods(f.Methods...),
			},
		},
		{
			Matchers: []web.Matcher{
				web.Path(web.ServiceInstancesURL + "/**"),
				web.Methods(f.Methods...),
			},
		},
	}
}
