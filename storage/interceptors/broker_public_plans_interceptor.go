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

package interceptors

import (
	"context"
	"fmt"
	"time"

	"github.com/Peripli/service-manager/pkg/log"
	"github.com/Peripli/service-manager/pkg/query"
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/storage"
	"github.com/gofrs/uuid"
)

const (
	CreateBrokerPublicPlanInterceptorName = "CreateBrokerPublicPlansInterceptor"
	UpdateBrokerPublicPlanInterceptorName = "UpdateBrokerPublicPlansInterceptor"
)

type publicPlanProcessor func(broker *types.ServiceBroker, catalogService *types.ServiceOffering, catalogPlan *types.ServicePlan) (bool, error)

type PublicPlanCreateInterceptorProvider struct {
	IsCatalogPlanPublicFunc publicPlanProcessor
}

func (p *PublicPlanCreateInterceptorProvider) Provide() storage.CreateInterceptor {
	return &publicPlanCreateInterceptor{
		isCatalogPlanPublicFunc: p.IsCatalogPlanPublicFunc,
	}
}

func (p *PublicPlanCreateInterceptorProvider) Name() string {
	return CreateBrokerPublicPlanInterceptorName
}

type PublicPlanUpdateInterceptorProvider struct {
	IsCatalogPlanPublicFunc publicPlanProcessor
}

func (p *PublicPlanUpdateInterceptorProvider) Name() string {
	return UpdateBrokerPublicPlanInterceptorName
}

func (p *PublicPlanUpdateInterceptorProvider) Provide() storage.UpdateInterceptor {
	return &publicPlanUpdateInterceptor{
		isCatalogPlanPublicFunc: p.IsCatalogPlanPublicFunc,
	}
}

type publicPlanCreateInterceptor struct {
	isCatalogPlanPublicFunc publicPlanProcessor
}

func (p *publicPlanCreateInterceptor) AroundTxCreate(h storage.InterceptCreateAroundTxFunc) storage.InterceptCreateAroundTxFunc {
	return h
}

func (p *publicPlanCreateInterceptor) OnTxCreate(f storage.InterceptCreateOnTxFunc) storage.InterceptCreateOnTxFunc {
	return func(ctx context.Context, txStorage storage.Repository, obj types.Object) (types.Object, error) {
		newObject, err := f(ctx, txStorage, obj)
		if err != nil {
			return nil, err
		}
		return newObject, resync(ctx, obj.(*types.ServiceBroker), txStorage, p.isCatalogPlanPublicFunc)
	}
}

type publicPlanUpdateInterceptor struct {
	isCatalogPlanPublicFunc publicPlanProcessor
}

func (p *publicPlanUpdateInterceptor) AroundTxUpdate(h storage.InterceptUpdateAroundTxFunc) storage.InterceptUpdateAroundTxFunc {
	return h
}

func (p *publicPlanUpdateInterceptor) OnTxUpdate(f storage.InterceptUpdateOnTxFunc) storage.InterceptUpdateOnTxFunc {
	return func(ctx context.Context, txStorage storage.Repository, oldObj, newObj types.Object, labelChanges ...*query.LabelChange) (types.Object, error) {
		result, err := f(ctx, txStorage, oldObj, newObj, labelChanges...)
		if err != nil {
			return nil, err
		}
		return result, resync(ctx, result.(*types.ServiceBroker), txStorage, p.isCatalogPlanPublicFunc)
	}
}

func resync(ctx context.Context, broker *types.ServiceBroker, txStorage storage.Repository, isCatalogPlanPublicFunc publicPlanProcessor) error {
	for _, serviceOffering := range broker.Services {
		for _, servicePlan := range serviceOffering.Plans {
			planID := servicePlan.ID
			isPublic, err := isCatalogPlanPublicFunc(broker, serviceOffering, servicePlan)
			if err != nil {
				return err
			}

			hasPublicVisibility := false
			byServicePlanID := query.ByField(query.EqualsOperator, "service_plan_id", planID)
			visibilitiesForPlan, err := txStorage.List(ctx, types.VisibilityType, byServicePlanID)
			if err != nil {
				return err
			}
			for i := 0; i < visibilitiesForPlan.Len(); i++ {
				visibility := visibilitiesForPlan.ItemAt(i).(*types.Visibility)
				byVisibilityID := query.ByField(query.EqualsOperator, "id", visibility.ID)
				if isPublic {
					if visibility.PlatformID == "" {
						hasPublicVisibility = true
						continue
					} else {
						if _, err := txStorage.Delete(ctx, types.VisibilityType, byVisibilityID); err != nil {
							return err
						}
					}
				} else {
					if visibility.PlatformID == "" {
						if _, err := txStorage.Delete(ctx, types.VisibilityType, byVisibilityID); err != nil {
							return err
						}
					} else {
						continue
					}
				}
			}

			if isPublic && !hasPublicVisibility {
				UUID, err := uuid.NewV4()
				if err != nil {
					return fmt.Errorf("could not generate GUID for visibility: %s", err)
				}

				currentTime := time.Now().UTC()
				planID, err := txStorage.Create(ctx, &types.Visibility{
					Base: types.Base{
						ID:        UUID.String(),
						UpdatedAt: currentTime,
						CreatedAt: currentTime,
					},
					ServicePlanID: servicePlan.ID,
				})
				if err != nil {
					return err
				}

				log.C(ctx).Debugf("Created new public visibility for broker with id %s and plan with id %s", broker.ID, planID)
			}
		}
	}
	return nil
}
