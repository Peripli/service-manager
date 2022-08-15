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
	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/web"
)

const (
	SupportedEncodingsFilterName = "SupportedEncodingsFilter"
)

type SupportedEncodingsFilter struct {
}

func (*SupportedEncodingsFilter) Name() string {
	return SupportedEncodingsFilterName
}

/** Allow to proxy only SM supported osb encodings (Identity/none are currently allowed)  */
func (l *SupportedEncodingsFilter) Run(req *web.Request, next web.Handler) (*web.Response, error) {
	req.Header.Del("Accept-Encoding")
	return next.Handle(req)
}

func (*SupportedEncodingsFilter) FilterMatchers() []web.FilterMatcher {
	return []web.FilterMatcher{
		{
			Matchers: []web.Matcher{
				web.Path(web.OSBURL + "/**"),
			},
		},
	}
}
