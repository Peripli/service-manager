package client

import (
	"context"
	"errors"
	"testing"

	"github.com/Peripli/service-manager/pkg/agent"

	"github.com/Peripli/service-manager/cmd/k8s-agent/k8s/api/apifakes"

	"github.com/Peripli/service-manager/cmd/k8s-agent/k8s/config"
	"github.com/Peripli/service-manager/pkg/agent/platform"

	"os"

	"github.com/kubernetes-sigs/service-catalog/pkg/apis/servicecatalog/v1beta1"
	servicecatalog "github.com/kubernetes-sigs/service-catalog/pkg/svcat/service-catalog"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
)

func TestClient(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Kubernetes Proxy Tests Suite")
}

var _ = Describe("Kubernetes Broker Proxy", func() {
	var expectedError = errors.New("expected")
	var clientConfig *config.ClientConfiguration
	var proxySettings *agent.Settings
	var settings *config.Settings
	var ctx context.Context
	var k8sApi *apifakes.FakeKubernetesAPI

	newDefaultPlatformClient := func() *PlatformClient {
		client, err := NewClient(settings)
		Expect(err).ToNot(HaveOccurred())
		client.platformAPI = k8sApi
		return client
	}

	BeforeSuite(func() {
		Expect(os.Setenv("KUBERNETES_SERVICE_HOST", "test")).ToNot(HaveOccurred())
		Expect(os.Setenv("KUBERNETES_SERVICE_PORT", "1234")).ToNot(HaveOccurred())
	})

	BeforeEach(func() {
		clientConfig = config.DefaultClientConfiguration()
		clientConfig.ClientSettings.NewClusterConfig = func(_ string) (*rest.Config, error) {
			return &rest.Config{
				Host:            "https://fakeme",
				BearerToken:     string("faketoken"),
				TLSClientConfig: rest.TLSClientConfig{},
			}, nil
		}
		clientConfig.Secret.Name = "secretName"
		clientConfig.Secret.Namespace = "secretNamespace"
		clientConfig.K8sClientCreateFunc = config.NewSvcatSDK

		proxySettings = agent.DefaultSettings()
		proxySettings.Sm.User = "user"
		proxySettings.Sm.Password = "pass"
		proxySettings.Sm.URL = "url"
		proxySettings.Reconcile.LegacyURL = "legacy_url"
		proxySettings.Reconcile.URL = "reconcile_url"

		settings = &config.Settings{
			Settings: *proxySettings,
			K8S:      clientConfig,
		}
		ctx = context.TODO()
		k8sApi = &apifakes.FakeKubernetesAPI{}
	})

	Describe("New Client", func() {
		Context("With invalid config", func() {
			It("should return error", func() {
				settings.K8S = config.DefaultClientConfiguration()
				_, err := NewClient(settings)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Equal("properties of K8S secret configuration for broker registration missing"))
			})
		})

		Context("With invalid config", func() {
			It("should return error", func() {
				clientConfig.K8sClientCreateFunc = func(libraryConfig *config.LibraryConfig) (*servicecatalog.SDK, error) {
					return nil, expectedError
				}
				_, err := NewClient(settings)
				Expect(err).To(Equal(expectedError))
			})
		})

		Context("With valid config", func() {

			It("should handle broker operations", func() {
				client := newDefaultPlatformClient()
				Expect(client.Broker()).ToNot(BeNil())
			})

			It("should handle catalog fetch operations", func() {
				client := newDefaultPlatformClient()
				Expect(client.CatalogFetcher()).ToNot(BeNil())
			})

			It("should handle visibility operations", func() {
				client := newDefaultPlatformClient()
				Expect(client.Visibility()).ToNot(BeNil())
			})
		})
	})

	Describe("Create a service broker", func() {

		Context("with no error", func() {
			It("returns broker", func() {
				platformClient := newDefaultPlatformClient()

				k8sApi.CreateClusterServiceBrokerStub = func(broker *v1beta1.ClusterServiceBroker) (*v1beta1.ClusterServiceBroker, error) {
					return &v1beta1.ClusterServiceBroker{
						ObjectMeta: v1.ObjectMeta{
							UID:  "1234",
							Name: broker.Name,
						},
						Spec: v1beta1.ClusterServiceBrokerSpec{
							CommonServiceBrokerSpec: v1beta1.CommonServiceBrokerSpec{
								URL: broker.Spec.URL,
							},
						},
					}, nil
				}

				requestBroker := &platform.CreateServiceBrokerRequest{
					Name:      "fake-broker",
					BrokerURL: "http://fake.broker.url",
				}
				createdBroker, err := platformClient.CreateBroker(ctx, requestBroker)

				Expect(err).To(BeNil())
				Expect(createdBroker.GUID).To(Equal("1234"))
				Expect(createdBroker.Name).To(Equal("fake-broker"))
				Expect(createdBroker.BrokerURL).To(Equal("http://fake.broker.url"))
			})
		})

		Context("with an error", func() {
			It("returns error", func() {
				platformClient := newDefaultPlatformClient()

				k8sApi.CreateClusterServiceBrokerStub = func(broker *v1beta1.ClusterServiceBroker) (*v1beta1.ClusterServiceBroker, error) {
					return nil, errors.New("error from service-catalog")
				}

				requestBroker := &platform.CreateServiceBrokerRequest{}
				createdBroker, err := platformClient.CreateBroker(ctx, requestBroker)

				Expect(createdBroker).To(BeNil())
				Expect(err).To(Equal(errors.New("error from service-catalog")))
			})
		})
	})

	Describe("Delete a service broker", func() {
		Context("with no error", func() {
			It("returns no error", func() {
				platformClient := newDefaultPlatformClient()

				k8sApi.DeleteClusterServiceBrokerStub = func(name string, options *v1.DeleteOptions) error {
					return nil
				}

				requestBroker := &platform.DeleteServiceBrokerRequest{
					GUID: "1234",
					Name: "fake-broker",
				}

				err := platformClient.DeleteBroker(ctx, requestBroker)

				Expect(err).To(BeNil())
			})
		})

		Context("with an error", func() {
			It("returns the error", func() {
				platformClient := newDefaultPlatformClient()

				k8sApi.DeleteClusterServiceBrokerStub = func(name string, options *v1.DeleteOptions) error {
					return errors.New("error deleting clusterservicebroker")
				}

				requestBroker := &platform.DeleteServiceBrokerRequest{}

				err := platformClient.DeleteBroker(ctx, requestBroker)

				Expect(err).To(Equal(errors.New("error deleting clusterservicebroker")))
			})
		})
	})

	Describe("Get all service brokers", func() {
		Context("with no error", func() {
			It("returns brokers", func() {
				platformClient := newDefaultPlatformClient()

				k8sApi.RetrieveClusterServiceBrokersStub = func() (*v1beta1.ClusterServiceBrokerList, error) {
					brokers := make([]v1beta1.ClusterServiceBroker, 0)
					brokers = append(brokers, v1beta1.ClusterServiceBroker{
						ObjectMeta: v1.ObjectMeta{
							UID:  "1234",
							Name: "fake-broker",
						},
						Spec: v1beta1.ClusterServiceBrokerSpec{
							CommonServiceBrokerSpec: v1beta1.CommonServiceBrokerSpec{
								URL: "http://fake.broker.url",
							},
						},
					})
					return &v1beta1.ClusterServiceBrokerList{
						Items: brokers,
					}, nil
				}

				brokers, err := platformClient.GetBrokers(ctx)

				Expect(err).To(BeNil())
				Expect(brokers).ToNot(BeNil())
				Expect(len(brokers)).To(Equal(1))
				Expect(brokers[0].GUID).To(Equal("1234"))
				Expect(brokers[0].Name).To(Equal("fake-broker"))
				Expect(brokers[0].BrokerURL).To(Equal("http://fake.broker.url"))
			})
		})

		Context("when no service brokers are registered", func() {
			It("returns empty array", func() {
				platformClient := newDefaultPlatformClient()

				k8sApi.RetrieveClusterServiceBrokersStub = func() (*v1beta1.ClusterServiceBrokerList, error) {
					brokers := make([]v1beta1.ClusterServiceBroker, 0)
					return &v1beta1.ClusterServiceBrokerList{
						Items: brokers,
					}, nil
				}

				brokers, err := platformClient.GetBrokers(ctx)

				Expect(err).To(BeNil())
				Expect(brokers).ToNot(BeNil())
				Expect(len(brokers)).To(Equal(0))
			})
		})

		Context("with an error", func() {
			It("returns the error", func() {
				platformClient := newDefaultPlatformClient()

				k8sApi.RetrieveClusterServiceBrokersStub = func() (*v1beta1.ClusterServiceBrokerList, error) {
					return nil, errors.New("error getting clusterservicebrokers")
				}

				brokers, err := platformClient.GetBrokers(ctx)

				Expect(brokers).To(BeNil())
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("error getting clusterservicebrokers"))
			})
		})
	})

	Describe("Get service broker by name", func() {
		Context("with no error", func() {
			It("returns the service broker", func() {
				platformClient := newDefaultPlatformClient()
				brokerName := "brokerName"

				k8sApi.RetrieveClusterServiceBrokerByNameStub = func(name string) (*v1beta1.ClusterServiceBroker, error) {
					return &v1beta1.ClusterServiceBroker{
						ObjectMeta: v1.ObjectMeta{
							UID:  "1234",
							Name: brokerName,
						},
						Spec: v1beta1.ClusterServiceBrokerSpec{
							CommonServiceBrokerSpec: v1beta1.CommonServiceBrokerSpec{
								URL: "http://fake.broker.url",
							},
						},
					}, nil
				}

				broker, err := platformClient.GetBrokerByName(ctx, brokerName)

				Expect(err).To(BeNil())
				Expect(broker).ToNot(BeNil())
				Expect(broker.Name).To(Equal(brokerName))
			})
		})

		Context("with an error", func() {
			It("returns the error", func() {
				platformClient := newDefaultPlatformClient()

				k8sApi.RetrieveClusterServiceBrokerByNameStub = func(name string) (*v1beta1.ClusterServiceBroker, error) {
					return nil, errors.New("error getting clusterservicebroker")
				}

				broker, err := platformClient.GetBrokerByName(ctx, "brokerName")

				Expect(broker).To(BeNil())
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("error getting clusterservicebroker"))
			})
		})
	})

	Describe("Update a service broker", func() {
		Context("with no errors", func() {
			It("returns updated broker", func() {
				platformClient := newDefaultPlatformClient()

				k8sApi.UpdateClusterServiceBrokerStub = func(broker *v1beta1.ClusterServiceBroker) (*v1beta1.ClusterServiceBroker, error) {
					// Return a new fake clusterservicebroker with the three attributes relevant for the OSBAPI guid, name and broker url.
					// UID cannot be modified, name and url can be modified
					return &v1beta1.ClusterServiceBroker{
						ObjectMeta: v1.ObjectMeta{
							Name: broker.Name + "-updated",
							UID:  "1234",
						},
						Spec: v1beta1.ClusterServiceBrokerSpec{
							CommonServiceBrokerSpec: v1beta1.CommonServiceBrokerSpec{
								URL: broker.Spec.CommonServiceBrokerSpec.URL + "-updated",
							},
						},
					}, nil
				}

				requestBroker := &platform.UpdateServiceBrokerRequest{
					GUID:      "1234",
					Name:      "fake-broker",
					BrokerURL: "http://fake.broker.url",
				}

				broker, err := platformClient.UpdateBroker(ctx, requestBroker)

				Expect(err).To(BeNil())
				Expect(broker.GUID).To(Equal("1234"))
				Expect(broker.Name).To(Equal("fake-broker-updated"))
				Expect(broker.BrokerURL).To(Equal("http://fake.broker.url-updated"))
			})
		})

		Context("with an error", func() {
			It("returns the error", func() {
				platformClient := newDefaultPlatformClient()

				k8sApi.UpdateClusterServiceBrokerStub = func(broker *v1beta1.ClusterServiceBroker) (*v1beta1.ClusterServiceBroker, error) {
					return nil, errors.New("error updating clusterservicebroker")
				}

				requestBroker := &platform.UpdateServiceBrokerRequest{}

				broker, err := platformClient.UpdateBroker(ctx, requestBroker)

				Expect(broker).To(BeNil())
				Expect(err).To(Equal(errors.New("error updating clusterservicebroker")))
			})
		})
	})

	Describe("Fetch the catalog information of a service broker", func() {
		Context("with no errors", func() {
			It("returns nil", func() {
				platformClient := newDefaultPlatformClient()

				requestBroker := &platform.ServiceBroker{
					GUID:      "1234",
					Name:      "fake-broker",
					BrokerURL: "http://fake.broker.url",
				}

				k8sApi.SyncClusterServiceBrokerStub = func(name string, retries int) error {
					return nil
				}

				err := platformClient.Fetch(ctx, requestBroker)

				Expect(err).To(BeNil())
			})
		})

		Context("with an error", func() {
			It("returns the error", func() {
				platformClient := newDefaultPlatformClient()

				requestBroker := &platform.ServiceBroker{}
				k8sApi.SyncClusterServiceBrokerStub = func(name string, retries int) error {
					return errors.New("error syncing service broker")
				}

				err := platformClient.Fetch(ctx, requestBroker)

				Expect(err).To(Equal(errors.New("error syncing service broker")))
			})
		})
	})

	Describe("GetVisibilitiesByBrokers", func() {
		It("returns no visibilities", func() {
			platformClient := newDefaultPlatformClient()
			visibilities, err := platformClient.GetVisibilitiesByBrokers(ctx, []string{})
			Expect(err).To(BeNil())
			Expect(visibilities).To(BeNil())
		})
	})

	Describe("VisibilityScopeLabelKey", func() {
		It("returns empty string", func() {
			Expect(newDefaultPlatformClient().VisibilityScopeLabelKey()).To(BeEmpty())
		})
	})

	Describe("EnableAccessForPlan", func() {
		It("should call Fetch", func() {
			platformClient := newDefaultPlatformClient()
			k8sApi.SyncClusterServiceBrokerStub = func(name string, retries int) error {
				return expectedError
			}
			Expect(platformClient.EnableAccessForPlan(ctx, &platform.ModifyPlanAccessRequest{})).To(Equal(expectedError))
		})
	})

	Describe("DisableAccessForPlan", func() {
		It("should call Fetch", func() {
			platformClient := newDefaultPlatformClient()
			k8sApi.SyncClusterServiceBrokerStub = func(name string, retries int) error {
				return expectedError
			}
			Expect(platformClient.DisableAccessForPlan(ctx, &platform.ModifyPlanAccessRequest{})).To(Equal(expectedError))
		})
	})
})
