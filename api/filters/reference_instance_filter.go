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
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/api/common/sharing"
	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/instance_sharing"
	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/log"
	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/query"
	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/types"
	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/util"
	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/web"
	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/storage"
	"net/http"
	"strings"
)

const ReferenceInstanceFilterName = "ReferenceInstanceFilter"

// referenceInstanceFilter handles the validations and responses of the provisioning, updating and get parameters of reference instances
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
	err := isValidRequestBody(req)
	if err != nil {
		log.C(req.Context()).Errorf("Reference instance filter error, invalid request body: %s", err)
		return nil, err
	}
	switch req.Request.Method {
	case http.MethodPost:
		return rif.handleProvision(req, next)
	case http.MethodPatch:
		return rif.handleServiceUpdate(req, next)
	case http.MethodGet:
		if strings.Contains(req.RequestURI, "/parameters") {
			return rif.handleGetParameters(req, next)
		}
	}
	return next.Handle(req)
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
	isReferencePlan, err := storage.IsReferencePlan(req, rif.repository, types.ServicePlanType.String(), "id", servicePlanID)
	// If plan not found on provisioning, allow sm to handle the issue
	if err == util.ErrNotFoundInStorage {
		log.C(ctx).Errorf("Failed to provision the reference instance to due a storage error: %s", err)
		return next.Handle(req)
	}
	if err != nil {
		log.C(ctx).Errorf("Failed to provision the reference instance: %s", err)
		return nil, err // handled by the IsReference function
	}
	if !isReferencePlan {
		return next.Handle(req)
	}

	referenceInstanceID, err := sharing.ExtractReferencedInstanceID(req, rif.repository, rif.tenantIdentifier, func() string {
		return query.RetrieveFromCriteria(rif.tenantIdentifier, query.CriteriaForContext(req.Context())...)
	}, true)

	if err != nil {
		log.C(ctx).Errorf("Failed to extract the referenced instance id: %s", err)
		return nil, err
	}

	req.Body, err = sjson.SetBytes(req.Body, instance_sharing.ReferencedInstanceIDKey, referenceInstanceID)
	if err != nil {
		log.C(ctx).Errorf("Unable to set the ReferencedInstanceIDKey: \"%s\",error details: %s", referenceInstanceID, err)
		return nil, err
	}

	log.C(ctx).Infof("Reference instance validation has passed successfully, %s: %s", instance_sharing.ReferencedInstanceIDKey, referenceInstanceID)
	return next.Handle(req)
}

func (rif *referenceInstanceFilter) handleServiceUpdate(req *web.Request, next web.Handler) (*web.Response, error) {
	// we don't want to allow plan_id and/or parameters changes on a reference service instance
	resourceID := req.PathParams["resource_id"]
	ctx := req.Context()

	dbInstanceObject, err := storage.GetObjectByField(ctx, rif.repository, types.ServiceInstanceType, "id", resourceID)
	if err != nil {
		log.C(ctx).Errorf("Resource id: %s, not found: %s", resourceID, err)
		return next.Handle(req)
	}
	instance := dbInstanceObject.(*types.ServiceInstance)

	if len(instance.ReferencedInstanceID) == 0 {
		return next.Handle(req)
	}

	err = sharing.IsValidReferenceInstancePatchRequest(req, instance, planIDProperty)
	if err != nil {
		log.C(ctx).Errorf("Failed updating the reference instance %s due to invalid patch request:\n%s", instance.ID, req.Body)
		return nil, err // handled by IsValidReferenceInstancePatchRequest
	}

	log.C(ctx).Infof("Reference instance update passed successfully, instanceID: \"%s\"", resourceID)
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
	if len(instance.ReferencedInstanceID) == 0 {
		return next.Handle(req)
	}

	body := map[string]string{
		instance_sharing.ReferencedInstanceIDKey: instance.ReferencedInstanceID,
	}

	return util.NewJSONResponse(http.StatusOK, body)
}

func isValidRequestBody(req *web.Request) error {
	refKeyOnBody := gjson.GetBytes(req.Body, instance_sharing.ReferencedInstanceIDKey).String()
	if len(refKeyOnBody) > 0 {
		return util.HandleInstanceSharingError(util.ErrRequestBodyContainsReferencedInstanceID, instance_sharing.ReferencedInstanceIDKey)
	}
	return nil
}
