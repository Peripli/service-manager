package interceptors

import (
	"context"
	"fmt"
	"github.com/Peripli/service-manager/pkg/query"
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/storage"
)

func NewBrokerNotificationsInterceptor() *NotificationsInterceptor {
	return &NotificationsInterceptor{
		PlatformIDsProviderFunc: func(ctx context.Context, obj types.Object, repository storage.Repository) ([]string, error) {
			broker := obj.(*types.ServiceBroker)

			var err error
			plans := make([]*types.ServicePlan, 0)
			if len(broker.Services) == 0 {
				plans, err = fetchBrokerPlans(ctx, broker.ID, repository)
				if err != nil {
					return nil, err
				}
			} else {
				for _, svc := range broker.Services {
					for _, plan := range svc.Plans {
						plans = append(plans, plan)
					}
				}
			}

			supportedPlatforms := getSupportedPlatformsForPlans(plans)

			criteria := []query.Criterion{
				query.ByField(query.NotEqualsOperator, "type", types.SMPlatform),
			}

			if len(supportedPlatforms) != 0 {
				criteria = append(criteria, query.ByField(query.InOperator, "type", supportedPlatforms...))
			}

			objList, err := repository.List(ctx, types.PlatformType, criteria...)
			if err != nil {
				return nil, err
			}

			platformIDs := make([]string, 0)
			for i := 0; i < objList.Len(); i++ {
				platformIDs = append(platformIDs, objList.ItemAt(i).GetID())
			}

			return platformIDs, nil
		},
		AdditionalDetailsFunc: func(ctx context.Context, objects types.ObjectList, repository storage.Repository) (objectDetails, error) {
			details := make(objectDetails, objects.Len())
			for i := 0; i < objects.Len(); i++ {
				broker := objects.ItemAt(i).(*types.ServiceBroker)
				details[broker.ID] = &BrokerAdditional{
					Services: broker.Services,
				}
			}
			return details, nil
		},
	}
}

type BrokerAdditional struct {
	Services []*types.ServiceOffering `json:"services,omitempty"`
}

func (ba BrokerAdditional) Validate() error {
	if len(ba.Services) == 0 {
		return fmt.Errorf("broker details services cannot be empty")
	}

	return nil
}

const (
	BrokerCreateNotificationInterceptorName = "BrokerNotificationsCreateInterceptorProvider"
	BrokerUpdateNotificationInterceptorName = "BrokerNotificationsUpdateInterceptorProvider"
	BrokerDeleteNotificationInterceptorName = "BrokerNotificationsDeleteInterceptorProvider"
)

type BrokerNotificationsCreateInterceptorProvider struct {
}

func (*BrokerNotificationsCreateInterceptorProvider) Name() string {
	return BrokerCreateNotificationInterceptorName
}

func (*BrokerNotificationsCreateInterceptorProvider) Provide() storage.CreateOnTxInterceptor {
	return NewBrokerNotificationsInterceptor()
}

type BrokerNotificationsUpdateInterceptorProvider struct {
}

func (*BrokerNotificationsUpdateInterceptorProvider) Name() string {
	return BrokerUpdateNotificationInterceptorName
}

func (*BrokerNotificationsUpdateInterceptorProvider) Provide() storage.UpdateOnTxInterceptor {
	return NewBrokerNotificationsInterceptor()
}

type BrokerNotificationsDeleteInterceptorProvider struct {
}

func (*BrokerNotificationsDeleteInterceptorProvider) Name() string {
	return BrokerDeleteNotificationInterceptorName
}

func (*BrokerNotificationsDeleteInterceptorProvider) Provide() storage.DeleteOnTxInterceptor {
	return NewBrokerNotificationsInterceptor()
}

func fetchBrokerPlans(ctx context.Context, brokerID string, repository storage.Repository) ([]*types.ServicePlan, error) {
	byBrokerID := query.ByField(query.EqualsOperator, "broker_id", brokerID)
	objList, err := repository.List(ctx, types.ServiceOfferingType, byBrokerID)
	if err != nil {
		return nil, err
	}

	if objList.Len() == 0 {
		return nil, nil
	}

	serviceOfferingIDs := make([]string, 0)
	for i := 0; i < objList.Len(); i++ {
		serviceOfferingIDs = append(serviceOfferingIDs, objList.ItemAt(i).GetID())
	}

	byOfferingIDs := query.ByField(query.InOperator, "service_offering_id", serviceOfferingIDs...)
	objList, err = repository.List(ctx, types.ServicePlanType, byOfferingIDs)
	if err != nil {
		return nil, err
	}

	return objList.(*types.ServicePlans).ServicePlans, nil
}

func getSupportedPlatformsForPlans(plans []*types.ServicePlan) []string {
	platformTypes := make(map[string]bool, 0)
	for _, plan := range plans {
		types := plan.SupportedPlatforms()
		for _, t := range types {
			platformTypes[t] = true
		}
	}

	supportedPlatforms := make([]string, 0)
	for platform := range platformTypes {
		supportedPlatforms = append(supportedPlatforms, platform)
	}

	return supportedPlatforms
}
