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

package info_test

import (
	"net/http"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/api/info"
	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/web"
)

func Test(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Info Suite")
}

var _ = Describe("Info API", func() {

	var controller web.Controller

	BeforeEach(func() {
		controller = &info.Controller{
			TokenIssuer: "https://example.com",
		}
	})

	Describe("Routes", func() {
		It("Returns one route", func() {
			routes := controller.Routes()
			Expect(len(routes)).To(Equal(1))

			route := routes[0]
			Expect(route.Endpoint.Path).To(Equal(info.URL))
			Expect(route.Endpoint.Method).To(Equal(http.MethodGet))
		})
	})
})
