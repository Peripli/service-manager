package app

import (
	"fmt"
	"os"

	"github.com/Peripli/service-manager/pkg/env"
	"github.com/cloudfoundry-community/go-cfenv"
)

// SetCFOverrides overrides some SM environment with values from CF's VCAP environment variables
func SetCFOverrides(env env.Environment) error {
	if _, exists := os.LookupEnv("VCAP_APPLICATION"); exists {
		cfEnv, err := cfenv.Current()
		if err != nil {
			return fmt.Errorf("could not load VCAP environment: %s", err)
		}

		env.Set("self_url", "https://"+cfEnv.ApplicationURIs[0])
		env.Set("server.port", cfEnv.Port)
		env.Set("cf.client.apiAddress", cfEnv.CFAPI)
	}
	return nil
}
