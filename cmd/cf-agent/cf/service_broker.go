package cf

import (
	"context"

	"github.com/Peripli/service-manager/pkg/agent/platform"
	"github.com/cloudfoundry-community/go-cfclient"
)

// GetBrokers implements service-broker-proxy/pkg/cf/Client.GetBrokers and provides logic for
// obtaining the brokers that are already registered in CF
func (pc *PlatformClient) GetBrokers(ctx context.Context) ([]*platform.ServiceBroker, error) {
	brokers, err := pc.client.ListServiceBrokers()
	if err != nil {
		return nil, wrapCFError(err)
	}

	var clientBrokers []*platform.ServiceBroker
	for _, broker := range brokers {
		serviceBroker := &platform.ServiceBroker{
			GUID:      broker.Guid,
			Name:      broker.Name,
			BrokerURL: broker.BrokerURL,
		}
		clientBrokers = append(clientBrokers, serviceBroker)
	}

	return clientBrokers, nil
}

// GetBrokerByName implements service-broker-proxy/pkg/cf/Client.GetBrokerByName and provides logic for getting a broker by name
// that is already registered in CF
func (pc *PlatformClient) GetBrokerByName(ctx context.Context, name string) (*platform.ServiceBroker, error) {
	broker, err := pc.client.GetServiceBrokerByName(name)
	if err != nil {
		return nil, wrapCFError(err)
	}

	return &platform.ServiceBroker{
		GUID:      broker.Guid,
		Name:      broker.Name,
		BrokerURL: broker.BrokerURL,
	}, nil
}

// CreateBroker implements service-broker-proxy/pkg/cf/Client.CreateBroker and provides logic for
// registering a new broker in CF
func (pc *PlatformClient) CreateBroker(ctx context.Context, r *platform.CreateServiceBrokerRequest) (*platform.ServiceBroker, error) {

	request := cfclient.CreateServiceBrokerRequest{
		Username:  pc.settings.Sm.User,
		Password:  pc.settings.Sm.Password,
		Name:      r.Name,
		BrokerURL: r.BrokerURL,
	}

	broker, err := pc.client.CreateServiceBroker(request)
	if err != nil {
		return nil, wrapCFError(err)
	}

	response := &platform.ServiceBroker{
		GUID:      broker.Guid,
		Name:      broker.Name,
		BrokerURL: broker.BrokerURL,
	}

	return response, nil
}

// DeleteBroker implements service-broker-proxy/pkg/cf/Client.DeleteBroker and provides logic for
// registering a new broker in CF
func (pc *PlatformClient) DeleteBroker(ctx context.Context, r *platform.DeleteServiceBrokerRequest) error {

	if err := pc.client.DeleteServiceBroker(r.GUID); err != nil {
		return wrapCFError(err)
	}

	return nil
}

// UpdateBroker implements service-broker-proxy/pkg/cf/Client.UpdateBroker and provides logic for
// updating a broker registration in CF
func (pc *PlatformClient) UpdateBroker(ctx context.Context, r *platform.UpdateServiceBrokerRequest) (*platform.ServiceBroker, error) {

	request := cfclient.UpdateServiceBrokerRequest{
		Username:  pc.settings.Sm.User,
		Password:  pc.settings.Sm.Password,
		Name:      r.Name,
		BrokerURL: r.BrokerURL,
	}

	broker, err := pc.client.UpdateServiceBroker(r.GUID, request)
	if err != nil {
		return nil, wrapCFError(err)
	}
	response := &platform.ServiceBroker{
		GUID:      broker.Guid,
		Name:      broker.Name,
		BrokerURL: broker.BrokerURL,
	}

	return response, nil
}
