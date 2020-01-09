package interceptors

import (
	"context"
	"fmt"

	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/storage"
)

func NewBrokerNotificationsInterceptor() *NotificationsInterceptor {
	return &NotificationsInterceptor{
		PlatformIdProviderFunc: func(ctx context.Context, obj types.Object) string {
			return ""
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
