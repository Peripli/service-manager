package cf

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"sync"

	"github.com/Peripli/service-manager/pkg/log"

	"github.com/pkg/errors"

	"github.com/Peripli/service-manager/pkg/agent/platform"

	"github.com/cloudfoundry-community/go-cfclient"
)

const maxChunkLength = 50

// OrgLabelKey label key for CF organization visibilities
const OrgLabelKey = "organization_guid"

// VisibilityScopeLabelKey returns key to be used when scoping visibilities
func (pc *PlatformClient) VisibilityScopeLabelKey() string {
	return OrgLabelKey
}

// GetVisibilitiesByBrokers returns platform visibilities grouped by brokers based on given SM brokers.
// The visibilities are taken from CF cloud controller.
// For public plans, visibilities are created so that sync with sm visibilities is possible
// nolint: gocyclo
func (pc *PlatformClient) GetVisibilitiesByBrokers(ctx context.Context, brokerNames []string) ([]*platform.Visibility, error) {
	logger := log.C(ctx)
	logger.Debugf("Gettings brokers from platform for names: %s", brokerNames)
	platformBrokers, err := pc.getBrokersByName(ctx, brokerNames)
	if err != nil {
		return nil, errors.Wrap(err, "could not get brokers from platform")
	}
	logger.Debugf("%d platform brokers found", len(platformBrokers))

	services, err := pc.getServicesByBrokers(ctx, platformBrokers)
	if err != nil {
		return nil, errors.Wrap(err, "could not get services from platform")
	}
	logger.Debugf("%d platform services found", len(services))

	plans, err := pc.getPlansByServices(ctx, services)
	if err != nil {
		return nil, errors.Wrap(err, "could not get plans from platform")
	}
	logger.Debugf("%d platform plans found", len(plans))

	visibilities, err := pc.getPlansVisibilities(ctx, plans)
	if err != nil {
		return nil, errors.Wrap(err, "could not get visibilities from platform")
	}

	type planBrokerIDs struct {
		PlanCatalogID      string
		PlatformBrokerName string
	}

	planUUIDToMapping := make(map[string]planBrokerIDs)
	platformBrokerGUIDToBrokerName := make(map[string]string)

	publicPlans := make([]cfclient.ServicePlan, 0)

	for _, broker := range platformBrokers {
		// Extract SM broker ID from platform broker name
		platformBrokerGUIDToBrokerName[broker.Guid] = broker.Name
	}

	for _, plan := range plans {
		if plan.Public {
			publicPlans = append(publicPlans, plan)
		}
		for _, service := range services {
			if plan.ServiceGuid == service.Guid {
				planUUIDToMapping[plan.Guid] = planBrokerIDs{
					PlatformBrokerName: platformBrokerGUIDToBrokerName[service.ServiceBrokerGuid],
					PlanCatalogID:      plan.UniqueId,
				}
			}
		}
	}

	result := make([]*platform.Visibility, 0, len(visibilities)+len(publicPlans))

	for _, visibility := range visibilities {
		labels := make(map[string]string)
		labels[OrgLabelKey] = visibility.OrganizationGuid
		planMapping := planUUIDToMapping[visibility.ServicePlanGuid]

		result = append(result, &platform.Visibility{
			Public:             false,
			CatalogPlanID:      planMapping.PlanCatalogID,
			PlatformBrokerName: planMapping.PlatformBrokerName,
			Labels:             labels,
		})
	}

	for _, plan := range publicPlans {
		result = append(result, &platform.Visibility{
			Public:             true,
			CatalogPlanID:      plan.UniqueId,
			PlatformBrokerName: planUUIDToMapping[plan.Guid].PlatformBrokerName,
			Labels:             map[string]string{},
		})
	}

	return result, nil
}

