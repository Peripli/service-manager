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

package info

import (
	"net/http"
	"github.com/Peripli/service-manager/server"
	"io/ioutil"
	"github.com/Peripli/service-manager/rest"
	"github.com/sirupsen/logrus"
	"encoding/json"
	"strings"
)

type controller struct {
	info map[string]string
}

func NewController(environment server.Environment) rest.Controller {
	bytes, err := ioutil.ReadFile("api/info/info.json")
	if err != nil {
		logrus.Panicf("Cannot read info file")
	}
	var infoJson map[string]string
	if err := json.Unmarshal(bytes, &infoJson); err != nil {
		logrus.Panic("Invalid JSON read from file")
	}
	ctrl := &controller{
		info: make(map[string]string),
	}
	for key, value := range infoJson {
		envVar := strings.Trim(value, "{}")
		actualValue, ok := environment.Get(envVar).(string)
		if !ok {
			logrus.Panicf("Variable %s is not set!", envVar)
		}
		ctrl.info[key] = actualValue
	}
	return ctrl
}

func (c *controller) getInfo(writer http.ResponseWriter, request *http.Request) error {
	rest.SendJSON(writer, http.StatusOK, c.info)
	return nil
}
