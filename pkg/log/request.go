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

package log

import (
	"net/http"

	"github.com/gofrs/uuid"
)

// CorrelationIDHeaders are the headers whose values will be taken as a correlation id for incoming requests
var CorrelationIDHeaders = []string{"X-Correlation-ID", "X-CorrelationID", "X-ForRequest-ID", "X-Request-ID"}

// CorrelationIDForRequest returns checks the http headers for any of the supported correlation id headers.
// The first that matches is taken as the correlation id. If none exists a new one is generated.
func CorrelationIDForRequest(request *http.Request) string {
	for _, header := range CorrelationIDHeaders {
		headerValue := request.Header.Get(header)
		if headerValue != "" {
			return headerValue
		}
	}
	newCorrelationID := ""
	uuids, err := uuid.NewV4()
	if err == nil {
		newCorrelationID = uuids.String()
		request.Header.Set(CorrelationIDHeaders[0], newCorrelationID)
	}
	return newCorrelationID
}
