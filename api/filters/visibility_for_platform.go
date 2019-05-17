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
	"errors"
	"net/http"

	"github.com/Peripli/service-manager/pkg/query"
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/pkg/web"
)

const PlatformAwareVisibilityFilterName = "PlatformAwareVisibilityFilter"

type PlatformAwareVisibilityFilter struct {
}

func (*PlatformAwareVisibilityFilter) Name() string {
	return PlatformAwareVisibilityFilterName
}

func (*PlatformAwareVisibilityFilter) Run(req *web.Request, next web.Handler) (*web.Response, error) {
	ctx := req.Context()
	user, ok := web.UserFromContext(ctx)
	if !ok {
		return nil, errors.New("user details not found in request context")
	}

	p := &types.Platform{}
	if err := user.Data.Data(p); err != nil {
		return nil, err
	}

	if p.ID != "" {
		byPlatformID := query.ByField(query.EqualsOrNilOperator, "platform_id", p.ID)
		var err error
		if ctx, err = query.AddCriteria(ctx, byPlatformID); err != nil {
			return nil, err
		}
		req.Request = req.WithContext(ctx)
	}
	return next.Handle(req)
}

func (*PlatformAwareVisibilityFilter) FilterMatchers() []web.FilterMatcher {
	return []web.FilterMatcher{
		{
			Matchers: []web.Matcher{
				web.Path(web.VisibilitiesURL + "/**"),
				web.Methods(http.MethodGet),
			},
		},
	}
}
