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
	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/util"
	"net/http"

	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/web"
)

// Controller info controller
type Controller struct {
	TokenIssuer string `json:"token_issuer_url"`

	// TokenBasicAuth specifies if client credentials should be sent in the header
	// as basic auth (true) or in the body (false)
	TokenBasicAuth               bool   `json:"token_basic_auth"`
	ServiceManagerTenantId       string `json:"service_manager_tenant_id"`
	ContextRSAPublicKey          string `json:"context_rsa_public_key,omitempty"`
	ContextSuccessorRSAPublicKey string `json:"context_successor_rsa_public_key,omitempty"`
}

var _ web.Controller = &Controller{}

func (c *Controller) getInfo(request *web.Request) (*web.Response, error) {
	return util.NewJSONResponse(http.StatusOK, c)
}
