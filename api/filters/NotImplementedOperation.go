package filters


import (
	"fmt"
	"github.com/Peripli/service-manager/pkg/util"
	"github.com/Peripli/service-manager/pkg/web"
	"github.com/tidwall/gjson"
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



const NotImplementedOperationName = "NotImplementedOperation"

type NotImplementedOperationFilter struct {
}

func (*NotImplementedOperationFilter) Name() string {
	return NotImplementedOperationName
}

func (*NotImplementedOperationFilter) Run(req *web.Request, next web.Handler) (*web.Response, error) {
	environmentParam:=req.URL.Query().Get(web.QueryParamEnvironment)
	if environmentParam!="" {
		return nil, &util.HTTPError{
			ErrorType:   "NotImplemented",
			Description: fmt.Sprintf("The server doesn't support %s operation. You should extend service manager to support it.", web.QueryParamEnvironment),
			StatusCode:  http.StatusNotImplemented,
		}
	}

	return next.Handle(req)
}

func (*NotImplementedOperationFilter) FilterMatchers() []web.FilterMatcher {
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

