package interceptors

import (
	"context"
	"encoding/json"

	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/storage"
)

func NewBrokerNotificationsInterceptor() *NotificationsInterceptor {
	return &NotificationsInterceptor{
		PlatformIdSetterFunc: func(ctx context.Context, obj types.Object) string {
			return ""
		},
		AdditionalDetailsFunc: func(ctx context.Context, obj types.Object, repository storage.Repository) (json.Marshaler, error) {
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

func (ba *BrokerAdditional) MarshalJSON() ([]byte, error) {
	type E BrokerAdditional
	toMarshal := struct {
		*E
	}{
		E: (*E)(ba),
	}
	return json.Marshal(toMarshal)
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
