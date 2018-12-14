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
	"net/http"
	"time"

	"github.com/Peripli/service-manager/pkg/query"

	"github.com/tidwall/gjson"

	"github.com/Peripli/service-manager/pkg/util"

	"github.com/Peripli/service-manager/pkg/types"
	"github.com/gofrs/uuid"

	"github.com/Peripli/service-manager/pkg/log"

	"github.com/Peripli/service-manager/storage"

	"github.com/Peripli/service-manager/pkg/web"
)

// FreeServicePlansFilter reconciles the state of the free plans offered by all service brokers registered in SM. The
// filter makes sure that a public visibility exists for each free plan present in SM DB.
type FreeServicePlansFilter struct {
	Repository storage.Repository
}

func (fsp *FreeServicePlansFilter) Name() string {
	return "FreePlansFilter"
}

func (fsp *FreeServicePlansFilter) Run(req *web.Request, next web.Handler) (*web.Response, error) {
	response, err := next.Handle(req)
	if err != nil {
		return nil, err
	}
	ctx := req.Context()
	brokerID := gjson.GetBytes(response.Body, "id").String()
	log.C(ctx).Debugf("Reconciling free plans for broker with id: %s", brokerID)
	if err := fsp.Repository.InTransaction(ctx, func(ctx context.Context, storage storage.Warehouse) error {
		soRepository := storage.ServiceOffering()
		vRepository := storage.Visibility()

		catalog, err := soRepository.ListWithServicePlansByBrokerID(ctx, brokerID)
		if err != nil {
			return err
		}
		for _, serviceOffering := range catalog {
			for _, servicePlan := range serviceOffering.Plans {
				planID := servicePlan.ID
				isFree := servicePlan.Free
				hasPublicVisibility := false
				byServicePlanID := query.ByField(query.EqualsOperator, "service_plan_id", planID)
				visibilitiesForPlan, err := vRepository.List(ctx, byServicePlanID)
				if err != nil {
					return err
				}
				for _, visibility := range visibilitiesForPlan {
					byVisibilityID := query.ByField(query.EqualsOperator, "id", visibility.ID)
					if isFree {
						if visibility.PlatformID == "" {
							hasPublicVisibility = true
							continue
						} else {
							if err := vRepository.Delete(ctx, byVisibilityID); err != nil {
								return err
							}
						}
					} else {
						if visibility.PlatformID == "" {
							if err := vRepository.Delete(ctx, byVisibilityID); err != nil {
								return err
							}
						} else {
							continue
						}
					}
				}

				if isFree && !hasPublicVisibility {
					UUID, err := uuid.NewV4()
					if err != nil {
						return fmt.Errorf("could not generate GUID for visibility: %s", err)
					}

					currentTime := time.Now().UTC()
					planID, err := vRepository.Create(ctx, &types.Visibility{
						ID:            UUID.String(),
						ServicePlanID: servicePlan.ID,
						CreatedAt:     currentTime,
						UpdatedAt:     currentTime,
					})
					if err != nil {
						return util.HandleStorageError(err, "visibility")
					}

					log.C(ctx).Debugf("Created new public visibility for broker with id %s and plan with id %s", brokerID, planID)
				}
			}
		}
		return nil
	}); err != nil {
		return nil, err
	}
	log.C(ctx).Debugf("Successfully finished reconciling free plans for broker with id %s", brokerID)
	return response, nil
}

func (fsp *FreeServicePlansFilter) FilterMatchers() []web.FilterMatcher {
	return []web.FilterMatcher{
		{
			Matchers: []web.Matcher{
				web.Path(web.BrokersURL + "/**"),
				web.Methods(http.MethodPost, http.MethodPatch),
			},
		},
	}
}
