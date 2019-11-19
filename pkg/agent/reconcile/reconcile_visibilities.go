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

package reconcile

import (
	"context"
	"strings"
	"sync"

	"github.com/Peripli/service-manager/pkg/agent/platform"
	"github.com/Peripli/service-manager/pkg/log"
	"github.com/Peripli/service-manager/pkg/types"
)

// reconcileVisibilities handles the reconciliation of the service visibilities
func (r *resyncJob) reconcileVisibilities(ctx context.Context, smVisibilities []*platform.Visibility, smBrokers []*platform.ServiceBroker) {
	log.C(ctx).Infof("Calling platform API to fetch actual platform visibilities")
	platformVisibilities, err := r.getPlatformVisibilitiesByBrokersFromPlatform(ctx, smBrokers)
	if err != nil {
		log.C(ctx).WithError(err).Error("An error occurred while loading visibilities from platform")
		return
	}

	errorOccured := r.reconcileServiceVisibilities(ctx, platformVisibilities, smVisibilities)
	if errorOccured {
		log.C(ctx).Error("Could not reconcile visibilities")
	}
}

func (r *resyncJob) getPlatformVisibilitiesByBrokersFromPlatform(ctx context.Context, brokers []*platform.ServiceBroker) ([]*platform.Visibility, error) {
	logger := log.C(ctx)
	logger.Info("resyncJob getting visibilities from platform")

	names := r.brokerNames(brokers)
	visibilities, err := r.platformClient.Visibility().GetVisibilitiesByBrokers(ctx, names)
	if err != nil {
		return nil, err
	}
	logger.Infof("resyncJob SUCCESSFULLY retrieved %d visibilities from platform", len(visibilities))

	return visibilities, nil
}

func (r *resyncJob) brokerNames(brokers []*platform.ServiceBroker) []string {
	names := make([]string, 0, len(brokers))
	for _, broker := range brokers {
		names = append(names, r.brokerProxyName(broker))
	}
	return names
}

func (r *resyncJob) getSMPlansByBrokersAndOfferings(ctx context.Context, offerings map[string][]*types.ServiceOffering) (map[string][]*types.ServicePlan, error) {
	result := make(map[string][]*types.ServicePlan)
	count := 0
	log.C(ctx).Info("resyncJob getting service plans from platform")
	for brokerID, sos := range offerings {
		if len(sos) == 0 {
			continue
		}
		brokerPlans, err := r.smClient.GetPlansByServiceOfferings(ctx, sos)
		if err != nil {
			return nil, err
		}
		result[brokerID] = brokerPlans
		count += len(brokerPlans)
	}
	log.C(ctx).Infof("resyncJob SUCCESSFULLY retrieved %d plans from Service Manager", count)

	return result, nil
}

func (r *resyncJob) getSMServiceOfferingsByBrokers(ctx context.Context, brokers []*platform.ServiceBroker) (map[string][]*types.ServiceOffering, error) {
	result := make(map[string][]*types.ServiceOffering)
	brokerIDs := make([]string, 0, len(brokers))
	for _, broker := range brokers {
		brokerIDs = append(brokerIDs, broker.GUID)
	}
	log.C(ctx).Info("resyncJob getting service offerings from Service Manager...")
	offerings, err := r.smClient.GetServiceOfferingsByBrokerIDs(ctx, brokerIDs)
	if err != nil {
		return nil, err
	}
	log.C(ctx).Infof("resyncJob SUCCESSFULLY retrieved %d service offerings from Service Manager", len(offerings))

	for _, offering := range offerings {
		if result[offering.BrokerID] == nil {
			result[offering.BrokerID] = make([]*types.ServiceOffering, 0)
		}
		result[offering.BrokerID] = append(result[offering.BrokerID], offering)
	}

	return result, nil
}

