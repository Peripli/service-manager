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

package cf

import (
	"fmt"

	"os"

	"github.com/Peripli/service-manager/pkg/env"
	"github.com/cloudfoundry-community/go-cfenv"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cast"
)

// SetCFOverrides overrides some SM environment with values from CF's VCAP environment variables
func SetCFOverrides(env env.Environment) error {
	if _, exists := os.LookupEnv("VCAP_APPLICATION"); exists {
		cfEnv, err := cfenv.Current()
		if err != nil {
			return fmt.Errorf("could not load VCAP environment: %s", err)
		}

		env.Set("server.port", cfEnv.Port)

		pgServiceName := cast.ToString(env.Get("storage.name"))
		if pgServiceName == "" {
			logrus.Warning("No PostgreSQL service name found")
			return nil
		}
		service, err := cfEnv.Services.WithName(pgServiceName)
		if err != nil {
			return fmt.Errorf("could not find service with name %s: %v", pgServiceName, err)
		}
		env.Set("storage.uri", service.Credentials["uri"])
	}
	return nil
}
