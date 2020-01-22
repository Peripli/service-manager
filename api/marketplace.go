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

package api

import (
	"context"
	"net/http"

	"github.com/Peripli/service-manager/pkg/util"

	"github.com/Peripli/service-manager/pkg/query"
	"github.com/Peripli/service-manager/pkg/types"

	"github.com/Peripli/service-manager/storage"

	"github.com/Peripli/service-manager/pkg/web"
)

type marketplaceItem struct {
	BrokerName      string `json:"broker_name"`
	ServiceName     string `json:"service_name"`
	PlanName        string `json:"plan_name"`
	PlanDescription string `json:"plan_description"`
}

type MarketplaceController struct {
	Repository         storage.TransactionalRepository
	ExtractTenantFunc  func(*web.Request) (string, error)
	VisibilityLabelKey string
}

func (m *MarketplaceController) Routes() []web.Route {
	return []web.Route{
		{
			Endpoint: web.Endpoint{
				Method: http.MethodGet,
				Path:   "/v1/marketplace",
			},
			Handler: m.marketplace,
		},
	}
}

func (m *MarketplaceController) marketplace(request *web.Request) (*web.Response, error) {
	ctx := request.Context()
	tenant, err := m.ExtractTenantFunc(request)
	if err != nil {
		return nil, err
	}
	var marketplace []marketplaceItem

	if err := m.Repository.InTransaction(ctx, func(ctx context.Context, storage storage.Repository) error {
		byLabelKey := query.ByLabel(query.EqualsOperator, m.VisibilityLabelKey, tenant)
		scopedVisibilities, err := storage.List(ctx, types.VisibilityType, byLabelKey)
		if err != nil {
			return err
		}
		byEmptyPlatformID := query.ByField(query.EqualsOrNilOperator, "platform_id", "")
		publicVisibilities, err := storage.List(ctx, types.VisibilityType, byEmptyPlatformID)
		if err != nil {
			return err
		}

		var planIDs []string
		for i := 0; i < scopedVisibilities.Len(); i++ {
			visibility := scopedVisibilities.ItemAt(i).(*types.Visibility)
			planIDs = append(planIDs, visibility.ServicePlanID)
		}

		for i := 0; i < publicVisibilities.Len(); i++ {
			visibility := publicVisibilities.ItemAt(i).(*types.Visibility)
			planIDs = append(planIDs, visibility.ServicePlanID)
		}

		byPlanIDs := query.ByField(query.InOperator, "id", planIDs...)
		plans, err := storage.List(ctx, types.ServicePlanType, byPlanIDs)
		if err != nil {
			return err
		}

		var offeringIDs []string
		offeringIdToPlans := make(map[string][]*types.ServicePlan)
		for i := 0; i < plans.Len(); i++ {
			plan := plans.ItemAt(i).(*types.ServicePlan)
			offeringIDs = append(offeringIDs, plan.ServiceOfferingID)
			offeringIdToPlans[plan.ServiceOfferingID] = append(offeringIdToPlans[plan.ServiceOfferingID], plan)
		}

		byOfferingIDs := query.ByField(query.InOperator, "id", offeringIDs...)
		offerings, err := storage.List(ctx, types.ServiceOfferingType, byOfferingIDs)
		if err != nil {
			return err
		}

		var brokerIDs []string
		brokerIdToOfferingIDs := make(map[string][]string)
		offeringIdToOfferingName := make(map[string]string)
		for i := 0; i < offerings.Len(); i++ {
			offering := offerings.ItemAt(i).(*types.ServiceOffering)
			brokerIDs = append(brokerIDs, offering.BrokerID)
			brokerIdToOfferingIDs[offering.BrokerID] = append(brokerIdToOfferingIDs[offering.BrokerID], offering.GetID())
			offeringIdToOfferingName[offering.ID] = offering.Name
		}

		byBrokerIDs := query.ByField(query.InOperator, "id", brokerIDs...)
		brokers, err := storage.List(ctx, types.ServiceBrokerType, byBrokerIDs)
		if err != nil {
			return err
		}

		for i := 0; i < brokers.Len(); i++ {
			broker := brokers.ItemAt(i).(*types.ServiceBroker)
			offerings := brokerIdToOfferingIDs[broker.ID]
			for _, offeringID := range offerings {
				offeringName := offeringIdToOfferingName[offeringID]
				plans := offeringIdToPlans[offeringID]
				for _, plan := range plans {
					item := marketplaceItem{
						BrokerName:      broker.Name,
						ServiceName:     offeringName,
						PlanName:        plan.Name,
						PlanDescription: plan.Description,
					}
					marketplace = append(marketplace, item)
				}
			}
		}

		return nil
	}); err != nil {
		return nil, err
	}

	response := make(map[string][]marketplaceItem)
	response["marketplace"] = marketplace
	return util.NewJSONResponse(http.StatusOK, response)
}
