package interceptors

import (
	"context"
	"fmt"

	"github.com/Peripli/service-manager/pkg/query"

	"github.com/Peripli/service-manager/pkg/util"

	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/storage"
)

func NewVisibilityNotificationsInterceptor() *NotificationsInterceptor {
	return &NotificationsInterceptor{
		PlatformIdProviderFunc: func(ctx context.Context, obj types.Object) string {
			return obj.(*types.Visibility).PlatformID
		},
		AdditionalDetailsFunc: func(ctx context.Context, obj types.Object, repository storage.Repository) (util.InputValidator, error) {
			visibility := obj.(*types.Visibility)

			byPlanID := query.ByField(query.EqualsOperator, "id", visibility.ServicePlanID)
			plan, err := repository.Get(ctx, types.ServicePlanType, byPlanID)
			if err != nil {
				return nil, err
			}
			servicePlan := plan.(*types.ServicePlan)

			byServiceID := query.ByField(query.EqualsOperator, "id", servicePlan.ServiceOfferingID)
			service, err := repository.Get(ctx, types.ServiceOfferingType, byServiceID)
			if err != nil {
				return nil, err
			}
			serviceOffering := service.(*types.ServiceOffering)

			byBrokerID := query.ByField(query.EqualsOperator, "id", serviceOffering.BrokerID)
			broker, err := repository.Get(ctx, types.ServiceBrokerType, byBrokerID)
			if err != nil {
				return nil, err
			}

			serviceBroker := broker.(*types.ServiceBroker)

			return &VisibilityAdditional{
				BrokerID:    serviceBroker.ID,
				BrokerName:  serviceBroker.Name,
				ServicePlan: plan.(*types.ServicePlan),
			}, nil
		},
	}
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

func (*VisibilityCreateNotificationsInterceptorProvider) Provide() storage.CreateInterceptor {
	return NewVisibilityNotificationsInterceptor()
}

type VisibilityUpdateNotificationsInterceptorProvider struct {
}

func (*VisibilityUpdateNotificationsInterceptorProvider) Name() string {
	return "VisibilityUpdateNotificationsInterceptorProvider"
}

func (*VisibilityUpdateNotificationsInterceptorProvider) Provide() storage.UpdateInterceptor {
	return NewVisibilityNotificationsInterceptor()
}

type VisibilityDeleteNotificationsInterceptorProvider struct {
}

func (*VisibilityDeleteNotificationsInterceptorProvider) Name() string {
	return "VisibilityDeleteNotificationsInterceptorProvider"
}

func (*VisibilityDeleteNotificationsInterceptorProvider) Provide() storage.DeleteInterceptor {
	return NewVisibilityNotificationsInterceptor()
}
