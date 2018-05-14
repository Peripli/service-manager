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

	"github.com/Peripli/service-manager/server"
)

const configMountEnvVarName = "CONFIG_MOUNT_PATH"
const postgresMountEnvVarName = "POSTGRES_MOUNT_PATH"

func readConfig(configMount string, configName string) (string, error) {
	data, err := ioutil.ReadFile(configMount + configName)
	if err != nil {
		return "", fmt.Errorf("Could not get configuration for %s. Reason: %s", configName, err)
	}
	if len(data) == 0 {
		return "", fmt.Errorf("Configuration for %s is empty", configName)
	}
	return string(data), nil
}

func getMountPath(mountPathEnvVar string) (string, error) {
	mountPath := os.Getenv(mountPathEnvVar)
	if mountPath == "" {
		return "", fmt.Errorf("Environment variable %s not set", mountPathEnvVar)
	}
	return mountPath, nil
}

// New returns a K8S environment with a delegate
func New(delegate server.Environment) server.Environment {
	return &k8sEnvironment{Environment: delegate}
}

type k8sEnvironment struct {
	server.Environment
}

func (e *k8sEnvironment) setConfig(mountPath string, configFileName string, configName string) error {
	config, err := readConfig(mountPath, configFileName)
	if err != nil {
		return err
	}
	e.Environment.Set(configName, config)
	return nil
}

func (e *k8sEnvironment) Load() error {
	if err := e.Environment.Load(); err != nil {
		return err
	}
	configMountPath, err := getMountPath(configMountEnvVarName)
	if err != nil {
		return err
	}
	postgresMountPath, err := getMountPath(postgresMountEnvVarName)
	if err != nil {
		return err
	}

	e.setConfig(configMountPath, "logLevel", "log.level")
	e.setConfig(configMountPath, "logFormat", "log.format")
	e.setConfig(postgresMountPath, "uri", "db.uri")

	return nil
}
