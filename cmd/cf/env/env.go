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
	"github.com/cloudfoundry-community/go-cfenv"
	"github.com/Peripli/service-manager/server"
	"github.com/sirupsen/logrus"
	"strings"
)

// New returns a Cloud Foundry environment with a delegate
func New(delegate server.Environment) server.Environment {
	return &cfEnvironment{delegate: delegate}
}

type cfEnvironment struct {
	cfEnv    *cfenv.App
	delegate server.Environment
}

func (e *cfEnvironment) Load() error {
	var err error
	if err = e.delegate.Load(); err != nil {
		return err
	}
	e.cfEnv, err = cfenv.Current()
	e.delegate.Set("db.uri", e.databaseURI())
	return err
}

func (e *cfEnvironment) Get(key string) interface{} {
	value, exists := cfenv.CurrentEnv()[strings.Title(key)]
	if !exists {
		return e.delegate.Get(key)
	}
	return value
}

func (e *cfEnvironment) Set(key string, value interface{}) {
	e.delegate.Set(key, value)
}

func (e *cfEnvironment) Unmarshal(value interface{}) error {
	return e.delegate.Unmarshal(value)
}

func (e *cfEnvironment) databaseURI() string {
	dbName := e.delegate.Get("db.name").(string)
	service, err := e.cfEnv.Services.WithName(dbName)
	if err != nil {
		logrus.Panicf("Could not find service with name %s", dbName)
	}
	return service.Credentials["uri"].(string)
}
