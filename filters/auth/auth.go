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

package auth

import (
	"github.com/Peripli/service-manager/pkg/web"
	"github.com/Peripli/service-manager/authentication/basic"
)

// Filter which authenticates requests coming from Broker proxies using Basic Authentication
func (authFilter AuthenticationFilter) basicAuth(req *web.Request, handler web.Handler) (*web.Response, error) {
	authenticator := basic.NewAuthenticator(authFilter.CredentialsStorage)
	_, err := authenticator.Authenticate(req.Request)
	if err != nil {
		return nil, err
	}

	return handler(req)
}
