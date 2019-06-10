package label

import (
	"fmt"
	"net/http"

	"github.com/Peripli/service-manager/pkg/types"

	"github.com/Peripli/service-manager/pkg/query"
	"github.com/Peripli/service-manager/pkg/util"
	"github.com/Peripli/service-manager/pkg/web"
	"github.com/tidwall/gjson"
)

// DenyLabelOperations describe denied label key operations
type DenyLabelOperations map[string][]query.LabelOperation

type FilterOperations struct {
	deniedOperations DenyLabelOperations
}

func NewFilterOperations(deniedOperations DenyLabelOperations) *FilterOperations {
	return &FilterOperations{
		deniedOperations: deniedOperations,
	}
}

func (flo *FilterOperations) Name() string {
	return "FilterOperations"
}

func (flo *FilterOperations) Run(req *web.Request, next web.Handler) (*web.Response, error) {
	if req.Method == http.MethodPost {
		bodyMap := gjson.ParseBytes(req.Body).Map()
		labelsBytes := []byte(bodyMap["labels"].String())
		if len(labelsBytes) == 0 {
			return next.Handle(req)
		}

		labels := types.Labels{}
		if err := util.BytesToObject(labelsBytes, &labels); err != nil {
			return nil, err
		}
		for lKey := range labels {
			labelOperations, found := flo.deniedOperations[lKey]
			if !found {
				continue
			}
			for _, lo := range labelOperations {
				if lo == query.AddLabelOperation || lo == query.AddLabelValuesOperation {
					return nil, &util.HTTPError{
						ErrorType:   "BadRequest",
						Description: fmt.Sprintf("Set/Add values for label %s is not allowed", lKey),
						StatusCode:  http.StatusBadRequest,
					}
				}
			}
		}
	} else if req.Method == http.MethodPatch {
		labelChanges, err := query.LabelChangesFromJSON(req.Body)
		if err != nil {
			return nil, err
		}
		for _, lc := range labelChanges {
			deniedLabelOperations, found := flo.deniedOperations[lc.Key]
			if !found {
				continue
			}
			for _, op := range deniedLabelOperations {
				if op == lc.Operation {
					return nil, &util.HTTPError{
						ErrorType:   "BadRequest",
						Description: fmt.Sprintf("Operation %s is not allowed for label %s", lc.Operation, lc.Key),
						StatusCode:  http.StatusBadRequest,
					}
				}
			}
		}
	}
	return next.Handle(req)
}

func (flo *FilterOperations) FilterMatchers() []web.FilterMatcher {
	return []web.FilterMatcher{
		{
			Matchers: []web.Matcher{
				web.Path(web.ServiceBrokersURL + "/**"),
			},
		},
		{
			Matchers: []web.Matcher{
				web.Path(web.PlatformsURL + "/**"),
			},
		},
	}
}