func (r *resyncJob) getVisibilitiesFromSM(ctx context.Context, smPlansMap map[brokerPlanKey]*types.ServicePlan, smBrokers []*platform.ServiceBroker) ([]*platform.Visibility, error) {
	logger := log.C(ctx)
	logger.Info("resyncJob getting visibilities from Service Manager...")

	visibilities, err := r.smClient.GetVisibilities(ctx)
	if err != nil {
		return nil, err
	}
	logger.Infof("resyncJob SUCCESSFULLY retrieved %d visibilities from Service Manager", len(visibilities))

	result := make([]*platform.Visibility, 0)

	for _, visibility := range visibilities {
		for _, broker := range smBrokers {
			key := brokerPlanKey{
				brokerID: broker.GUID,
				planID:   visibility.ServicePlanID,
			}
			smPlan, found := smPlansMap[key]
			if !found {
				continue
			}
			converted := r.convertSMVisibility(visibility, smPlan, broker)
			result = append(result, converted...)
		}
	}
	logger.Infof("resyncJob SUCCESSFULLY converted %d Service Manager visibilities to %d platform visibilities", len(visibilities), len(result))

	return result, nil
}

func (r *resyncJob) convertSMVisibility(visibility *types.Visibility, smPlan *types.ServicePlan, broker *platform.ServiceBroker) []*platform.Visibility {
	scopeLabelKey := r.platformClient.Visibility().VisibilityScopeLabelKey()
	shouldBePublic := visibility.PlatformID == "" || len(visibility.Labels[scopeLabelKey]) == 0

	if shouldBePublic {
		return []*platform.Visibility{
			{
				Public:             true,
				CatalogPlanID:      smPlan.CatalogID,
				PlatformBrokerName: r.brokerProxyName(broker),
				Labels:             map[string]string{},
			},
		}
	}

	scopes := visibility.Labels[scopeLabelKey]
	result := make([]*platform.Visibility, 0, len(scopes))
	for _, scope := range scopes {
		result = append(result, &platform.Visibility{
			Public:             false,
			CatalogPlanID:      smPlan.CatalogID,
			PlatformBrokerName: r.brokerProxyName(broker),
			Labels:             map[string]string{scopeLabelKey: scope},
		})
	}
	return result
}

func (r *resyncJob) reconcileServiceVisibilities(ctx context.Context, platformVis, smVis []*platform.Visibility) bool {
	logger := log.C(ctx)
	logger.Info("resyncJob reconciling platform and Service Manager visibilities...")

	platformMap := r.convertVisListToMap(platformVis)
	visibilitiesToCreate := make([]*platform.Visibility, 0)
	for _, visibility := range smVis {
		key := r.getVisibilityKey(visibility)
		existingVis := platformMap[key]
		delete(platformMap, key)
		if existingVis == nil {
			visibilitiesToCreate = append(visibilitiesToCreate, visibility)
		}
	}

	logger.Infof("resyncJob %d visibilities will be removed from the platform", len(platformMap))
	if errorOccured := r.deleteVisibilities(ctx, platformMap); errorOccured != nil {
		logger.WithError(errorOccured).Error("resyncJob - could not remove visibilities from platform")
		return true
	}

	logger.Infof("resyncJob %d visibilities will be created in the platform", len(visibilitiesToCreate))
	if errorOccured := r.createVisibilities(ctx, visibilitiesToCreate); errorOccured != nil {
		logger.WithError(errorOccured).Error("resyncJob - could not create visibilities in platform")
		return true
	}

	return false
}

type visibilityProcessingState struct {
	Ctx           context.Context
	Mutex         sync.Mutex
	ErrorOccurred error

	WaitGroupLimit chan struct{}
	WaitGroup      sync.WaitGroup
}

func (r *resyncJob) newVisibilityProcessingState(ctx context.Context) *visibilityProcessingState {
	return &visibilityProcessingState{
		Ctx:            ctx,
		WaitGroupLimit: make(chan struct{}, r.options.MaxParallelRequests),
	}
}

// deleteVisibilities deletes visibilities from platform. Returns true if error has occurred
func (r *resyncJob) deleteVisibilities(ctx context.Context, visibilities map[string]*platform.Visibility) error {
	state := r.newVisibilityProcessingState(ctx)

	for _, visibility := range visibilities {
		if err := execAsync(state, visibility, r.deleteVisibility); err != nil {
			return err
		}
	}
	return await(state)
}

// createVisibilities creates visibilities from platform. Returns true if error has occurred
func (r *resyncJob) createVisibilities(ctx context.Context, visibilities []*platform.Visibility) error {
	state := r.newVisibilityProcessingState(ctx)

	for _, visibility := range visibilities {
		if err := execAsync(state, visibility, r.createVisibility); err != nil {
			return err
		}
	}
	return await(state)
}

