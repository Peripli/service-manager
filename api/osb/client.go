package osb

import osbc "sigs.k8s.io/go-open-service-broker-client/v2"

// NewBrokerClientProvider provides a function which constructs an OSB client based on a provided configuration
func NewBrokerClientProvider(skipSsl bool, timeout int) osbc.CreateFunc {
	return func(configuration *osbc.ClientConfiguration) (osbc.Client, error) {
		configuration.TimeoutSeconds = timeout
		configuration.Insecure = skipSsl
		return osbc.NewClient(configuration)
	}
}
