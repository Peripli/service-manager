package client

import (
	"context"
	"fmt"

	"github.com/Peripli/service-manager/cmd/k8s-agent/k8s/config"
	"github.com/Peripli/service-manager/pkg/agent/k8s/api"
	servicecatalog "github.com/kubernetes-sigs/service-catalog/pkg/svcat/service-catalog"

	"github.com/Peripli/service-manager/pkg/agent/platform"
	"github.com/kubernetes-sigs/service-catalog/pkg/apis/servicecatalog/v1beta1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const resyncBrokerRetryCount = 3

// NewDefaultKubernetesAPI returns default kubernetes api interface
func NewDefaultKubernetesAPI(cli *servicecatalog.SDK) *ServiceCatalogAPI {
	return &ServiceCatalogAPI{cli}
}

// ServiceCatalogAPI uses service catalog SDK to interact with the kubernetes resources
type ServiceCatalogAPI struct {
	*servicecatalog.SDK
}

// CreateClusterServiceBroker creates a cluster service broker
func (sca *ServiceCatalogAPI) CreateClusterServiceBroker(broker *v1beta1.ClusterServiceBroker) (*v1beta1.ClusterServiceBroker, error) {
	return sca.ServiceCatalog().ClusterServiceBrokers().Create(broker)
}

// DeleteClusterServiceBroker deletes a cluster service broker
func (sca *ServiceCatalogAPI) DeleteClusterServiceBroker(name string, options *v1.DeleteOptions) error {
	return sca.ServiceCatalog().ClusterServiceBrokers().Delete(name, options)
}

// RetrieveClusterServiceBrokers returns all cluster service brokers
func (sca *ServiceCatalogAPI) RetrieveClusterServiceBrokers() (*v1beta1.ClusterServiceBrokerList, error) {
	return sca.ServiceCatalog().ClusterServiceBrokers().List(v1.ListOptions{})
}

// RetrieveClusterServiceBrokerByName returns a cluster service broker by name
func (sca *ServiceCatalogAPI) RetrieveClusterServiceBrokerByName(name string) (*v1beta1.ClusterServiceBroker, error) {
	return sca.ServiceCatalog().ClusterServiceBrokers().Get(name, v1.GetOptions{})
}

// UpdateClusterServiceBroker updates a cluster service broker
func (sca *ServiceCatalogAPI) UpdateClusterServiceBroker(broker *v1beta1.ClusterServiceBroker) (*v1beta1.ClusterServiceBroker, error) {
	return sca.ServiceCatalog().ClusterServiceBrokers().Update(broker)
}

// SyncClusterServiceBroker synchronizes a cluster service broker including its catalog
func (sca *ServiceCatalogAPI) SyncClusterServiceBroker(name string, retries int) error {
	return sca.Sync(name, servicecatalog.ScopeOptions{
		Scope: servicecatalog.ClusterScope,
	}, retries)
}

// PlatformClient implements all broker, visibility and catalog specific operations for kubernetes
type PlatformClient struct {
	platformAPI  api.KubernetesAPI
	regSecretRef *v1beta1.ObjectReference
}

var _ platform.Client = &PlatformClient{}

// NewClient create a client to communicate with the kubernetes service-catalog.
func NewClient(settings *config.Settings) (*PlatformClient, error) {
	if err := settings.Validate(); err != nil {
		return nil, err
	}
	svcatSDK, err := settings.K8S.K8sClientCreateFunc(settings.K8S.ClientSettings)
	if err != nil {
		return nil, err
	}
	return &PlatformClient{
		platformAPI: NewDefaultKubernetesAPI(svcatSDK),
		regSecretRef: &v1beta1.ObjectReference{
			Namespace: settings.K8S.Secret.Namespace,
			Name:      settings.K8S.Secret.Name,
		},
	}, nil
}

// Broker returns the platform client which handles broker operations
func (pc *PlatformClient) Broker() platform.BrokerClient {
	return pc
}

// CatalogFetcher returns the platform client which handles catalog fetch operations
func (pc *PlatformClient) CatalogFetcher() platform.CatalogFetcher {
	return pc
}

// Visibility returns the platform client which handles visibility operations
func (pc *PlatformClient) Visibility() platform.VisibilityClient {
	return pc
}

// GetBrokers returns all service-brokers currently registered in kubernetes service-catalog.
func (pc *PlatformClient) GetBrokers(ctx context.Context) ([]*platform.ServiceBroker, error) {
	brokers, err := pc.platformAPI.RetrieveClusterServiceBrokers()
	if err != nil {
		return nil, fmt.Errorf("unable to list cluster-scoped brokers (%s)", err)
	}
	var clientBrokers = make([]*platform.ServiceBroker, 0)
	for _, broker := range brokers.Items {
		serviceBroker := &platform.ServiceBroker{
			GUID:      string(broker.ObjectMeta.UID),
			Name:      broker.Name,
			BrokerURL: broker.Spec.URL,
		}
		clientBrokers = append(clientBrokers, serviceBroker)
	}
	return clientBrokers, nil
}

// GetBrokerByName returns the service-broker with the specified name currently registered in kubernetes service-catalog with.
func (pc *PlatformClient) GetBrokerByName(ctx context.Context, name string) (*platform.ServiceBroker, error) {
	broker, err := pc.platformAPI.RetrieveClusterServiceBrokerByName(name)
	if err != nil {
		return nil, fmt.Errorf("unable to get cluster-scoped broker (%s)", err)
	}

	return &platform.ServiceBroker{
		GUID:      string(broker.ObjectMeta.UID),
		Name:      broker.Name,
		BrokerURL: broker.Spec.URL,
	}, nil
}

// CreateBroker registers a new broker in kubernetes service-catalog.
func (pc *PlatformClient) CreateBroker(ctx context.Context, r *platform.CreateServiceBrokerRequest) (*platform.ServiceBroker, error) {
	broker := newServiceBroker(r.Name, r.BrokerURL, pc.regSecretRef)
	broker.Spec.CommonServiceBrokerSpec.RelistBehavior = "Manual"

	csb, err := pc.platformAPI.CreateClusterServiceBroker(broker)
	if err != nil {
		return nil, err
	}
	return &platform.ServiceBroker{
		GUID:      string(csb.UID),
		Name:      r.Name,
		BrokerURL: r.BrokerURL,
	}, nil
}

// DeleteBroker deletes an existing broker in from kubernetes service-catalog.
func (pc *PlatformClient) DeleteBroker(ctx context.Context, r *platform.DeleteServiceBrokerRequest) error {
	return pc.platformAPI.DeleteClusterServiceBroker(r.Name, &v1.DeleteOptions{})
}

// UpdateBroker updates a service broker in the kubernetes service-catalog.
func (pc *PlatformClient) UpdateBroker(ctx context.Context, r *platform.UpdateServiceBrokerRequest) (*platform.ServiceBroker, error) {
	// Name and broker url are updateable
	broker := newServiceBroker(r.Name, r.BrokerURL, pc.regSecretRef)

	updatedBroker, err := pc.platformAPI.UpdateClusterServiceBroker(broker)
	if err != nil {
		return nil, err
	}
	return &platform.ServiceBroker{
		GUID:      string(updatedBroker.ObjectMeta.UID),
		Name:      updatedBroker.Name,
		BrokerURL: updatedBroker.Spec.URL,
	}, nil
}

// Fetch the new catalog information from reach service-broker registered in kubernetes,
// so that it is visible in the kubernetes service-catalog.
func (pc *PlatformClient) Fetch(ctx context.Context, serviceBroker *platform.ServiceBroker) error {
	return pc.platformAPI.SyncClusterServiceBroker(serviceBroker.Name, resyncBrokerRetryCount)
}

func newServiceBroker(name string, url string, secret *v1beta1.ObjectReference) *v1beta1.ClusterServiceBroker {
	return &v1beta1.ClusterServiceBroker{
		ObjectMeta: v1.ObjectMeta{
			Name: name,
		},
		Spec: v1beta1.ClusterServiceBrokerSpec{
			CommonServiceBrokerSpec: v1beta1.CommonServiceBrokerSpec{
				URL: url,
			},
			AuthInfo: &v1beta1.ClusterServiceBrokerAuthInfo{
				Basic: &v1beta1.ClusterBasicAuthConfig{
					SecretRef: secret,
				},
			},
		},
	}
}

// GetVisibilitiesByBrokers get currently available visibilities in the platform for specific broker names
func (pc *PlatformClient) GetVisibilitiesByBrokers(ctx context.Context, brokers []string) ([]*platform.Visibility, error) {
	// This will cause all brokers to re-fetch their catalogs
	return nil, nil
}

// VisibilityScopeLabelKey returns a specific label key which should be used when converting SM visibilities to platform.Visibilities
func (pc *PlatformClient) VisibilityScopeLabelKey() string {
	return ""
}

// EnableAccessForPlan enables the access for the specified plan
func (pc *PlatformClient) EnableAccessForPlan(ctx context.Context, request *platform.ModifyPlanAccessRequest) error {
	return pc.Fetch(ctx, &platform.ServiceBroker{
		Name: request.BrokerName,
	})
}

// DisableAccessForPlan disables the access for the specified plan
func (pc *PlatformClient) DisableAccessForPlan(ctx context.Context, request *platform.ModifyPlanAccessRequest) error {
	return pc.Fetch(ctx, &platform.ServiceBroker{
		Name: request.BrokerName,
	})
}
