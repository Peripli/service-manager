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
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/Peripli/service-manager/pkg/security/filters"

	"github.com/Peripli/service-manager/pkg/types"

	"github.com/Peripli/service-manager/pkg/query"

	httpsec "github.com/Peripli/service-manager/pkg/security/http"

	"github.com/Peripli/service-manager/pkg/web"
	"github.com/Peripli/service-manager/storage"
)

const BasicAuthnFilterName string = "BasicAuthnFilter"

func NewBasicAuthnFilter(repository storage.Repository) *filters.AuthenticationFilter {
	return filters.NewAuthenticationFilter(&basicAuthenticator{
		Repository: repository,
	}, BasicAuthnFilterName, basicAuthnMatchers())
}

type basicAuthnData struct {
	data json.RawMessage
}

func (bad *basicAuthnData) Data(v interface{}) error {
	return json.Unmarshal([]byte(bad.data), v)
}

// basicAuthenticator for basic security
type basicAuthenticator struct {
	Repository storage.Repository
}

// Authenticate authenticates by using the provided Basic credentials
func (a *basicAuthenticator) Authenticate(request *http.Request) (*web.UserContext, httpsec.Decision, error) {
	username, password, ok := request.BasicAuth()
	if !ok {
		return nil, httpsec.Abstain, nil
	}

	ctx := request.Context()
	byUsername := query.ByField(query.EqualsOperator, "username", username)
	objectList, err := a.Repository.List(ctx, types.PlatformType, byUsername)
	if err != nil {
		return nil, httpsec.Abstain, fmt.Errorf("could not get credentials entity from storage: %s", err)
	}

	if objectList.Len() != 1 {
		return nil, httpsec.Deny, fmt.Errorf("provided credentials are invalid")
	}

	obj := objectList.ItemAt(0)
	securedObj, isSecured := obj.(types.Secured)
	if !isSecured {
		return nil, httpsec.Abstain, fmt.Errorf("object of type %s is used in authentication and must be secured", obj.GetType())
	}

	if securedObj.GetCredentials().Basic.Password != password {
		return nil, httpsec.Deny, fmt.Errorf("provided credentials are invalid")
	}

	bytes, err := json.Marshal(obj)
	if err != nil {
		return nil, httpsec.Abstain, err
	}

	return &web.UserContext{
		Data: &basicAuthnData{
			data: bytes,
		},
		Name: username,
	}, httpsec.Allow, nil
}

func basicAuthnMatchers() []web.FilterMatcher {
	return []web.FilterMatcher{
		{
			Matchers: []web.Matcher{
				web.Path(web.OSBURL + "/**"),
			},
		},
		{
			Matchers: []web.Matcher{
				web.Methods(http.MethodGet),
				web.Path(web.PlatformsURL + "/**"),
			},
		},
		{
			Matchers: []web.Matcher{
				web.Methods(http.MethodGet),
				web.Path(web.ServiceBrokersURL + "/**"),
			},
		},
		{
			Matchers: []web.Matcher{
				web.Methods(http.MethodGet),
				web.Path(web.ServiceOfferingsURL + "/**"),
			},
		},
		{
			Matchers: []web.Matcher{
				web.Methods(http.MethodGet),
				web.Path(web.ServicePlansURL + "/**"),
			},
		},
		{
			Matchers: []web.Matcher{
				web.Methods(http.MethodGet),
				web.Path(web.VisibilitiesURL + "/**"),
			},
		},
		{
			Matchers: []web.Matcher{
				web.Methods(http.MethodGet),
				web.Path(web.NotificationsURL + "/**"),
			},
		},
	}
}
