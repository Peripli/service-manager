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
	SupportedPlatforms      func(ctx context.Context, plan *types.ServicePlan, repository storage.Repository) (map[string]*types.Platform, error)
	TenantKey               string
}

func (p *PublicPlanCreateInterceptorProvider) Provide() storage.CreateInterceptor {
	return &publicPlanCreateInterceptor{
		isCatalogPlanPublicFunc: p.IsCatalogPlanPublicFunc,
		supportedPlatforms:      p.SupportedPlatforms,
		tenantKey:               p.TenantKey,
	}
}

func (p *PublicPlanCreateInterceptorProvider) Name() string {
	return CreateBrokerPublicPlanInterceptorName
}

type PublicPlanUpdateInterceptorProvider struct {
	IsCatalogPlanPublicFunc publicPlanProcessor
	SupportedPlatforms      func(ctx context.Context, plan *types.ServicePlan, repository storage.Repository) (map[string]*types.Platform, error)
	TenantKey               string
}

func (p *PublicPlanUpdateInterceptorProvider) Name() string {
	return UpdateBrokerPublicPlanInterceptorName
}

func (p *PublicPlanUpdateInterceptorProvider) Provide() storage.UpdateInterceptor {
	return &publicPlanUpdateInterceptor{
		isCatalogPlanPublicFunc: p.IsCatalogPlanPublicFunc,
		supportedPlatforms:      p.SupportedPlatforms,
		tenantKey:               p.TenantKey,
	}
}

type publicPlanCreateInterceptor struct {
	isCatalogPlanPublicFunc publicPlanProcessor
	supportedPlatforms      func(ctx context.Context, plan *types.ServicePlan, repository storage.Repository) (map[string]*types.Platform, error)
	tenantKey               string
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
		return newObject, resync(ctx, obj.(*types.ServiceBroker), txStorage, p.isCatalogPlanPublicFunc, p.supportedPlatforms, p.tenantKey)
	}
}

type publicPlanUpdateInterceptor struct {
	isCatalogPlanPublicFunc publicPlanProcessor
	supportedPlatforms      func(ctx context.Context, plan *types.ServicePlan, repository storage.Repository) (map[string]*types.Platform, error)
	tenantKey               string
}

func (p *publicPlanUpdateInterceptor) AroundTxUpdate(h storage.InterceptUpdateAroundTxFunc) storage.InterceptUpdateAroundTxFunc {
	return h
}

func (p *publicPlanUpdateInterceptor) OnTxUpdate(f storage.InterceptUpdateOnTxFunc) storage.InterceptUpdateOnTxFunc {
	return func(ctx context.Context, txStorage storage.Repository, oldObj, newObj types.Object, labelChanges ...*types.LabelChange) (types.Object, error) {
		result, err := f(ctx, txStorage, oldObj, newObj, labelChanges...)
		if err != nil {
			return nil, err
		}
		return result, resync(ctx, result.(*types.ServiceBroker), txStorage, p.isCatalogPlanPublicFunc, p.supportedPlatforms, p.tenantKey)
	}
}

func resync(ctx context.Context, broker *types.ServiceBroker, txStorage storage.Repository, isCatalogPlanPublicFunc publicPlanProcessor, supportedPlatforms func(ctx context.Context, plan *types.ServicePlan, repository storage.Repository) (map[string]*types.Platform, error), tenantKey string) error {
	for _, serviceOffering := range broker.Services {
		for _, servicePlan := range serviceOffering.Plans {
			planID := servicePlan.ID

			isPlanPublic, err := isCatalogPlanPublicFunc(broker, serviceOffering, servicePlan)
			if err != nil {
				return err
			}

			byServicePlanID := query.ByField(query.EqualsOperator, "service_plan_id", planID)
			planVisibilities, err := txStorage.List(ctx, types.VisibilityType, byServicePlanID)
			if err != nil {
				return err
			}

			if servicePlan.SupportsAllPlatforms() {
				err = resyncPublicPlanVisibilities(ctx, txStorage, planVisibilities, isPlanPublic, planID, broker)
			} else { // not all platforms are supported -> create single visibility for each supported platform
				supportedPlatformIDs, err := supportedPlatforms(ctx, servicePlan, txStorage)
				if err != nil {
					return err
				}

				err = resyncPlanVisibilitiesWithSupportedPlatforms(ctx, txStorage, planVisibilities, isPlanPublic, planID, broker, supportedPlatformIDs, tenantKey)
			}

			if err != nil {
				return err
			}
		}
	}
	return nil
}

