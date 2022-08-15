package filters

import (
	"fmt"
	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/util"
	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/web"
	"net/http"
)

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

const DisabledQueryParametersName = "DisabledQueryParameter"

type DisabledQueryParametersFilter struct {
	DisabledQueryParameters []string
}

func (f *DisabledQueryParametersFilter) Name() string {
	return DisabledQueryParametersName
}

func (f *DisabledQueryParametersFilter) Run(req *web.Request, next web.Handler) (*web.Response, error) {
	for _, param := range f.DisabledQueryParameters {
		paramValue := req.URL.Query().Get(param)
		if paramValue != "" {
			return nil, &util.HTTPError{
				ErrorType:   "NotImplemented",
				Description: fmt.Sprintf("The '%s' parameter is not supported in this service manager installation.", param),
				StatusCode:  http.StatusNotImplemented,
			}

		}
	}
	return next.Handle(req)
}

func (f *DisabledQueryParametersFilter) FilterMatchers() []web.FilterMatcher {
	return []web.FilterMatcher{
		{
			Matchers: []web.Matcher{
				web.Path(web.ServiceOfferingsURL),
				web.Methods(http.MethodGet),
			},
		},
		{
			Matchers: []web.Matcher{
				web.Path(web.ServicePlansURL),
				web.Methods(http.MethodGet),
			},
		},
	}
}
