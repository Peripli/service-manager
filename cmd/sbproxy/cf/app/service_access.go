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

package app

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/cloudfoundry-community/go-cfclient"

	"github.com/Peripli/service-manager/pkg/log"
	"github.com/Peripli/service-manager/pkg/sbproxy/platform"
	"github.com/pkg/errors"
)

// Metadata represents CF specific metadata that the proxy is concerned with.
// It is currently used to provide context details for enabling and disabling of service access.
type Metadata struct {
	OrgGUID string `json:"org_guid"`
}

// ServicePlanRequest represents a service plan request
type ServicePlanRequest struct {
	Public bool `json:"public"`
}

var _ platform.ServiceAccess = &PlatformClient{}

// EnableAccessForService implements service-broker-proxy/pkg/cf/ServiceAccess.EnableAccessForService
// and provides logic for enabling the service access for all plans of a service by the service's catalog GUID.
func (pc PlatformClient) EnableAccessForService(ctx context.Context, context json.RawMessage, catalogServiceGUID string) error {
	return pc.updateAccessForService(ctx, context, catalogServiceGUID, true)
}

// DisableAccessForService implements service-broker-proxy/pkg/cf/ServiceAccess.DisableAccessForService
// and provides logic for disabling the service access for all plans of a service by the service's catalog GUID.
func (pc PlatformClient) DisableAccessForService(ctx context.Context, context json.RawMessage, catalogServiceGUID string) error {
	return pc.updateAccessForService(ctx, context, catalogServiceGUID, false)
}

// EnableAccessForPlan implements service-broker-proxy/pkg/cf/ServiceAccess.EnableAccessForPlan
// and provides logic for enabling the service access for a specified plan by the plan's catalog GUID.
func (pc PlatformClient) EnableAccessForPlan(ctx context.Context, context json.RawMessage, catalogPlanGUID string) error {
	return pc.updateAccessForPlan(ctx, context, catalogPlanGUID, true)
}

// DisableAccessForPlan implements service-broker-proxy/pkg/cf/ServiceAccess.DisableAccessForPlan
// and provides logic for disabling the service access for a specified plan by the plan's catalog GUID.
func (pc PlatformClient) DisableAccessForPlan(ctx context.Context, context json.RawMessage, catalogPlanGUID string) error {
	return pc.updateAccessForPlan(ctx, context, catalogPlanGUID, false)
}

func (pc PlatformClient) updateAccessForService(ctx context.Context, context json.RawMessage, catalogServiceGUID string, isEnabled bool) error {
	metadata := &Metadata{}
	if err := json.Unmarshal(context, metadata); err != nil {
		return err
	}

	service, err := pc.getServiceForCatalogServiceGUID(catalogServiceGUID)
	if err != nil {
		return err
	}

	plans, err := pc.getPlansForPlatformServiceGUID(service.Guid)
	if err != nil {
		return err
	}

	if metadata.OrgGUID != "" {
		if err := pc.updateOrgVisibilitiesForPlans(ctx, plans, isEnabled, metadata.OrgGUID); err != nil {
			return err
		}
	} else {
		if err := pc.updatePlans(plans, isEnabled); err != nil {
			return err
		}
	}

	return nil
}

func (pc PlatformClient) updateAccessForPlan(ctx context.Context, context json.RawMessage, catalogPlanGUID string, isEnabled bool) error {
	metadata := &Metadata{}
	if err := json.Unmarshal(context, metadata); err != nil {
		return err
	}

	plan, err := pc.getPlanForCatalogPlanGUID(catalogPlanGUID)
	if err != nil {
		return err
	}

	if metadata.OrgGUID != "" {
		if err := pc.updateOrgVisibilityForPlan(ctx, plan, isEnabled, metadata.OrgGUID); err != nil {
			return err
		}
	} else {
		if err := pc.updatePlan(plan, isEnabled); err != nil {
			return err
		}
	}

	return nil
}

func (pc PlatformClient) updateOrgVisibilitiesForPlans(ctx context.Context, plans []cfclient.ServicePlan, isEnabled bool, orgGUID string) error {
	for _, plan := range plans {
		if err := pc.updateOrgVisibilityForPlan(ctx, plan, isEnabled, orgGUID); err != nil {
			return err
		}
	}

	return nil
}

func (pc PlatformClient) updateOrgVisibilityForPlan(ctx context.Context, plan cfclient.ServicePlan, isEnabled bool, orgGUID string) error {
	switch {
	case plan.Public:
		log.C(ctx).Info("Plan with GUID = %s and NAME = %s is already public and therefore attempt to update access "+
			"visibility for org with GUID = %s will be ignored", plan.Guid, plan.Name, orgGUID)
	case isEnabled:
		if _, err := pc.Client.CreateServicePlanVisibility(plan.Guid, orgGUID); err != nil {
			return wrapCFError(err)
		}
	case !isEnabled:
		query := url.Values{"q": []string{fmt.Sprintf("service_plan_guid:%s;organization_guid:%s", plan.Guid, orgGUID)}}
		if err := pc.deleteAccessVisibilities(query); err != nil {
			return wrapCFError(err)
		}
	}

	return nil
}

