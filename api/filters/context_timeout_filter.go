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
	"context"
	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/web"
	"net/http"
	"time"
)

const (
	// ContextTimeoutFilterName is the name of the logging filter
	ContextTimeoutFilterName = "ContextTimeoutFilter"
)

// contextTimeout is filter that configures logging per request.
type contextTimeout struct {
	timeout time.Duration
}

// Name implements the web.Filter interface and returns the identifier of the filter.
func (*contextTimeout) Name() string {
	return ContextTimeoutFilterName
}

func NewContextTimeoutFilter(timeout time.Duration) *contextTimeout {
	return &contextTimeout{
		timeout: timeout,
	}
}

// Run represents the logging middleware function that processes the request and configures the request-scoped logging.
func (l *contextTimeout) Run(req *web.Request, next web.Handler) (*web.Response, error) {
	ctx, cancel := context.WithTimeout(req.Context(), l.timeout)
	defer cancel()
	req.Request = req.WithContext(ctx)
	return next.Handle(req)
}

// FilterMatchers implements the web.Filter interface and returns the conditions on which the filter should be executed.
func (*contextTimeout) FilterMatchers() []web.FilterMatcher {
	return []web.FilterMatcher{
		{
			Matchers: []web.Matcher{
				web.Path(web.OSBURL + "/*/v2/service_instances/*"),
				web.Methods(http.MethodPut),
			},
		},
	}
}
