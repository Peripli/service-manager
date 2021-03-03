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
	"encoding/json"
	"errors"
	"github.com/Peripli/service-manager/pkg/log"
	"github.com/Peripli/service-manager/pkg/query"
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/pkg/util"
	"github.com/gofrs/uuid"
	"github.com/tidwall/gjson"
	"net/http"
	"time"

	"github.com/Peripli/service-manager/storage"

	"github.com/Peripli/service-manager/pkg/web"
)

const ReferenceInstanceFilterName = "ReferenceInstanceFilter"

// serviceInstanceVisibilityFilter ensures that the tenant making the provisioning/update request
// has the necessary visibilities - i.e. that he has the right to consume the requested plan.
type referenceInstanceFilter struct {
	repository storage.TransactionalRepository
}

func NewReferenceInstanceFilter(repository storage.TransactionalRepository) *referenceInstanceFilter {
	return &referenceInstanceFilter{
		repository: repository,
	}
}

func (*referenceInstanceFilter) Name() string {
	return ReferenceInstanceFilterName
}

func (f *referenceInstanceFilter) Run(req *web.Request, next web.Handler) (*web.Response, error) {
	referencedKey := "referenced_instance_id"
	parameters := gjson.GetBytes(req.Body, "parameters").Map()
	referencedInstanceID, exists := parameters[referencedKey]
	if !exists {
		return next.Handle(req)
	}

	ctx := req.Context()

	planID := gjson.GetBytes(req.Body, planIDProperty).String()

	byID := query.ByField(query.EqualsOperator, "id", planID)
	planObject, err := f.repository.Get(ctx, types.ServicePlanType, byID)
	if err != nil {
		return nil, util.HandleStorageError(err, types.ServicePlanType.String())
	}
	plan := planObject.(*types.ServicePlan)

	if isReferencePlan(plan) {
		// set as !isReferencePlan
		return nil, errors.New("plan_id is not a reference plan")
	}

	byID = query.ByField(query.EqualsOperator, "id", referencedInstanceID.Str)
	referencedObject, err := f.repository.Get(ctx, types.ServiceInstanceType, byID)
	if err != nil {
		return nil, util.HandleStorageError(err, types.ServiceInstanceType.String())
	}
	instance := referencedObject.(*types.ServiceInstance)

	if !instance.Shared {
		return nil, errors.New("referenced instance is not shared")
	}

	//instanceRequestBody := decodeRequestToObject(req.Body)
	err = f.createReferenceInstance(ctx, f.generateReferenceInstance(req.Body))
	if err != nil {
		return nil, util.HandleStorageError(err, types.ServiceInstanceType.String())
	}

	return util.NewJSONResponse(http.StatusCreated, instance)
}

func decodeRequestToObject(body []byte) provisionRequest {
	var instanceRequest = provisionRequest{}
	util.BytesToObject(body, instanceRequest)
	return instanceRequest
}

func isReferencePlan(plan *types.ServicePlan) bool {
	return plan.Name == "reference-plan"
}

func (f *referenceInstanceFilter) createReferenceInstance(ctx context.Context, instance *types.ServiceInstance) error {
	sharingErr := f.repository.InTransaction(ctx, func(ctx context.Context, storage storage.Repository) error {
		_, err := storage.Create(ctx, instance)
		if err != nil {
			log.C(ctx).Errorf("Could not update shared property for instance (%s): %v", instance.ID, err)
			return err
		}
		return nil
	})
	return sharingErr
}

func (*referenceInstanceFilter) FilterMatchers() []web.FilterMatcher {
	return []web.FilterMatcher{
		{
			Matchers: []web.Matcher{
				web.Path(web.ServiceInstancesURL + "/**"),
				web.Methods(http.MethodPost, http.MethodPatch, http.MethodDelete),
			},
		},
	}
}

type provisionRequest struct {
	Name               string          `json:"name"`
	PlanID             string          `json:"plan_id"`
	PlatformID         string          `json:"platform_id"`
	RawContext         json.RawMessage `json:"context"`
	RawMaintenanceInfo json.RawMessage `json:"maintenance_info"`
}

func (f *referenceInstanceFilter) generateReferenceInstance(body []byte) *types.ServiceInstance {

	UUID, _ := uuid.NewV4()
	currentTime := time.Now().UTC()

	instance := &types.ServiceInstance{
		Base: types.Base{
			ID:        UUID.String(),
			CreatedAt: currentTime,
			UpdatedAt: currentTime,
			Labels:    make(map[string][]string),
			Ready:     true,
		},
		Name:                 gjson.GetBytes(body, "name").String(),
		ServicePlanID:        gjson.GetBytes(body, "service_plan_id").String(),
		PlatformID:           gjson.GetBytes(body, "platform_id").String(),
		MaintenanceInfo:      json.RawMessage(gjson.GetBytes(body, "maintenance_info").String()),
		Context:              json.RawMessage(gjson.GetBytes(body, "contextt").String()),
		Usable:               true,
		ReferencedInstanceID: gjson.GetBytes(body, "parameters.referenced_instance_id").String(),
	}

	return instance
}
