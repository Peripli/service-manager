package interceptors

import (
	"context"
	"fmt"
	"github.com/Peripli/service-manager/pkg/log"
	"github.com/Peripli/service-manager/pkg/query"
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/pkg/util"
	"github.com/Peripli/service-manager/storage"
	"github.com/Peripli/service-manager/storage/catalog"
	"github.com/Peripli/service-manager/storage/service_plans"
	"time"
)

func NewBrokerNotificationsInterceptor(tenantKey string, notificationsKeepFor time.Duration) *NotificationsInterceptor {
	return &NotificationsInterceptor{
		PlatformIDsProviderFunc: func(ctx context.Context, obj types.Object, repository storage.Repository) ([]string, error) {
			broker := obj.(*types.ServiceBroker)

			var err error
			plans := make([]*types.ServicePlan, 0)
			if len(broker.Services) == 0 { // broker create/update might be triggered inside an existing transaction, which will result in not loading the broker catalog
				plans, err = fetchBrokerPlans(ctx, broker.ID, repository)
				if err != nil {
					return nil, err
				}
			} else {
				for _, svc := range broker.Services {
					plans = append(plans, svc.Plans...)
				}
			}

			var supportedPlatforms map[string]*types.Platform

			if tenantIDValues, found := broker.Labels[tenantKey]; found && len(tenantIDValues) > 0 {
				// tenant-scoped broker
				supportedPlatforms, err = service_plans.ResolveSupportedPlatformsForTenant(ctx, plans, repository, tenantKey, tenantIDValues[0])
			} else {
				// global broker
				supportedPlatforms, err = service_plans.ResolveSupportedPlatformsForPlans(ctx, plans, repository)
			}

			if err != nil {
				return nil, err
			}

			supportedPlatformIDs := make([]string, 0)
			for id, platform := range supportedPlatforms {
				if !platform.Technical && (platform.Active || platform.LastActive.After(time.Now().Add(-notificationsKeepFor))) {
					// only platforms that are active or were active in the time period we keep notifications for are notified
					supportedPlatformIDs = append(supportedPlatformIDs, id)
				}
			}

			return removeSMPlatform(supportedPlatformIDs), nil
		},
		AdditionalDetailsFunc: func(ctx context.Context, objects types.ObjectList, repository storage.Repository) (objectDetails, error) {
			details := make(objectDetails, objects.Len())
			for i := 0; i < objects.Len(); i++ {
				broker := objects.ItemAt(i).(*types.ServiceBroker)
				services := broker.Services
				if len(services) == 0 {
					var err error
					serviceOfferings, err := catalog.Load(ctx, broker.ID, repository)
					if err != nil {
						return nil, err
					}
					services = serviceOfferings.ServiceOfferings
				}
				details[broker.ID] = &BrokerAdditional{
					Services: services,
				}
			}
			return details, nil
		},
		DeletePostConditionFunc: func(ctx context.Context, object types.Object, repository storage.Repository, platformID string) error {
			criteria := []query.Criterion{
				query.ByField(query.EqualsOperator, "broker_id", object.GetID()),
				query.ByField(query.EqualsOperator, "platform_id", platformID),
			}

			log.C(ctx).Debugf("Proceeding with deletion of broker platform credentials for broker with id %s and platform with id %s", object.GetID(), platformID)
			if err := repository.Delete(ctx, types.BrokerPlatformCredentialType, criteria...); err != nil {
				if err != util.ErrNotFoundInStorage {
					return err
				}
			}
			return nil
		},
	}
}

func removeSMPlatform(platforms []string) []string {
	for i := range platforms {
		if platforms[i] == types.SMPlatform {
			platforms[i] = platforms[len(platforms)-1]
			return platforms[:len(platforms)-1]
		}
	}
	return platforms
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
	NotificationsKeepFor time.Duration
	TenantKey            string
}

func (*BrokerNotificationsCreateInterceptorProvider) Name() string {
	return BrokerCreateNotificationInterceptorName
}

func (b *BrokerNotificationsCreateInterceptorProvider) Provide() storage.CreateOnTxInterceptor {
	return NewBrokerNotificationsInterceptor(b.TenantKey, b.NotificationsKeepFor)
}

type BrokerNotificationsUpdateInterceptorProvider struct {
	NotificationsKeepFor time.Duration
	TenantKey            string
}

func (*BrokerNotificationsUpdateInterceptorProvider) Name() string {
	return BrokerUpdateNotificationInterceptorName
}

func (b *BrokerNotificationsUpdateInterceptorProvider) Provide() storage.UpdateOnTxInterceptor {
	return NewBrokerNotificationsInterceptor(b.TenantKey, b.NotificationsKeepFor)
}

type BrokerNotificationsDeleteInterceptorProvider struct {
	NotificationsKeepFor time.Duration
	TenantKey            string
}

func (*BrokerNotificationsDeleteInterceptorProvider) Name() string {
	return BrokerDeleteNotificationInterceptorName
}

func (b *BrokerNotificationsDeleteInterceptorProvider) Provide() storage.DeleteOnTxInterceptor {
	return NewBrokerNotificationsInterceptor(b.TenantKey, b.NotificationsKeepFor)
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
