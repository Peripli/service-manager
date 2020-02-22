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

package authenticators

import (
	"encoding/json"
	"fmt"
	"github.com/Peripli/service-manager/pkg/types"

	"github.com/Peripli/service-manager/pkg/query"

	httpsec "github.com/Peripli/service-manager/pkg/security/http"

	"github.com/Peripli/service-manager/pkg/web"
	"github.com/Peripli/service-manager/storage"
)

// Basic for basic security
type Basic struct {
	Repository storage.Repository
}

// Authenticate authenticates by using the provided Basic credentials
func (a *Basic) Authenticate(request *web.Request) (*web.UserContext, httpsec.Decision, error) {
	username, password, ok := request.BasicAuth()
	if !ok {
		return nil, httpsec.Abstain, nil
	}

	ctx := request.Context()
	byUsername := query.ByField(query.EqualsOperator, "username", username)
	platformList, err := a.Repository.List(ctx, types.PlatformType, byUsername)
	if err != nil {
		return nil, httpsec.Abstain, fmt.Errorf("could not get credentials entity from storage: %s", err)
	}

	if platformList.Len() != 1 {
		return nil, httpsec.Deny, fmt.Errorf("provided credentials are invalid")
	}

	platform := platformList.ItemAt(0).(*types.Platform)
	if platform.Credentials.Basic.Password != password {
		return nil, httpsec.Deny, fmt.Errorf("provided credentials are invalid")
	}

	bytes, err := json.Marshal(platform)
	if err != nil {
		return nil, httpsec.Abstain, err
	}

	return &web.UserContext{
		Data: func(v interface{}) error {
			return json.Unmarshal(bytes, v)
		},
		AuthenticationType: web.Basic,
		Name:               username,
		AccessLevel:        web.NoAccess,
	}, httpsec.Allow, nil
}
