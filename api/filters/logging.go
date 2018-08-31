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

	"github.com/Peripli/service-manager/pkg/log"
	"github.com/Peripli/service-manager/pkg/web"
)

const (
	// LoggingFilterName is the name of the logging filter
	LoggingFilterName = "LoggingFilter"
	logLevelHeader    = "X-Log-Level"
)

// Logging is filter that configures logging per request.
type Logging struct {
}

// Name implements the web.Filter interface and returns the identifier of the filter.
func (*Logging) Name() string {
	return LoggingFilterName
}

// Run represents the logging middleware function that processes the request and configures the request-scoped logging.
func (l *Logging) Run(req *web.Request, next web.Handler) (*web.Response, error) {
	correlationID := web.CorrelationIDFromHeaders(req.Header)
	entry := log.R(req, LoggingFilterName).WithField(log.FieldCorrelationID, correlationID)
	ctx := log.ContextWithLogger(req.Context(), entry)
	if requestLogLevel, exists := req.Header[logLevelHeader]; exists {
		entry.Debugf("Changing request log level to %s", requestLogLevel)
		ctx = context.WithValue(ctx, log.LevelKey{}, requestLogLevel[0])
	}
	req.Request = req.WithContext(ctx)
	return next.Handle(req)
}

// FilterMatchers implements the web.Filter interface and returns the conditions on which the filter should be executed.
func (*Logging) FilterMatchers() []web.FilterMatcher {
	return []web.FilterMatcher{
		{
			Matchers: []web.Matcher{
				web.Path("/**"),
			},
		},
	}
}
