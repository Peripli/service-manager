package interceptors

import (
	"context"
	"fmt"

	"github.com/Peripli/service-manager/pkg/util"

	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/storage"
)

func NewBrokerNotificationsInterceptor() *NotificationsInterceptor {
	return &NotificationsInterceptor{
		PlatformIdProviderFunc: func(ctx context.Context, obj types.Object) string {
			return ""
		},
		AdditionalDetailsFunc: func(ctx context.Context, obj types.Object, repository storage.Repository) (util.InputValidator, error) {
			broker := obj.(*types.ServiceBroker)

			return &BrokerAdditional{
				Services: broker.Services,
			}, nil
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

type BrokerNotificationsCreateInterceptorProvider struct {
}

func (*BrokerNotificationsCreateInterceptorProvider) Name() string {
	return "BrokerNotificationsCreateInterceptorProvider"
}

func (*BrokerNotificationsCreateInterceptorProvider) Provide() storage.CreateInterceptor {
	return NewBrokerNotificationsInterceptor()
}

type BrokerNotificationsUpdateInterceptorProvider struct {
}

func (*BrokerNotificationsUpdateInterceptorProvider) Name() string {
	return "BrokerNotificationsUpdateInterceptorProvider"
}

func (*BrokerNotificationsUpdateInterceptorProvider) Provide() storage.UpdateInterceptor {
	return NewBrokerNotificationsInterceptor()
}

type BrokerNotificationsDeleteInterceptorProvider struct {
}

func (*BrokerNotificationsDeleteInterceptorProvider) Name() string {
	return "BrokerNotificationsDeleteInterceptorProvider"
}

func (*BrokerNotificationsDeleteInterceptorProvider) Provide() storage.DeleteInterceptor {
	return NewBrokerNotificationsInterceptor()
}