func resyncPublicPlanVisibilities(ctx context.Context, txStorage storage.Repository, planVisibilities types.ObjectList, isPlanPublic bool, planID string, broker *types.ServiceBroker) error {
	publicVisibilityExists := false

	for i := 0; i < planVisibilities.Len(); i++ {
		visibility := planVisibilities.ItemAt(i).(*types.Visibility)
		byVisibilityID := query.ByField(query.EqualsOperator, "id", visibility.ID)

		shouldDeleteVisibility := true
		if isPlanPublic {
			if visibility.PlatformID == "" {
				publicVisibilityExists = true
				shouldDeleteVisibility = false
			}
		} else {
			if visibility.PlatformID != "" {
				shouldDeleteVisibility = false
			}
		}

		if shouldDeleteVisibility {
			if err := txStorage.Delete(ctx, types.VisibilityType, byVisibilityID); err != nil {
				return err
			}
		}
	}

	if isPlanPublic && !publicVisibilityExists {
		if err := persistVisibility(ctx, txStorage, "", planID, broker); err != nil {
			return err
		}
	}

	return nil
}

func resyncPlanVisibilitiesWithSupportedPlatforms(ctx context.Context, txStorage storage.Repository, planVisibilities types.ObjectList, isPlanPublic bool, planID string, broker *types.ServiceBroker, supportedPlatforms map[string]*types.Platform, tenantKey string) error {
	for i := 0; i < planVisibilities.Len(); i++ {
		visibility := planVisibilities.ItemAt(i).(*types.Visibility)

		shouldDeleteVisibility := true

		id, matches, isSubAccountScoped := platformsAnyMatchesVisibility(supportedPlatforms, visibility, tenantKey)
		if isPlanPublic { // trying to match the current visibility to one of the supported platforms that should have visibilities
			if matches && len(visibility.Labels) == 0 { // visibility is present, no need to create a new one or delete this one
				delete(supportedPlatforms, id)
				shouldDeleteVisibility = false
			}
		} else {
			// trying to match the current visibility to one of the supported platforms - if match is found and it has no labels - it's a public visibility and it has to be deleted
			if matches && len(visibility.Labels) != 0 || matches && isSubAccountScoped { // visibility is present, but has labels -> visibility for paid so don't delete it
				shouldDeleteVisibility = false
			}
		}

		if shouldDeleteVisibility {
			byVisibilityID := query.ByField(query.EqualsOperator, "id", visibility.ID)
			if err := txStorage.Delete(ctx, types.VisibilityType, byVisibilityID); err != nil {
				return err
			}
		}
	}

	if isPlanPublic {
		for platformID := range supportedPlatforms {
			if err := persistVisibility(ctx, txStorage, platformID, planID, broker); err != nil {
				return err
			}
		}
	}

	return nil
}

// platformsAnyMatchesVisibility checks whether any of the platform IDs matches the provided visibility
func platformsAnyMatchesVisibility(supportedPlatforms map[string]*types.Platform, visibility *types.Visibility, tenantKey string) (string, bool, bool) {
	for id, platform := range supportedPlatforms {
		if visibility.PlatformID == id {
			if _, ok := platform.GetLabels()[tenantKey]; ok {
				return id, true, true
			}
			return id, true, false
		}
	}

	return "", false, false
}

func persistVisibility(ctx context.Context, txStorage storage.Repository, platformID, planID string, broker *types.ServiceBroker) error {
	UUID, err := uuid.NewV4()
	if err != nil {
		return fmt.Errorf("could not generate GUID for visibility: %s", err)
	}

	currentTime := time.Now().UTC()
	visibility := &types.Visibility{
		Base: types.Base{
			ID:        UUID.String(),
			UpdatedAt: currentTime,
			CreatedAt: currentTime,
			Ready:     broker.GetReady(),
		},
		ServicePlanID: planID,
		PlatformID:    platformID,
	}

	_, err = txStorage.Create(ctx, visibility)
	if err != nil {
		return err
	}

	log.C(ctx).Debugf("Created new public visibility for broker with id (%s), plan with id (%s) and platform with id (%s)", broker.ID, planID, platformID)
	return nil
}
