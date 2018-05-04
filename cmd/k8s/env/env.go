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
	"os"
	"strings"

	"github.com/Peripli/service-manager/server"
)

// New returns a K8S environment with a delegate
func New(delegate server.Environment) server.Environment {
	return &k8sEnvironment{Environment: delegate}
}

type k8sEnvironment struct {
	server.Environment
}

func (e *k8sEnvironment) Load() error {
	var err error
	if err = e.Environment.Load(); err != nil {
		return err
	}

	uri := e.databaseURI()
	e.Environment.Set("db.uri", uri)
	return err
}

func (e *k8sEnvironment) Get(key string) interface{} {
	return e.Environment.Get(key)
}

func (e *k8sEnvironment) databaseURI() string {
	dbName := e.Environment.Get("db.name").(string)
	dbName = strings.ToUpper(dbName)
	dbName = strings.Replace(dbName, "-", "_", -1)

	dbHost := os.Getenv(dbName + "_SERVICE_HOST")
	dbPort := os.Getenv(dbName + "_SERVICE_PORT")

	dbUser := e.Environment.Get("db.username")
	dbPassword := e.Environment.Get("db.password")

	return fmt.Sprintf("postgres://%s:%s@%s:%s/postgres?sslmode=disable", dbUser, dbPassword, dbHost, dbPort)
}
