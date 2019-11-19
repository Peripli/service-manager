package cf

import (
	"context"
	"fmt"
	"os"
	"reflect"

	"github.com/Peripli/service-manager/pkg/agent"
	"github.com/Peripli/service-manager/pkg/env"
	"github.com/cloudfoundry-community/go-cfenv"
	"github.com/spf13/pflag"
)

// DefaultEnv creates a default environment for the CF service broker proxy
func DefaultEnv(ctx context.Context, additionalPFlags ...func(set *pflag.FlagSet)) (env.Environment, error) {
	additionalPFlagProviders := append(additionalPFlags, func(set *pflag.FlagSet) {
		CreatePFlagsForCFClient(set)
	})

	env, err := agent.DefaultEnv(ctx, additionalPFlagProviders...)
	if err != nil {
		panic(fmt.Errorf("error creating environment: %s", err))
	}

	if err := setCFOverrides(env); err != nil {
		panic(fmt.Errorf("error setting CF environment values: %s", err))
	}

	return env, nil
}

// setCFOverrides overrides some SM environment with values from CF's VCAP environment variables
func setCFOverrides(env env.Environment) error {
	if _, exists := os.LookupEnv("VCAP_APPLICATION"); exists {
		cfEnv, err := cfenv.Current()
		if err != nil {
			return fmt.Errorf("could not load VCAP environment: %s", err)
		}

		cfEnvMap := make(map[string]interface{})
		cfEnvMap["app.legacy_url"] = "https://" + cfEnv.ApplicationURIs[0]
		cfEnvMap["server.port"] = cfEnv.Port
		cfEnvMap["cf.client.apiAddress"] = cfEnv.CFAPI

		setMissingEnvironmentVariables(env, cfEnvMap)
	}
	return nil
}

func setMissingEnvironmentVariables(env env.Environment, cfEnv map[string]interface{}) {
	for key, value := range cfEnv {
		currVal := env.Get(key)
		if currVal == nil || isZeroValue(currVal) {
			env.Set(key, value)
		}
	}
}

func isZeroValue(v interface{}) bool {
	return reflect.DeepEqual(v, reflect.Zero(reflect.TypeOf(v)).Interface())
}
