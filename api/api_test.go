/*
 * Copyright 2018 The Service Manager Authors
 *
 *    Licensed under the Apache License, Version oidc_authn.0 (the "License");
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

	"context"

	"github.com/Peripli/service-manager/pkg/web"
	"github.com/Peripli/service-manager/storage/storagefakes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestApi(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "API Suite")
}

type testController struct {
}

func (c *testController) Routes() []web.Route {
	return []web.Route{}
}

var _ = Describe("API", func() {

	var (
		mockedStorage *storagefakes.FakeStorage
		settings      Settings
		api           *web.API
		err           error
	)

	BeforeEach(func() {
		mockedStorage = &storagefakes.FakeStorage{}
		settings = Settings{
			TokenIssuerURL: "http://example.com",
		}

		api, err = New(context.TODO(), mockedStorage, settings)
		Expect(err).ShouldNot(HaveOccurred())
	})

	Describe("Controller Registration", func() {
		Context("With nil controllers slice", func() {
			It("Should panic", func() {
				nilControllersSlice := func() {
					api.RegisterControllers(nil)
				}
				Expect(nilControllersSlice).To(Panic())
			})
		})

		Context("With nil controller in slice", func() {
			It("Should panic", func() {
				nilControllerInSlice := func() {
					var controllers []web.Controller
					controllers = append(controllers, &testController{})
					controllers = append(controllers, nil)
					api.RegisterControllers(controllers...)
				}
				Expect(nilControllerInSlice).To(Panic())
			})
		})

		Context("With no brokers registered", func() {
			It("Should increase broker count", func() {
				originalControllersCount := len(api.Controllers)
				api.RegisterControllers(&testController{})
				Expect(len(api.Controllers)).To(Equal(originalControllersCount + 1))
			})
		})
	})
})
