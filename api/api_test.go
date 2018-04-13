/*
 *    Copyright 2018 The Service Manager Authors
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

package api

import (
	"testing"

	"github.com/Peripli/service-manager/rest"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestApi(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "API Suite")
}

type testController struct {
}

func (c *testController) Routes() []rest.Route {
	return []rest.Route{}
}

var _ = Describe("API", func() {
	var defaultAPI rest.API

	BeforeEach(func() {
		defaultAPI = Default()
	})

	Describe("Controller Registration", func() {

		Context("With nil controller", func() {
			It("Should panic", func() {
				nilControllersSlice := func() {
					defaultAPI.RegisterControllers(nil)
				}
				Expect(nilControllersSlice).To(Panic())
			})
		})

		Context("With nil controller in slice", func() {
			It("Should panic", func() {
				nilControllerInSlice := func() {
					var controllers []rest.Controller
					controllers = append(controllers, &testController{})
					controllers = append(controllers, nil)
					defaultAPI.RegisterControllers(controllers...)
				}
				Expect(nilControllerInSlice).To(Panic())
			})
		})

		Context("With no brokers registered", func() {
			It("Should have only one broker", func() {
				defaultAPI.RegisterControllers(&testController{})
				Expect(len(defaultAPI.Controllers())).To(Equal(3))
			})
		})
	})
})
