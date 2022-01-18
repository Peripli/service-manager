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
	"net/http"

	"github.com/Peripli/service-manager/pkg/log"
	"github.com/Peripli/service-manager/pkg/web"
	"github.com/Peripli/service-manager/pkg/web/webfakes"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Logging Filter", func() {
	loggingFilter := &Logging{}
	var request *web.Request
	var handler *webfakes.FakeHandler
	BeforeEach(func() {
		request = &web.Request{Request: &http.Request{}}
		request.Header = http.Header{}
		handler = &webfakes.FakeHandler{}
	})
	Describe("Correlation Id", func() {
		Context("When none is provided in header", func() {
			It("Should generate a new for logger", func() {
				loggingFilter.Run(request, handler)
				logger := log.C(request.Context())
				correlationId := logger.Data[log.FieldCorrelationID].(string)
				Expect(correlationId).ToNot(BeEmpty())
			})
		})
		Context("When one is provided in header", func() {
			It("Uses it for logger", func() {
				expectedCorrelationId := "correlationId"
				request.Header.Set("X-Correlation-ID", expectedCorrelationId)
				loggingFilter.Run(request, handler)
				logger := log.C(request.Context())
				correlationId := logger.Data[log.FieldCorrelationID].(string)
				Expect(correlationId).To(Equal(expectedCorrelationId))
			})
		})
	})
})
