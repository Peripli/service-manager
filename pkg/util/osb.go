package util

import osbc "github.com/pmorie/go-open-service-broker-client/v2"

func NewOSBClient(skipSsl bool) osbc.CreateFunc {
	return func(configuration *osbc.ClientConfiguration) (osbc.Client, error) {
		configuration.Insecure = skipSsl
		return osbc.NewClient(configuration)
	}
}
