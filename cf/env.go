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
	"strings"

	"os"

	"github.com/Peripli/service-manager/config"
	"github.com/cloudfoundry-community/go-cfenv"
	"github.com/sirupsen/logrus"
)

// NewEnv returns a Cloud Foundry environment with a delegate
func NewEnv(delegate config.Environment) config.Environment {
	if _, exists := os.LookupEnv("VCAP_APPLICATION"); exists {
		return &cfEnvironment{Environment: delegate}
	}
	return delegate
}

type cfEnvironment struct {
	cfEnv *cfenv.App
	config.Environment
}

func (e *cfEnvironment) Load() (err error) {
	if err = e.Environment.Load(); err != nil {
		return err
	}
	var postgreServiceName string
	if serviceName := e.Environment.Get("db.name"); serviceName != nil {
		postgreServiceName = serviceName.(string)
	} else {
		logrus.Warning("No PostgreSQL service name found")
		return
	}
	if e.cfEnv, err = cfenv.Current(); err != nil {
		return err
	}
	service, err := e.cfEnv.Services.WithName(postgreServiceName)
	if err != nil {
		return fmt.Errorf("could not find service with name %s: %v", postgreServiceName, err)
	}
	e.Environment.Set("db.uri", service.Credentials["uri"].(string))
	return
}

func (e *cfEnvironment) Get(key string) interface{} {
	value, exists := cfenv.CurrentEnv()[strings.Title(key)]
	if !exists {
		return e.Environment.Get(key)
	}
	return value
}
