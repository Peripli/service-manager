/*
 *    Copyright 2018 The Service Manager Authors
 *
 *    Licensed under the Apache License, Version 2.0 (the "License");
 *    you may not use this file except in compliance with the License.
 *    You may obtain a copy of the License at
 *
 *        http://www.apache.org/licenses/LICENSE-2.0
 *
 *    Unless required by applicable law or agreed to in writing, software
 *    distributed under the License is distributed on an "AS IS" BASIS,
 *    WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *    See the License for the specific language governing permissions and
 *    limitations under the License.
 */

package common

import (
	"context"
	"fmt"
	"net/http"

	"github.com/Peripli/service-manager/sm"
	"github.com/gavv/httpexpect"
	"github.com/gorilla/mux"
	. "github.com/onsi/ginkgo"
	"github.com/sirupsen/logrus"
	"github.com/Peripli/service-manager/config"
)

type Object = map[string]interface{}
type Array = []interface{}

func GetServerRouter() *mux.Router {
	serverEnv := config.NewEnv()
	srv, err := sm.New(context.Background(), serverEnv)
	if err != nil {
		logrus.Fatal("Error creating server: ", err)
	}
	return srv.Router
}

func MapContains(actual Object, expected Object) {
	for k, v := range expected {
		value, ok := actual[k]
		if !ok {
			Fail(fmt.Sprintf("Missing property '%s'", k), 1)
		}
		if value != v {
			Fail(
				fmt.Sprintf("For property '%s':\nExpected: %s\nActual: %s", k, v, value),
				1)
		}
	}
}

func RemoveAllBrokers(SM *httpexpect.Expect) {
	removeAll(SM, "brokers", "/v1/service_brokers")
}

func RemoveAllPlatforms(SM *httpexpect.Expect) {
	removeAll(SM, "platforms", "/v1/platforms")
}

func removeAll(SM *httpexpect.Expect, entity string, rootURLPath string) {
	By("remove all " + entity)
	resp := SM.GET(rootURLPath).
		Expect().Status(http.StatusOK).JSON().Object()
	for _, val := range resp.Value(entity).Array().Iter() {
		id := val.Object().Value("id").String().Raw()
		SM.DELETE(rootURLPath + "/" + id).
			Expect().Status(http.StatusOK)
	}
}

func MakeBroker(name string, url string, description string) Object {
	return Object{
		"name":        name,
		"broker_url":  url,
		"description": description,
		"credentials": Object{
			"basic": Object{
				"username": "buser",
				"password": "bpass",
			},
		},
	}
}

func MakePlatform(id string, name string, atype string, description string) Object {
	return Object{
		"id":          id,
		"name":        name,
		"type":        atype,
		"description": description,
	}
}
