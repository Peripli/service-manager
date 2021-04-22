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
	"fmt"
	"github.com/Peripli/service-manager/constant"
	"github.com/Peripli/service-manager/pkg/log"
	"github.com/Peripli/service-manager/pkg/query"
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/pkg/util"
	"github.com/Peripli/service-manager/storage"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
	"net/http"

	"github.com/Peripli/service-manager/pkg/web"
)

const (
	ReferenceInstanceFilterName        = "ReferenceInstanceFilter"
	ReferenceParametersAreNotSupported = "Reference service instance parameters are not supported"
)

// serviceInstanceVisibilityFilter ensures that the tenant making the provisioning/update request
// has the necessary visibilities - i.e. that he has the right to consume the requested plan.
type referenceInstanceFilter struct {
	repository       storage.TransactionalRepository
	tenantIdentifier string
}

func NewReferenceInstanceFilter(repository storage.TransactionalRepository, tenantIdentifier string) *referenceInstanceFilter {
	return &referenceInstanceFilter{
		repository:       repository,
		tenantIdentifier: tenantIdentifier,
	}
}

func (*referenceInstanceFilter) Name() string {
	return ReferenceInstanceFilterName
}

func (f *referenceInstanceFilter) Run(req *web.Request, next web.Handler) (*web.Response, error) {
	switch req.Request.Method {
	case http.MethodPost:
		return f.handleProvision(req, next)
	case http.MethodPatch:
		return f.handleServiceUpdate(req, next)
	}
	return nil, nil
}

func (*referenceInstanceFilter) FilterMatchers() []web.FilterMatcher {
	return []web.FilterMatcher{
		{
			Matchers: []web.Matcher{
				web.Path(web.ServiceInstancesURL + "/**"),
				web.Methods(http.MethodPost, http.MethodPatch),
			},
		},
	}
}

func (f *referenceInstanceFilter) handleProvision(req *web.Request, next web.Handler) (*web.Response, error) {
	ctx := req.Context()
	servicePlanID := gjson.GetBytes(req.Body, planIDProperty).Str
	isReferencePlan, err := f.isReferencePlan(ctx, servicePlanID)
	if err != nil {
		return nil, err
	}
	if !isReferencePlan {
		return next.Handle(req)
	}

	// Ownership validation
	callerTenantID := gjson.GetBytes(req.Body, "context."+f.tenantIdentifier).String()
	if callerTenantID != "" {
		err = f.validateOwnership(req)
		if err != nil {
			return nil, err
		}
	}

	parameters := gjson.GetBytes(req.Body, "parameters").Map()

	referencedInstanceID, exists := parameters[constant.ReferencedInstanceIDKey]
	// epsilontal todo: should we validate that the input is string? can be any object for example...
	if !exists {
		return nil, util.HandleInstanceSharingError(util.ErrMissingReferenceParameter, constant.ReferencedInstanceIDKey)
	}
	req.Body, err = sjson.SetBytes(req.Body, constant.ReferencedInstanceIDKey, referencedInstanceID.Str)
	if err != nil {
		return nil, err
	}
	_, err = f.isReferencedShared(ctx, referencedInstanceID.Str)
	if err != nil {
		return nil, err
	}
	return next.Handle(req)
}

func (f *referenceInstanceFilter) handleServiceUpdate(req *web.Request, next web.Handler) (*web.Response, error) {
	// we don't want to allow plan_id and/or parameters changes on a reference service instance
	resourceID := req.PathParams["resource_id"]
	if resourceID == "" {
		return next.Handle(req)
	}
	ctx := req.Context()

	dbInstanceObject, err := f.getObjectByID(ctx, types.ServiceInstanceType, resourceID)
	if err != nil {
		return next.Handle(req)
	}
	instance := dbInstanceObject.(*types.ServiceInstance)

	isReferencePlan, err := f.isReferencePlan(ctx, instance.ServicePlanID)
	if err != nil {
		return nil, err
	}
	if !isReferencePlan {
		return next.Handle(req)
	}

	_, err = f.isValidPatchRequest(req, instance)
	if err != nil {
		return nil, err
	}
	return next.Handle(req)
}

func (f *referenceInstanceFilter) isReferencedShared(ctx context.Context, referencedInstanceID string) (bool, error) {
	byID := query.ByField(query.EqualsOperator, "id", referencedInstanceID)
	dbReferencedObject, err := f.repository.Get(ctx, types.ServiceInstanceType, byID)
	if err != nil {
		return false, util.HandleStorageError(err, types.ServiceInstanceType.String())
	}
	referencedInstance := dbReferencedObject.(*types.ServiceInstance)

	if !*referencedInstance.Shared {
		return false, util.HandleInstanceSharingError(util.ErrReferencedInstanceNotShared, referencedInstanceID)
	}
	return true, nil
}

func (f *referenceInstanceFilter) isValidPatchRequest(req *web.Request, instance *types.ServiceInstance) (bool, error) {
	newPlanID := gjson.GetBytes(req.Body, planIDProperty).String()
	if instance.ServicePlanID != newPlanID {
		return false, util.HandleInstanceSharingError(util.ErrChangingPlanOfReferenceInstance, instance.Name)
	}

	parametersRaw := gjson.GetBytes(req.Body, "parameters").Raw
	if parametersRaw != "" {
		return false, util.HandleInstanceSharingError(util.ErrChangingParametersOfReferenceInstance, instance.Name)
	}

	return true, nil
}

func (f *referenceInstanceFilter) isReferencePlan(ctx context.Context, servicePlanID string) (bool, error) {
	dbPlanObject, err := f.getObjectByID(ctx, types.ServicePlanType, servicePlanID)
	if err != nil {
		return false, err
	}
	plan := dbPlanObject.(*types.ServicePlan)
	return plan.Name == constant.ReferencePlanName, nil
}

func (f *referenceInstanceFilter) getObjectByID(ctx context.Context, objectType types.ObjectType, resourceID string) (types.Object, error) {
	byID := query.ByField(query.EqualsOperator, "id", resourceID)
	dbObject, err := f.repository.Get(ctx, objectType, byID)
	if err != nil {
		return nil, util.HandleStorageError(err, objectType.String())
	}
	return dbObject, nil
}

func (f *referenceInstanceFilter) validateOwnership(req *web.Request) error {
	ctx := req.Context()
	callerTenantID := gjson.GetBytes(req.Body, "context."+f.tenantIdentifier).String()
	path := fmt.Sprintf("parameters.%s", constant.ReferencedInstanceIDKey)
	referencedInstanceID := gjson.GetBytes(req.Body, path).String()
	byID := query.ByField(query.EqualsOperator, "id", referencedInstanceID)
	dbReferencedObject, err := f.repository.Get(ctx, types.ServiceInstanceType, byID)
	if err != nil {
		return util.HandleStorageError(err, types.ServiceInstanceType.String())
	}
	instance := dbReferencedObject.(*types.ServiceInstance)
	referencedOwnerTenantID := instance.Labels[f.tenantIdentifier][0]

	if referencedOwnerTenantID != callerTenantID {
		log.C(ctx).Errorf("Instance owner %s is not the same as the caller %s", referencedOwnerTenantID, callerTenantID)
		return &util.HTTPError{
			ErrorType:   "UnsupportedContextUpdate",
			Description: "could not find such service instance",
			StatusCode:  http.StatusBadRequest,
		}
	}
	return nil
}
