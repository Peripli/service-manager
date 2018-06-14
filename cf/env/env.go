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
	"strings"

	"github.com/Peripli/service-manager/server"
	cfenv "github.com/cloudfoundry-community/go-cfenv"
	"github.com/sirupsen/logrus"
)

// New returns a Cloud Foundry environment with a delegate
func New(delegate server.Environment) server.Environment {
	return &cfEnvironment{Environment: delegate}
}

type cfEnvironment struct {
	cfEnv *cfenv.App
	server.Environment
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
		return fmt.Errorf("Could not find service with name %s: %v", postgreServiceName, err)
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
