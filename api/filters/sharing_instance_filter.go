/*
 * Copyright 2018 The Service Manager Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package filters

import (
	"context"
	"errors"
	"github.com/Peripli/service-manager/pkg/query"
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/pkg/util"
	"github.com/Peripli/service-manager/storage"
	"github.com/Peripli/service-manager/storage/service_plans"
	"github.com/gofrs/uuid"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
	"net/http"
	"time"

	"github.com/Peripli/service-manager/pkg/log"
	"github.com/Peripli/service-manager/pkg/web"
)

const SharingInstanceFilterName = "SharingInstanceFilter"

// ServiceInstanceStripFilter checks patch request body for unmodifiable properties
type sharingInstanceFilter struct {
	repository        storage.TransactionalRepository
	storageRepository storage.Repository
	labelKey          string
}

func NewSharingInstanceFilter(repository storage.TransactionalRepository, storageRepository storage.Repository, labelKey string) *sharingInstanceFilter {
	return &sharingInstanceFilter{
		repository:        repository,
		storageRepository: storageRepository,
		labelKey:          labelKey,
	}
}

func (*sharingInstanceFilter) Name() string {
	return SharingInstanceFilterName
}

func (f *sharingInstanceFilter) Run(req *web.Request, next web.Handler) (*web.Response, error) {
	// Ignore the filter if has no shared property
	sharedBytes := gjson.GetBytes(req.Body, "shared")
	if len(sharedBytes.Raw) == 0 {
		return next.Handle(req)
	}

	ctx := req.Context()
	logger := log.C(ctx)

	instanceID := req.PathParams["resource_id"]
	shared := sharedBytes.Bool()

	// Get instance from database
	byID := query.ByField(query.EqualsOperator, "id", instanceID)
	instanceObject, err := f.repository.Get(ctx, types.ServiceInstanceType, byID)
	if err != nil {
		return nil, util.HandleStorageError(err, types.ServiceInstanceType.String())
	}
	instance := instanceObject.(*types.ServiceInstance)

	body := map[string]bool{}
	util.BytesToObject(req.Body, body)

	if !isSMPlatform(instance.PlatformID) && len(body) > 1 {
		return nil, errors.New("could not modify the 'shared' property with other changes at the same time")
	}

	planID := instance.ServicePlanID
	// get plan object from database, on service_instance patch flow
	byID = query.ByField(query.EqualsOperator, "id", planID)
	planObject, err := f.repository.Get(ctx, types.ServicePlanType, byID)
	if err != nil {
		return nil, util.HandleStorageError(err, types.ServicePlanType.String())
	}
	plan := planObject.(*types.ServicePlan)

	// Get instance from database
	//byID = query.ByField(query.EqualsOperator, "id", "reference-plan")
	//entityObject, err := f.repository.Get(ctx, types.ServiceInstanceType, byID)
	//if err != nil {
	//	return nil, util.HandleStorageError(err, types.Entity.String())
	//}
	//visibility := instanceObject.(*types.ServiceInstance)

	if shared && !plan.IsShareablePlan() {
		return nil, &util.UnsupportedQueryError{
			Message: "Plan is non-shared",
		}
	}

	if plan.IsShareablePlan() {
		err = f.shareInstance(ctx, instance, shared)
		// todo: return error to client
		if err != nil {
			logger.Errorf("Could not update shared property for instance (%s): %v", instanceID, err)
			return nil, err
		}

		plans := make([]*types.ServicePlan, 0)
		plans = append(plans, plan)

		platforms, _ := service_plans.ResolveSupportedPlatformsForPlans(ctx, plans, f.storageRepository)

		//additionalLabel := "org_id"

		err = f.setVisibilityOfReferencePlan(ctx, platforms)

		if err != nil {
			logger.Errorf("Could not set a visibility label of reference plan when sharing the instance (%s): %v", instanceID, err)
			return nil, err
		}

		if isSMPlatform(instance.PlatformID) {
			if req.Body, err = sjson.DeleteBytes(req.Body, "shared"); err != nil {
				return nil, err
			}
		} else {
			return util.NewJSONResponse(http.StatusOK, instance)
		}
	}

	return next.Handle(req)
}

func (f *sharingInstanceFilter) retrieveTenantID(ctx context.Context) (string, error) {
	tenantID := query.RetrieveFromCriteria(f.labelKey, query.CriteriaForContext(ctx)...)
	if tenantID == "" {
		log.C(ctx).Errorf("Tenant identifier not found in request criteria.")
		return "", &util.HTTPError{
			ErrorType:   "BadRequest",
			Description: "no tenant identifier provided",
			StatusCode:  http.StatusBadRequest,
		}
	}
	return tenantID, nil
}

func (*sharingInstanceFilter) FilterMatchers() []web.FilterMatcher {
	return []web.FilterMatcher{
		{
			Matchers: []web.Matcher{
				web.Path(web.ServiceInstancesURL + "/**"),
				web.Methods(http.MethodPatch),
			},
		},
	}
}

func (f *sharingInstanceFilter) shareInstance(ctx context.Context, instance *types.ServiceInstance, shared bool) error {
	logger := log.C(ctx)
	instance.Shared = shared
	sharingErr := f.repository.InTransaction(ctx, func(ctx context.Context, storage storage.Repository) error {
		_, err := storage.Update(ctx, instance, nil)
		if err != nil {
			logger.Errorf("Could not update shared property for instance (%s): %v", instance.ID, err)
			return err
		}
		return nil
	})
	return sharingErr
}

func (f *sharingInstanceFilter) setVisibilityOfReferencePlan(ctx context.Context, platformIDs map[string]*types.Platform) error {
	tenantID, err := f.retrieveTenantID(ctx)
	if err != nil {
		return err
	}
	type SharingInstanceOverrideLabels struct{}
	overrideLabelsObj := ctx.Value(SharingInstanceOverrideLabels{})
	var overrideLabels map[string]map[string][]string
	if overrideLabelsObj != nil {
		overrideLabels = overrideLabelsObj.(map[string]map[string][]string)
	}
	//else {
	//	overrideLabels = make(map[string]map[string][]string)
	//	var cfLabels = make(map[string][]string)
	//	var orgList []string
	//	cfLabels["organization_id"] = append(orgList, "org_id")
	//	overrideLabels["platform-type"] = cfLabels
	//}

	for _, platform := range platformIDs {
		var platformOverrideLabels map[string][]string
		for platformType := range overrideLabels {
			if platform.Type == platformType {
				platformOverrideLabels = overrideLabels[platformType]
			}
		}
		sharingErr := f.repository.InTransaction(ctx, func(ctx context.Context, storage storage.Repository) error {
			visibility := f.generateVisibility(platform.ID, "reference-plan", tenantID, platformOverrideLabels)
			_, err := storage.Create(ctx, visibility)
			if err != nil {
				return err
			}
			return nil
		})
		if sharingErr != nil {
			return sharingErr
		}
	}
	return nil
}

func (f *sharingInstanceFilter) generateVisibility(platformID, planID, tenantID string, overrideLabels map[string][]string) *types.Visibility {
	UUID, err := uuid.NewV4()
	if err != nil {
		//return fmt.Errorf("could not generate GUID for visibility: %s", err)
	}

	var labels types.Labels
	if overrideLabels != nil {
		labels = overrideLabels
	} else {
		labels = types.Labels{
			f.labelKey: {
				tenantID,
			},
		}
	}

	currentTime := time.Now().UTC()
	visibility := &types.Visibility{
		Base: types.Base{
			ID:        UUID.String(),
			UpdatedAt: currentTime,
			CreatedAt: currentTime,
			Ready:     true,
			Labels:    labels,
		},
		ServicePlanID: planID,
		PlatformID:    platformID,
	}

	return visibility
}

func isSMPlatform(platformID string) bool {
	return platformID == types.SMPlatform
}

type shareInstanceType struct {
	shared bool
}
