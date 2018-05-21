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
	"github.com/Peripli/service-manager/rest"
)

type informationResponse struct{
	TokenIssuer string `mapstructure:"token_issuer_url" json:"token_issuer_url"`
}

type controller struct {
	info informationResponse
}

// NewController returns a new info controller
func NewController(environment server.Environment) rest.Controller {
	var info informationResponse
	if err := environment.Unmarshal(&info); err != nil {
		panic(err)
	}
	return &controller{info}
}

func (c *controller) getInfo(writer http.ResponseWriter, request *http.Request) error {
	return rest.SendJSON(writer, http.StatusOK, c.info)
}
