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

package oauth

import (
	"github.com/Peripli/service-manager/authorization"
	"github.com/sirupsen/logrus"
)

type ScopeAuthorizer struct {
	RequiredScopes []string
	Resource       string
}

func (a *ScopeAuthorizer) Authorize(attributes authorization.Attributes) (authorization.Decision, error) {
	if a.Resource != attributes.Resource {
		logrus.Debugf("Abstaining from deciding access for resource %s...", attributes.Resource)
		// can't decide for this resource
		return authorization.DecisionAbstain, nil
	}
	if a.hasAny(a.RequiredScopes, attributes.User.Scopes) {
		return authorization.DecisionAllow, nil
	}
	return authorization.DecisionDeny, nil
}

func (a *ScopeAuthorizer) hasAny(c1, c2 []string) bool {
	m := make(map[string]bool)
	for _, v := range c1 {
		m[v] = true
	}
	for _, v := range c2 {
		if m[v] {
			return true
		}
	}
	return false
}
