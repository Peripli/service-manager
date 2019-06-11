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

package filters

import (
	"github.com/Peripli/service-manager/pkg/security/filters/middlewares"
	"github.com/Peripli/service-manager/pkg/security/http"
	"github.com/Peripli/service-manager/pkg/web"
)

func NewAuthenticationFilter(authenticator http.Authenticator, name string, matchers []web.FilterMatcher) *AuthenticationFilter {
	return &AuthenticationFilter{
		Authentication: &middlewares.Authentication{
			Authenticator: authenticator,
		},
		matchers: matchers,
		name:     name,
	}
}

type AuthenticationFilter struct {
	*middlewares.Authentication

	matchers []web.FilterMatcher
	name     string
}

func (af *AuthenticationFilter) Name() string {
	return af.name
}

func (af *AuthenticationFilter) FilterMatchers() []web.FilterMatcher {
	return af.matchers
}