func execAsync(state *visibilityProcessingState, visibility *platform.Visibility, f func(context.Context, *platform.Visibility) error) error {
	select {
	case <-state.Ctx.Done():
		return state.Ctx.Err()
	case state.WaitGroupLimit <- struct{}{}:
	}
	state.WaitGroup.Add(1)
	go func() {
		defer func() {
			<-state.WaitGroupLimit
			state.WaitGroup.Done()
		}()

		err := f(state.Ctx, visibility)
		if err != nil {
			state.Mutex.Lock()
			defer state.Mutex.Unlock()
			if state.ErrorOccurred == nil {
				state.ErrorOccurred = err
			}
		}
	}()

	return nil
}

func await(state *visibilityProcessingState) error {
	state.WaitGroup.Wait()
	return state.ErrorOccurred
}

// getVisibilityKey maps a generic visibility to a specific string. The string contains catalogID and scope for non-public plans
func (r *resyncJob) getVisibilityKey(visibility *platform.Visibility) string {
	scopeLabelKey := r.platformClient.Visibility().VisibilityScopeLabelKey()

	const idSeparator = "|"
	if visibility.Public {
		return strings.Join([]string{"public", "", visibility.PlatformBrokerName, visibility.CatalogPlanID}, idSeparator)
	}
	return strings.Join([]string{"!public", visibility.Labels[scopeLabelKey], visibility.PlatformBrokerName, visibility.CatalogPlanID}, idSeparator)
}

func (r *resyncJob) createVisibility(ctx context.Context, visibility *platform.Visibility) error {
	logger := log.C(ctx)
	logger.Infof("resyncJob creating visibility for catalog plan %s with labels %v...", visibility.CatalogPlanID, visibility.Labels)

	if err := r.platformClient.Visibility().EnableAccessForPlan(ctx, &platform.ModifyPlanAccessRequest{
		BrokerName:    visibility.PlatformBrokerName,
		CatalogPlanID: visibility.CatalogPlanID,
		Labels:        mapToLabels(visibility.Labels),
	}); err != nil {
		return err
	}
	logger.Infof("resyncJob SUCCESSFULLY created visibility for catalog plan %s with labels %v", visibility.CatalogPlanID, visibility.Labels)

	return nil
}

func (r *resyncJob) deleteVisibility(ctx context.Context, visibility *platform.Visibility) error {
	logger := log.C(ctx)
	logger.Infof("resyncJob deleting visibility for catalog plan %s with labels %v...", visibility.CatalogPlanID, visibility.Labels)

	if err := r.platformClient.Visibility().DisableAccessForPlan(ctx, &platform.ModifyPlanAccessRequest{
		BrokerName:    visibility.PlatformBrokerName,
		CatalogPlanID: visibility.CatalogPlanID,
		Labels:        mapToLabels(visibility.Labels),
	}); err != nil {
		return err
	}
	logger.Infof("resyncJob SUCCESSFULLY deleted visibility for catalog plan %s with labels %v", visibility.CatalogPlanID, visibility.Labels)

	return nil
}

func (r *resyncJob) convertVisListToMap(list []*platform.Visibility) map[string]*platform.Visibility {
	result := make(map[string]*platform.Visibility, len(list))
	for _, vis := range list {
		key := r.getVisibilityKey(vis)
		result[key] = vis
	}
	return result
}

func mapPlansByBrokerPlanID(plansByBroker map[string][]*types.ServicePlan) map[brokerPlanKey]*types.ServicePlan {
	plansMap := make(map[brokerPlanKey]*types.ServicePlan, len(plansByBroker))
	for brokerID, brokerPlans := range plansByBroker {
		for _, plan := range brokerPlans {
			key := brokerPlanKey{
				brokerID: brokerID,
				planID:   plan.ID,
			}
			plansMap[key] = plan
		}
	}
	return plansMap
}

type brokerPlanKey struct {
	brokerID string
	planID   string
}

func mapToLabels(m map[string]string) types.Labels {
	labels := types.Labels{}
	for k, v := range m {
		labels[k] = []string{
			v,
		}
	}
	return labels
}
