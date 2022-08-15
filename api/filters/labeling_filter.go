package filters

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/tidwall/sjson"

	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/query"

	"github.com/tidwall/gjson"

	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/log"

	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/web"
)

const LabelCriteriaFilterNameSuffix = "CriteriaFilter"
const ResourceLabelingFilterNameSuffix = "LabelingFilter"

// NewLabelingFilters returns set of filters which applies resource labeling rules
func NewLabelingFilters(labelName, labelKey string, basePaths []string, extractValueFunc func(request *web.Request) (string, error)) []web.Filter {
	return []web.Filter{
		newLabelCriteriaFilter(labelName, labelKey, basePaths, extractValueFunc),
		newResourceLabelingFilter(labelName, labelKey, basePaths, extractValueFunc),
	}
}

// newLabelCriteriaFilter creates a new LabelingFilter from the specified settings that filters the returned resources based on a filtering label
func newLabelCriteriaFilter(labelName, labelKey string, bastPaths []string, extractValueFunc func(request *web.Request) (string, error)) *LabelingFilter {
	return &LabelingFilter{
		LabelKey:     labelKey,
		FilterName:   labelName + LabelCriteriaFilterNameSuffix,
		BasePaths:    bastPaths,
		Methods:      []string{http.MethodGet, http.MethodPatch, http.MethodDelete, http.MethodPost},
		ExtractValue: extractValueFunc,
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

// newResourceLabelingFilter creates a new LabelingFilter from the specified settings that adds a filtering label when creating resources
func newResourceLabelingFilter(labelName, labelKey string, bastPaths []string, extractValueFunc func(request *web.Request) (string, error)) *LabelingFilter {
	return &LabelingFilter{
		LabelKey:     labelKey,
		FilterName:   labelName + ResourceLabelingFilterNameSuffix,
		BasePaths:    bastPaths,
		Methods:      []string{http.MethodPost},
		ExtractValue: extractValueFunc,
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

// LabelingFilter applies labeling on the resources based on extracted values
type LabelingFilter struct {
	// LabelKey is the key of the label
	LabelKey     string
	ExtractValue func(request *web.Request) (string, error)
	LabelingFunc func(request *web.Request, labelKey, labelValue string) error

	FilterName string
	BasePaths  []string
	Methods    []string
}

// Name implements web.Named and returns the filter name
func (f *LabelingFilter) Name() string {
	return f.FilterName
}

// Run implements web.Middleware and attempts to extract label value and apply labeling function
func (f *LabelingFilter) Run(request *web.Request, next web.Handler) (*web.Response, error) {
	if f.ExtractValue == nil {
		return nil, errors.New("ExtractValue function should be provided")
	}

	if f.LabelingFunc == nil {
		return nil, errors.New("LabelingFunc function should be provided")
	}

	value, err := f.ExtractValue(request)
	if err != nil {
		return nil, err
	}

	if len(value) == 0 {
		return next.Handle(request)
	}

	if err := f.LabelingFunc(request, f.LabelKey, value); err != nil {
		return nil, err
	}

	return next.Handle(request)
}

// FilterMatchers implements web.Filter.FilterMatchers and specifies that the filter should run on configured method
func (f *LabelingFilter) FilterMatchers() []web.FilterMatcher {
	matchers := make([]web.FilterMatcher, 0)
	for _, basePath := range f.BasePaths {
		matchers = append(matchers, web.FilterMatcher{
			Matchers: []web.Matcher{
				web.Path(basePath + "/**"),
				web.Methods(f.Methods...),
			},
		})
	}
	return matchers
}