func (pc *PlatformClient) getBrokersByName(ctx context.Context, names []string) ([]cfclient.ServiceBroker, error) {
	var errorOccured error
	var mutex sync.Mutex
	var wg sync.WaitGroup
	wgLimitChannel := make(chan struct{}, pc.settings.Reconcile.MaxParallelRequests)

	result := make([]cfclient.ServiceBroker, 0, len(names))
	chunks := splitStringsIntoChunks(names)

	for _, chunk := range chunks {
		select {
		case <-ctx.Done():
			return nil, errors.WithStack(ctx.Err())
		case wgLimitChannel <- struct{}{}:
		}
		wg.Add(1)
		go func(chunk []string) {
			defer func() {
				<-wgLimitChannel
				wg.Done()
			}()
			brokerNames := make([]string, 0, len(chunk))
			brokerNames = append(brokerNames, chunk...)
			query := queryBuilder{}
			query.set("name", brokerNames)
			brokers, err := pc.client.ListServiceBrokersByQuery(query.build(ctx))

			mutex.Lock()
			defer mutex.Unlock()
			if err != nil {
				if errorOccured == nil {
					errorOccured = err
				}
			} else if errorOccured == nil {
				result = append(result, brokers...)
			}
		}(chunk)
	}
	wg.Wait()
	if errorOccured != nil {
		return nil, errorOccured
	}
	return result, nil
}

func (pc *PlatformClient) getServicesByBrokers(ctx context.Context, brokers []cfclient.ServiceBroker) ([]cfclient.Service, error) {
	var errorOccured error
	var mutex sync.Mutex
	var wg sync.WaitGroup
	wgLimitChannel := make(chan struct{}, pc.settings.Reconcile.MaxParallelRequests)

	result := make([]cfclient.Service, 0, len(brokers))
	chunks := splitBrokersIntoChunks(brokers)

	for _, chunk := range chunks {
		select {
		case <-ctx.Done():
			return nil, errors.WithStack(ctx.Err())
		case wgLimitChannel <- struct{}{}:
		}
		wg.Add(1)
		go func(chunk []cfclient.ServiceBroker) {
			defer func() {
				<-wgLimitChannel
				wg.Done()
			}()
			brokerGUIDs := make([]string, 0, len(chunk))
			for _, broker := range chunk {
				brokerGUIDs = append(brokerGUIDs, broker.Guid)
			}
			brokers, err := pc.getServicesByBrokerGUIDs(ctx, brokerGUIDs)

			mutex.Lock()
			defer mutex.Unlock()
			if err != nil {
				if errorOccured == nil {
					errorOccured = err
				}
			} else if errorOccured == nil {
				result = append(result, brokers...)
			}
		}(chunk)
	}
	wg.Wait()
	if errorOccured != nil {
		return nil, errorOccured
	}
	return result, nil
}

func (pc *PlatformClient) getServicesByBrokerGUIDs(ctx context.Context, brokerGUIDs []string) ([]cfclient.Service, error) {
	query := queryBuilder{}
	query.set("service_broker_guid", brokerGUIDs)
	return pc.client.ListServicesByQuery(query.build(ctx))
}

func (pc *PlatformClient) getPlansByServices(ctx context.Context, services []cfclient.Service) ([]cfclient.ServicePlan, error) {
	var errorOccured error
	var mutex sync.Mutex
	var wg sync.WaitGroup
	wgLimitChannel := make(chan struct{}, pc.settings.Reconcile.MaxParallelRequests)

	result := make([]cfclient.ServicePlan, 0, len(services))
	chunks := splitServicesIntoChunks(services)

	for _, chunk := range chunks {
		select {
		case <-ctx.Done():
			return nil, errors.WithStack(ctx.Err())
		case wgLimitChannel <- struct{}{}:
		}
		wg.Add(1)
		go func(chunk []cfclient.Service) {
			defer func() {
				<-wgLimitChannel
				wg.Done()
			}()
			serviceGUIDs := make([]string, 0, len(chunk))
			for _, service := range chunk {
				serviceGUIDs = append(serviceGUIDs, service.Guid)
			}
			plans, err := pc.getPlansByServiceGUIDs(ctx, serviceGUIDs)

			mutex.Lock()
			defer mutex.Unlock()
			if err != nil {
				if errorOccured == nil {
					errorOccured = err
				}
			} else if errorOccured == nil {
				result = append(result, plans...)
			}
		}(chunk)
	}
	wg.Wait()
	if errorOccured != nil {
		return nil, errorOccured
	}
	return result, nil
}

func (pc *PlatformClient) getPlansByServiceGUIDs(ctx context.Context, serviceGUIDs []string) ([]cfclient.ServicePlan, error) {
	query := queryBuilder{}
	query.set("service_guid", serviceGUIDs)
	return pc.client.ListServicePlansByQuery(query.build(ctx))
}

