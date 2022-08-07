/*
 * Copyright 2018 The Service Manager Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package env

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/Peripli/service-manager/pkg/auth/util"
	"github.com/Peripli/service-manager/pkg/log"
	"github.com/cloudfoundry-community/go-cfenv"
	"github.com/spf13/cast"
)

// setCFOverrides overrides some SM environment with values from CF's VCAP environment variables
func setCFOverrides(env Environment) error {
	if _, exists := os.LookupEnv("VCAP_APPLICATION"); exists {
		cfEnv, err := cfenv.Current()
		if err != nil {
			return fmt.Errorf("could not load VCAP environment: %s", err)
		}

		env.Set("server.port", cfEnv.Port)

		pgServiceName := cast.ToString(env.Get("storage.name"))
		if pgServiceName == "" {
			log.D().Warning("No PostgreSQL service name found")
			return nil
		}
		service, err := cfEnv.Services.WithName(pgServiceName)
		if err != nil {
			return fmt.Errorf("could not find service with name %s: %v", pgServiceName, err)
		}
		env.Set("storage.uri", service.Credentials["uri"])
		if err := setPostgresSSL(env, service.Credentials); err != nil {
			return err
		}

		cacheEnabled := cast.ToBool(env.Get("cache.enabled"))
		if cacheEnabled {
			if err := useRedisBinding(env, cfEnv); err != nil {
				return err
			}
		}

	}
	return nil
}

func useRedisBinding(env Environment, cfEnv *cfenv.App) error {
	redisServiceName := cast.ToString(env.Get("cache.name"))
	if redisServiceName == "" {
		log.D().Warning("No Redis service name found")
		return nil
	}
	service, err := cfEnv.Services.WithName(redisServiceName)
	if err != nil {
		errorMessage := fmt.Sprintf("could not find service with name %s: %v", redisServiceName, err)
		log.D().Error(errorMessage)
		return fmt.Errorf(errorMessage)
	}

	env.Set("cache.hostname", service.Credentials["hostname"])
	env.Set("cache.password", service.Credentials["password"])
	env.Set("cache.port", service.Credentials["port"])
	return nil
}

func setPostgresSSL(env Environment, credentials map[string]interface{}) error {
	if sslRootCert, hasRootCert := credentials["sslrootcert"]; hasRootCert {
		filename := "./root.crt"
		env.Set("storage.sslmode", "verify-ca")
		env.Set("storage.sslrootcert", filename)
		sslRootCertStr := util.ConvertBackSlashN(sslRootCert.(string))
		err := ioutil.WriteFile(filename, []byte(sslRootCertStr), 0666)
		if err != nil {
			return err
		}
	}
	return nil
}