func (pc PlatformClient) updatePlans(plans []cfclient.ServicePlan, isPublic bool) error {
	for _, plan := range plans {
		if err := pc.updatePlan(plan, isPublic); err != nil {
			return err
		}
	}

	return nil
}

func (pc PlatformClient) updatePlan(plan cfclient.ServicePlan, isPublic bool) error {
	query := url.Values{"q": []string{fmt.Sprintf("service_plan_guid:%s", plan.Guid)}}
	if err := pc.deleteAccessVisibilities(query); err != nil {
		return err
	}
	if plan.Public == isPublic {
		return nil
	}
	_, err := pc.UpdateServicePlan(plan.Guid, ServicePlanRequest{
		Public: isPublic,
	})

	return err
}

func (pc PlatformClient) deleteAccessVisibilities(query url.Values) error {
	servicePlanVisibilities, err := pc.Client.ListServicePlanVisibilitiesByQuery(query)
	if err != nil {
		return wrapCFError(err)
	}

	for _, visibility := range servicePlanVisibilities {
		if err := pc.Client.DeleteServicePlanVisibility(visibility.Guid, false); err != nil {
			return wrapCFError(err)
		}
	}

	return nil
}

// UpdateServicePlan updates the public property of the plan with the specified GUID
func (pc PlatformClient) UpdateServicePlan(planGUID string, request ServicePlanRequest) (cfclient.ServicePlan, error) {
	var planResource cfclient.ServicePlanResource
	buf := bytes.NewBuffer(nil)
	if err := json.NewEncoder(buf).Encode(request); err != nil {
		return cfclient.ServicePlan{}, wrapCFError(err)
	}

	req := pc.NewRequestWithBody(http.MethodPut, "/v2/service_plans/"+planGUID, buf)

	response, err := pc.DoRequest(req)
	if err != nil {
		return cfclient.ServicePlan{}, wrapCFError(err)
	}
	if response.StatusCode != http.StatusCreated {
		return cfclient.ServicePlan{}, errors.Errorf("error updating service plan, response code: %d", response.StatusCode)
	}

	decoder := json.NewDecoder(response.Body)
	defer response.Body.Close() // nolint
	if err := decoder.Decode(&planResource); err != nil {
		return cfclient.ServicePlan{}, errors.Wrap(err, "error decoding response body")
	}

	servicePlan := planResource.Entity
	servicePlan.Guid = planResource.Meta.Guid

	return servicePlan, nil
}

func (pc PlatformClient) getServiceForCatalogServiceGUID(catalogServiceGUID string) (cfclient.Service, error) {
	query := url.Values{}
	query.Set("q", fmt.Sprintf("unique_id:%s", catalogServiceGUID))

	services, err := pc.Client.ListServicesByQuery(query)
	if err != nil {
		return cfclient.Service{}, wrapCFError(err)
	}
	if len(services) == 0 {
		return cfclient.Service{}, errors.Errorf("zero services with catalog service GUID = %s found", catalogServiceGUID)
	}
	if len(services) > 1 {
		return cfclient.Service{}, errors.Errorf("more than one service with catalog service GUID = %s found", catalogServiceGUID)

	}

	return services[0], nil
}

func (pc PlatformClient) getPlanForCatalogPlanGUID(catalogPlanGUID string) (cfclient.ServicePlan, error) {
	query := url.Values{}
	query.Set("q", fmt.Sprintf("unique_id:%s", catalogPlanGUID))
	plans, err := pc.Client.ListServicePlansByQuery(query)
	if err != nil {
		return cfclient.ServicePlan{}, wrapCFError(err)
	}
	if len(plans) == 0 {
		return cfclient.ServicePlan{}, errors.Errorf("zero plans with catalog plan GUID = %s found", catalogPlanGUID)
	}
	if len(plans) > 1 {
		return cfclient.ServicePlan{}, errors.Errorf("more than one plan with catalog plan GUID = %s found", catalogPlanGUID)

	}

	return plans[0], nil
}

func (pc PlatformClient) getPlansForPlatformServiceGUID(serviceGUID string) ([]cfclient.ServicePlan, error) {
	query := url.Values{}
	query.Set("q", fmt.Sprintf("service_guid:%s", serviceGUID))

	servicePlans, err := pc.Client.ListServicePlansByQuery(query)
	if err != nil {
		return []cfclient.ServicePlan{}, wrapCFError(err)
	}

	return servicePlans, nil
}