func (pc *PlatformClient) getPlansVisibilities(ctx context.Context, plans []cfclient.ServicePlan) ([]cfclient.ServicePlanVisibility, error) {
	var result []cfclient.ServicePlanVisibility
	var errorOccured error
	var wg sync.WaitGroup
	var mutex sync.Mutex
	wgLimitChannel := make(chan struct{}, pc.settings.Reconcile.MaxParallelRequests)

	chunks := splitCFPlansIntoChunks(plans)

	for _, chunk := range chunks {
		select {
		case <-ctx.Done():
			return nil, errors.WithStack(ctx.Err())
		case wgLimitChannel <- struct{}{}:
		}
		wg.Add(1)
		go func(chunk []cfclient.ServicePlan) {
			defer func() {
				<-wgLimitChannel
				wg.Done()
			}()

			plansGUID := make([]string, 0, len(chunk))
			for _, p := range chunk {
				plansGUID = append(plansGUID, p.Guid)
			}
			visibilities, err := pc.getPlanVisibilitiesByPlanGUID(ctx, plansGUID)

			mutex.Lock()
			defer mutex.Unlock()

			if err != nil {
				if errorOccured == nil {
					errorOccured = err
				}
			} else if errorOccured == nil {
				result = append(result, visibilities...)
			}
		}(chunk)
	}
	wg.Wait()
	if errorOccured != nil {
		return nil, errorOccured
	}
	return result, nil
}

func (pc *PlatformClient) getPlanVisibilitiesByPlanGUID(ctx context.Context, plansGUID []string) ([]cfclient.ServicePlanVisibility, error) {
	query := queryBuilder{}
	query.set("service_plan_guid", plansGUID)
	return pc.client.ListServicePlanVisibilitiesByQuery(query.build(ctx))
}

type queryBuilder struct {
	filters map[string]string
}

func (q *queryBuilder) set(key string, elements []string) *queryBuilder {
	if q.filters == nil {
		q.filters = make(map[string]string)
	}
	searchParameters := strings.Join(elements, ",")
	q.filters[key] = searchParameters
	return q
}

func (q *queryBuilder) build(ctx context.Context) map[string][]string {
	queryComponents := make([]string, 0)
	for key, params := range q.filters {
		component := fmt.Sprintf("%s IN %s", key, params)
		queryComponents = append(queryComponents, component)
	}
	query := strings.Join(queryComponents, ";")
	log.C(ctx).Debugf("CF filter query built: %s", query)
	return url.Values{
		"q": []string{query},
	}
}

func splitCFPlansIntoChunks(plans []cfclient.ServicePlan) [][]cfclient.ServicePlan {
	resultChunks := make([][]cfclient.ServicePlan, 0)

	for count := len(plans); count > 0; count = len(plans) {
		sliceLength := min(count, maxChunkLength)
		resultChunks = append(resultChunks, plans[:sliceLength])
		plans = plans[sliceLength:]
	}
	return resultChunks
}

func splitStringsIntoChunks(names []string) [][]string {
	resultChunks := make([][]string, 0)

	for count := len(names); count > 0; count = len(names) {
		sliceLength := min(count, maxChunkLength)
		resultChunks = append(resultChunks, names[:sliceLength])
		names = names[sliceLength:]
	}
	return resultChunks
}

func splitBrokersIntoChunks(brokers []cfclient.ServiceBroker) [][]cfclient.ServiceBroker {
	resultChunks := make([][]cfclient.ServiceBroker, 0)

	for count := len(brokers); count > 0; count = len(brokers) {
		sliceLength := min(count, maxChunkLength)
		resultChunks = append(resultChunks, brokers[:sliceLength])
		brokers = brokers[sliceLength:]
	}
	return resultChunks
}

func splitServicesIntoChunks(services []cfclient.Service) [][]cfclient.Service {
	resultChunks := make([][]cfclient.Service, 0)

	for count := len(services); count > 0; count = len(services) {
		sliceLength := min(count, maxChunkLength)
		resultChunks = append(resultChunks, services[:sliceLength])
		services = services[sliceLength:]
	}
	return resultChunks
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
