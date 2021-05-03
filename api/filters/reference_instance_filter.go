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
	"github.com/Peripli/service-manager/pkg/instance_sharing"
	"github.com/Peripli/service-manager/pkg/log"
	"github.com/Peripli/service-manager/pkg/query"
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/pkg/util"
	"github.com/Peripli/service-manager/storage"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
	"net/http"
	"strings"

	"github.com/Peripli/service-manager/pkg/web"
)

const ReferenceInstanceFilterName = "ReferenceInstanceFilter"

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

func (rif *referenceInstanceFilter) Run(req *web.Request, next web.Handler) (*web.Response, error) {
	switch req.Request.Method {
	case http.MethodGet:
		if strings.Contains(req.RequestURI, "/parameters") {
			return rif.handleGetParameters(req, next)
		}
	case http.MethodPost:
		return rif.handleProvision(req, next)
	case http.MethodPatch:
		return rif.handleServiceUpdate(req, next)
	}
	return next.Handle(req)
}

func (rif *referenceInstanceFilter) handleGetParameters(req *web.Request, next web.Handler) (*web.Response, error) {
	ctx := req.Context()
	instanceID := req.PathParams[web.PathParamResourceID]

	dbInstanceObject, err := storage.GetObjectByField(ctx, rif.repository, types.ServiceInstanceType, "id", instanceID)
	if err != nil {
		return next.Handle(req)
	}
	instance := dbInstanceObject.(*types.ServiceInstance)
	if instance.ReferencedInstanceID == "" {
		return next.Handle(req)
	}

	body := map[string]string{
		instance_sharing.ReferencedInstanceIDKey: instance.ReferencedInstanceID,
	}

	return util.NewJSONResponse(http.StatusOK, body)
}

func (*referenceInstanceFilter) FilterMatchers() []web.FilterMatcher {
	return []web.FilterMatcher{
		{
			Matchers: []web.Matcher{
				web.Path(web.ServiceInstancesURL + "/**"),
				web.Methods(http.MethodPost, http.MethodPatch, http.MethodGet),
			},
		},
	}
}

func (rif *referenceInstanceFilter) handleProvision(req *web.Request, next web.Handler) (*web.Response, error) {
	ctx := req.Context()
	servicePlanID := gjson.GetBytes(req.Body, planIDProperty).String()
	isReferencePlan, err := storage.IsReferencePlan(ctx, rif.repository, types.ServicePlanType.String(), "id", servicePlanID)
	// If plan not found on provisioning, allow sm to handle the issue
	if err == util.ErrNotFoundInStorage {
		return next.Handle(req)
	}
	if err != nil {
		return nil, err // handled by the IsReference function
	}
	if !isReferencePlan {
		return next.Handle(req)
	}

	parameters := gjson.GetBytes(req.Body, "parameters").Map()

	referencedInstanceID, exists := parameters[instance_sharing.ReferencedInstanceIDKey]
	if !exists {
		return nil, util.HandleInstanceSharingError(util.ErrMissingOrInvalidReferenceParameter, instance_sharing.ReferencedInstanceIDKey)
	}

	req.Body, err = sjson.SetBytes(req.Body, instance_sharing.ReferencedInstanceIDKey, referencedInstanceID.String())
	if err != nil {
		log.C(ctx).Errorf("Unable to set the instance_sharing.ReferencedInstanceIDKey: \"%s\",error details: %s", referencedInstanceID, err)
		return nil, err
	}

	// retrieve instance
	dbTargetInstanceObject, err := storage.GetObjectByField(ctx, rif.repository, types.ServiceInstanceType, "id", referencedInstanceID.String())
	if err != nil {
		log.C(ctx).Errorf("Failed retrieving the reference-instance by the ID: %s", referencedInstanceID)
		return nil, util.HandleStorageError(util.ErrNotFoundInStorage, types.ServiceInstanceType.String())
	}
	targetInstance := dbTargetInstanceObject.(*types.ServiceInstance)

	// Ownership validation
	if rif.tenantIdentifier != "" {
		callerTenantID := query.RetrieveFromCriteria(rif.tenantIdentifier, query.CriteriaForContext(req.Context())...)
		err = storage.ValidateOwnership(targetInstance, rif.tenantIdentifier, req, callerTenantID)
		if err != nil {
			if err == util.ErrNotFoundInStorage {
				return nil, util.HandleInstanceSharingError(util.ErrMissingOrInvalidReferenceParameter, instance_sharing.ReferencedInstanceIDKey)
			}
			log.C(ctx).Errorf("Unable to Validate Ownership: \"%s\",error details: %s", rif.tenantIdentifier, err)
			return nil, err
		}
	}

	if !targetInstance.IsShared() {
		log.C(ctx).Debugf("The target instance %s is not shared.", targetInstance)
		return nil, util.HandleInstanceSharingError(util.ErrReferencedInstanceNotShared, targetInstance.ID)
	}

	log.C(ctx).Infof("Reference Instance Provision passed successfully, instanceID: \"%s\"", referencedInstanceID)
	return next.Handle(req)
}

func (rif *referenceInstanceFilter) handleServiceUpdate(req *web.Request, next web.Handler) (*web.Response, error) {
	// we don't want to allow plan_id and/or parameters changes on a reference service instance
	resourceID := req.PathParams["resource_id"]
	ctx := req.Context()

	dbInstanceObject, err := storage.GetObjectByField(ctx, rif.repository, types.ServiceInstanceType, "id", resourceID)
	if err != nil {
		return next.Handle(req)
	}
	instance := dbInstanceObject.(*types.ServiceInstance)

	if instance.ReferencedInstanceID == "" {
		return next.Handle(req)
	}

	err = storage.IsValidReferenceInstancePatchRequest(req, instance, planIDProperty)
	if err != nil {
		log.C(ctx).Errorf("Failed updating the reference instance %s due to invalid patch request:\n%s", instance.ID, req.Body)
		return nil, err // handled by IsValidReferenceInstancePatchRequest
	}

	log.C(ctx).Infof("Reference Instance Update passed successfully, instanceID: \"%s\"", resourceID)
	return next.Handle(req)
}
