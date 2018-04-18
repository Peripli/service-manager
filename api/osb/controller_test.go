/*
 * Copyright 2018 The Service Manager Authors
 *
 *    Licensed under the Apache License, Version 2.0 (the "License");
 *    you may not use this file except in compliance with the License.
 *    You may obtain a copy of the License at
 *
 *        http://www.apache.org/licenses/LICENSE-2.0
 *
 *    Unless required by applicable law or agreed to in writing, software
 *    distributed under the License is distributed on an "AS IS" BASIS,
 *    WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *    See the License for the specific language governing permissions and
 *    limitations under the License.
 */

package osb

import (
	"github.com/Peripli/service-manager/rest"
	"github.com/Peripli/service-manager/storage/storagefakes"
	"github.com/gorilla/mux"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Controller", func() {

	var (
		fakeBroker *storagefakes.FakeBroker
		controller *Controller
	)

	Describe("Routes", func() {
		BeforeEach(func() {
			fakeBroker = &storagefakes.FakeBroker{}
			controller = &Controller{
				BrokerStorage: fakeBroker,
			}

		})

		It("returns one valid OSB endpoint with router as handler", func() {
			routes := controller.Routes()
			Expect(len(routes)).To(Equal(1))

			route := routes[0]
			Expect(route.Handler).To(BeAssignableToTypeOf(mux.NewRouter()))
			Expect(route.Endpoint.Path).To(ContainSubstring("/osb"))
			Expect(route.Endpoint.Method).To(Equal(rest.AllMethods))
		})
	})
})
