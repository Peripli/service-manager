package interceptors

import (
	"context"
	"fmt"

	"github.com/Peripli/service-manager/pkg/query"

	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/storage"
)

func NewVisibilityNotificationsInterceptor() *NotificationsInterceptor {
	return &NotificationsInterceptor{
		PlatformIDsProviderFunc: func(ctx context.Context, obj types.Object, _ storage.Repository) ([]string, error) {
			platformID := obj.(*types.Visibility).PlatformID
			platformIDS := make([]string, 0)
			if platformID != types.SMPlatform {
				platformIDS = append(platformIDS, platformID)
			}
			//TODO: filter platforms with technical=true
			return platformIDS, nil
		},
		AdditionalDetailsFunc: func(ctx context.Context, objects types.ObjectList, repository storage.Repository) (objectDetails, error) {
			var visibilities []*types.Visibility
			switch t := objects.(type) {
			case *types.Visibilities:
				visibilities = t.Visibilities
			default:
				visibilities = make([]*types.Visibility, objects.Len())
				for i := 0; i < objects.Len(); i++ {
					visibilities[i] = objects.ItemAt(i).(*types.Visibility)
				}
			}
			if len(visibilities) == 0 {
				return objectDetails{}, nil
			}

			plans, err := fetchVisibilityPlans(ctx, repository, visibilities)
			if err != nil {
				return nil, err
			}
			offerings, err := fetchPlanOfferings(ctx, repository, plans)
			if err != nil {
				return nil, err
			}
			brokers, err := fetchOfferingBrokers(ctx, repository, offerings)
			if err != nil {
				return nil, err
			}

			details := make(objectDetails, len(visibilities))
			for _, vis := range visibilities {
				plan := plans[vis.ServicePlanID]
				offering := offerings[plan.ServiceOfferingID]
				broker := brokers[offering.BrokerID]
				details[vis.ID] = &VisibilityAdditional{
					BrokerID:    broker.ID,
					BrokerName:  broker.Name,
					ServicePlan: plan,
				}
			}
			return details, nil
		},
		DeletePostConditionFunc: func(ctx context.Context, object types.Object, repository storage.Repository, platformID string) error {
			return nil
		},
	}
}

func fetchVisibilityPlans(ctx context.Context, repository storage.Repository, visibilities []*types.Visibility) (map[string]*types.ServicePlan, error) {
	planSet := make(map[string]bool, len(visibilities))
	for _, vis := range visibilities {
		planSet[vis.ServicePlanID] = true
	}
	planIDs := make([]string, len(planSet))
	i := 0
	for id := range planSet {
		planIDs[i] = id
		i++
	}
	list, err := repository.List(ctx, types.ServicePlanType,
		query.ByField(query.InOperator, "id", planIDs...))
	if err != nil {
		return nil, err
	}
	plans := list.(*types.ServicePlans).ServicePlans
	planMap := make(map[string]*types.ServicePlan, len(plans))
	for _, plan := range plans {
		planMap[plan.ID] = plan
	}
	return planMap, nil
}

func fetchPlanOfferings(ctx context.Context, repository storage.Repository, plans map[string]*types.ServicePlan) (map[string]*types.ServiceOffering, error) {
	offeringSet := make(map[string]bool, len(plans))
	for _, plan := range plans {
		offeringSet[plan.ServiceOfferingID] = true
	}
	offeringIDs := make([]string, len(offeringSet))
	i := 0
	for id := range offeringSet {
		offeringIDs[i] = id
		i++
	}
	list, err := repository.List(ctx, types.ServiceOfferingType,
		query.ByField(query.InOperator, "id", offeringIDs...))
	if err != nil {
		return nil, err
	}
	offerings := list.(*types.ServiceOfferings).ServiceOfferings
	offeringMap := make(map[string]*types.ServiceOffering, len(offerings))
	for _, offering := range offerings {
		offeringMap[offering.ID] = offering
	}
	return offeringMap, nil
}

func fetchOfferingBrokers(ctx context.Context, repository storage.Repository, offerings map[string]*types.ServiceOffering) (map[string]*types.ServiceBroker, error) {
	brokerSet := make(map[string]bool, len(offerings))
	for _, offering := range offerings {
		brokerSet[offering.BrokerID] = true
	}
	brokerIDs := make([]string, len(brokerSet))
	i := 0
	for id := range brokerSet {
		brokerIDs[i] = id
		i++
	}
	list, err := repository.List(ctx, types.ServiceBrokerType,
		query.ByField(query.InOperator, "id", brokerIDs...))
	if err != nil {
		return nil, err
	}
	brokers := list.(*types.ServiceBrokers).ServiceBrokers
	brokerMap := make(map[string]*types.ServiceBroker, len(brokers))
	for _, broker := range brokers {
		brokerMap[broker.ID] = broker
	}
	return brokerMap, nil
}

type VisibilityAdditional struct {
	BrokerID    string             `json:"broker_id"`
	BrokerName  string             `json:"broker_name"`
	ServicePlan *types.ServicePlan `json:"service_plan,omitempty"`
}

func (va VisibilityAdditional) Validate() error {
	if va.BrokerID == "" {
		return fmt.Errorf("broker id cannot be empty")
	}
	if va.BrokerName == "" {
		return fmt.Errorf("broker name cannot be empty")
	}
	if va.ServicePlan == nil {
		return fmt.Errorf("visibility details service plan cannot be empty")
	}

	return va.ServicePlan.Validate()
}

type VisibilityCreateNotificationsInterceptorProvider struct {
}

func (*VisibilityCreateNotificationsInterceptorProvider) Name() string {
	return "VisibilityCreateNotificationsInterceptorProvider"
}

func (*VisibilityCreateNotificationsInterceptorProvider) Provide() storage.CreateOnTxInterceptor {
	return NewVisibilityNotificationsInterceptor()
}

type VisibilityUpdateNotificationsInterceptorProvider struct {
}

func (*VisibilityUpdateNotificationsInterceptorProvider) Name() string {
	return "VisibilityUpdateNotificationsInterceptorProvider"
}

func (*VisibilityUpdateNotificationsInterceptorProvider) Provide() storage.UpdateOnTxInterceptor {
	return NewVisibilityNotificationsInterceptor()
}

type VisibilityDeleteNotificationsInterceptorProvider struct {
}

func (*VisibilityDeleteNotificationsInterceptorProvider) Name() string {
	return "VisibilityDeleteNotificationsInterceptorProvider"
}

func (*VisibilityDeleteNotificationsInterceptorProvider) Provide() storage.DeleteOnTxInterceptor {
	return NewVisibilityNotificationsInterceptor()
}
