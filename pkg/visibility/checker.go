package visibility

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/Peripli/service-manager/pkg/log"
	"github.com/Peripli/service-manager/pkg/query"
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/pkg/util"
	"github.com/Peripli/service-manager/pkg/web"
	"github.com/Peripli/service-manager/storage"
	"github.com/tidwall/gjson"
)

type Checker struct {
	repository   storage.Repository
	platformType string
	labelKey     string
}

func NewChecker(repository storage.Repository, platformType, labelKey string) *Checker {
	return &Checker{
		repository:   repository,
		platformType: platformType,
		labelKey:     labelKey,
	}
}

func (vc *Checker) CheckVisibility(req *web.Request, platform *types.Platform, planID string, osbContext json.RawMessage) error {
	ctx := req.Context()

	byPlanID := query.ByField(query.EqualsOperator, "service_plan_id", planID)
	visibilitiesList, err := vc.repository.List(ctx, types.VisibilityType, byPlanID)
	if err != nil {
		return util.HandleStorageError(err, string(types.VisibilityType))
	}
	visibilities := visibilitiesList.(*types.Visibilities)

	switch platform.Type {
	case vc.platformType:
		if vc.labelKey == "" {
			break
		}
		if len(osbContext) == 0 {
			log.C(ctx).Errorf("Could not find context in the osb request.")
			return &util.HTTPError{
				ErrorType:   "BadRequest",
				Description: "missing context in request body",
				StatusCode:  http.StatusBadRequest,
			}
		}
		payloadOSBVisiblityKey := gjson.GetBytes(osbContext, vc.labelKey).String()
		if len(payloadOSBVisiblityKey) == 0 {
			log.C(ctx).Errorf("Could not find %s in the context of the osb request.", vc.labelKey)
			return &util.HTTPError{
				ErrorType:   "BadRequest",
				Description: fmt.Sprintf("%s missing in osb context", vc.labelKey),
				StatusCode:  http.StatusBadRequest,
			}
		}
		for _, v := range visibilities.Visibilities {
			if v.PlatformID == "" { // public visibility
				return nil
			}
			if v.PlatformID == platform.ID {
				if v.Labels == nil { // tenant-scoped platform
					return nil
				}
				specialVisiblityLabels, ok := v.Labels[vc.labelKey]
				if !ok { // tenant-scoped platform
					return nil
				}
				for _, visLabelValue := range specialVisiblityLabels {
					if payloadOSBVisiblityKey == visLabelValue {
						return nil
					}
				}
			}
		}
		log.C(ctx).Errorf("Service plan %v is not visible on platform %v", planID, platform.ID)
		return &util.HTTPError{
			ErrorType:   "NotFound",
			Description: "could not find such service plan",
			StatusCode:  http.StatusNotFound,
		}
	}

	for _, v := range visibilities.Visibilities {
		if v.PlatformID == "" { // public visibility
			return nil
		}
		if v.PlatformID == platform.ID {
			return nil
		}
	}
	log.C(ctx).Errorf("Service plan %v is not visible on platform %v", planID, platform.ID)
	return &util.HTTPError{
		ErrorType:   "NotFound",
		Description: "could not find such service plan",
		StatusCode:  http.StatusNotFound,
	}
}
