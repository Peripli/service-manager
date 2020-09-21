package osb

import (
	"github.com/Peripli/service-manager/pkg/types"
	osbc "sigs.k8s.io/go-open-service-broker-client/v2"
)


var defaultBrokerClientProvider osbc.CreateFunc

// NewBrokerClientProvider provides a function which constructs an OSB client based on a provided configuration
func NewBrokerClientProvider(skipSsl bool, timeout int) osbc.CreateFunc {
	brokerClientProvider:= func(configuration *osbc.ClientConfiguration) (osbc.Client, error) {
		configuration.TimeoutSeconds = timeout
		configuration.Insecure = skipSsl
		return osbc.NewClient(configuration)
	}
	defaultBrokerClientProvider = brokerClientProvider
	return brokerClientProvider
}
func  CreateDefaultOSBClient(broker *types.ServiceBroker) (osbc.Client, error){
	return CreateOSBClient(broker,defaultBrokerClientProvider)
}

func CreateOSBClient(broker *types.ServiceBroker, osbClientFunc osbc.CreateFunc) (osbc.Client, error){
	tlsConfig, err := broker.GetTLSConfig()
	if err != nil {
		return nil,  err
	}
	osbClientConfig := &osbc.ClientConfiguration{
		Name:                broker.Name + " broker client",
		EnableAlphaFeatures: true,
		URL:                 broker.BrokerURL,
		APIVersion:          osbc.LatestAPIVersion(),
	}

	if broker.Credentials.Basic != nil {
		osbClientConfig.AuthConfig = &osbc.AuthConfig{
			BasicAuthConfig: &osbc.BasicAuthConfig{
				Username: broker.Credentials.Basic.Username,
				Password: broker.Credentials.Basic.Password,
			},
		}
	}

	if tlsConfig != nil {
		osbClientConfig.TLSConfig = tlsConfig
	}

	osbClient, err := osbClientFunc(osbClientConfig)
	if err != nil {
		return nil,  err
	}

	return osbClient, nil
}
