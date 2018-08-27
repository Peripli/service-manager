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
	"github.com/Peripli/service-manager/pkg/log"
	"github.com/Peripli/service-manager/pkg/util/slice"
	"github.com/Peripli/service-manager/pkg/web"
	"github.com/gofrs/uuid"
)

var CorrelationIdHeaders = []string{"X-Correlation-ID", "X-CorrelationID", "X-ForRequest-ID"}

const LoggingFilterName = "LoggingFilter"

type Logging struct {
}

func (*Logging) Name() string {
	return LoggingFilterName
}

func (l *Logging) Run(req *web.Request, next web.Handler) (*web.Response, error) {
	var correlationId string
	for key, val := range req.Header {
		if slice.StringsAnyEquals(CorrelationIdHeaders, key) {
			correlationId = val[0]
			break
		}
	}
	if correlationId == "" {
		uuids, err := uuid.NewV4()
		if err != nil {
			return nil, err
		}
		correlationId = uuids.String()
	}
	entry := log.R(req, LoggingFilterName).WithField("correlation_id", correlationId)
	ctx := log.ContextWithLogger(req.Context(), entry)
	req.Request = req.WithContext(ctx)
	return next.Handle(req)
}

func (*Logging) FilterMatchers() []web.FilterMatcher {
	return []web.FilterMatcher{
		{
			Matchers: []web.Matcher{
				web.Path("/**"),
			},
		},
	}
}
