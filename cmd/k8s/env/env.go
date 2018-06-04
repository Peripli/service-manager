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
	"path/filepath"

	"github.com/Peripli/service-manager/server"
)

// K8SConfigLocationEnvVarName location of the config file for k8s deployment
const K8SConfigLocationEnvVarName = "SM_CONFIG_LOCATION"

// K8SPostgresConfigLocationEnvVarName location of the PostgreSQL config file for k8s deployment
const K8SPostgresConfigLocationEnvVarName = "SM_POSTGRES_CONFIG_LOCATION"

func readConfig(configMount string, configName string) (string, error) {
	data, err := ioutil.ReadFile(filepath.Join(configMount, configName))
	if err != nil {
		return "", fmt.Errorf("Could not get configuration for %s. Reason: %s", configName, err)
	}
	if len(data) == 0 {
		return "", fmt.Errorf("Configuration for %s is empty", configName)
	}
	return string(data), nil
}

// New returns a K8S environment with a delegate
func New(delegate server.Environment) server.Environment {
	return &k8sEnvironment{Environment: delegate}
}

type k8sEnvironment struct {
	server.Environment
}

func (e *k8sEnvironment) Load() error {
	if err := e.Environment.Load(); err != nil {
		return err
	}
	postgresMountPath := os.Getenv(K8SPostgresConfigLocationEnvVarName)
	if postgresMountPath == "" {
		return fmt.Errorf("Expected %s environment variable to be set", K8SPostgresConfigLocationEnvVarName)
	}
	postgresURI, err := readConfig(postgresMountPath, "uri")
	if err != nil {
		return err
	}
	e.Environment.Set("db.uri", postgresURI)

	return nil
}
