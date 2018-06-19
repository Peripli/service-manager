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

	"github.com/Peripli/service-manager/config"
	"github.com/cloudfoundry-community/go-cfenv"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cast"
)

// NewEnv returns a Cloud Foundry environment with a delegate
func NewEnv(delegate config.Environment) config.Environment {
	if _, exists := os.LookupEnv("VCAP_APPLICATION"); exists {
		return &cfEnvironment{Environment: config.NewEnv()}
	}
	return delegate
}

type cfEnvironment struct {
	config.Environment
}

func (e *cfEnvironment) Load() error {
	if err := e.Environment.Load(); err != nil {
		return err
	}
	pgServiceName := cast.ToString(e.Environment.Get("db_name"))
	if pgServiceName == "" {
		logrus.Warning("No PostgreSQL service name found")
		return nil
	}
	cfEnv, err := cfenv.Current()
	if err != nil {
		return err
	}
	service, err := cfEnv.Services.WithName(pgServiceName)
	if err != nil {
		return fmt.Errorf("could not find service with name %s: %v", pgServiceName, err)
	}
	e.Environment.Set("db_uri", service.Credentials["uri"])
	return nil
}
