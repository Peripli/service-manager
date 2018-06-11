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

package info

import (
	"errors"
	"net/http"
	"testing"

	"github.com/Peripli/service-manager/rest"
	"github.com/Peripli/service-manager/server/serverfakes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func Test(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Info Suite")
}

var _ = Describe("Info API", func() {

	environment := &serverfakes.FakeEnvironment{}
	var constructNewController func()
	var controller rest.Controller

	configureEnvironmentUnmarshalInfo := func() {
		response := Details{TokenIssuer: "https://example.com"}
		environment.UnmarshalStub = func(value interface{}) error {
			val, ok := value.(*Details)
			if ok {
				*val = response
			}
			return nil
		}
	}

	BeforeEach(func() {
		constructNewController = func() {
			controller = NewController(environment)
		}
	})

	Describe("Construct Info Controller", func() {
		Context("When unmarshal", func() {
			Context("Returns an error", func() {
				BeforeEach(func() {
					environment.UnmarshalReturns(errors.New("error"))
				})

				It("Should panic", func() {
					Expect(constructNewController).To(Panic())
				})
			})
			Context("Doesn't return an error", func() {
				BeforeEach(func() {
					environment.UnmarshalReturns(nil)
				})
				Context("And TokenIssuer is empty", func() {
					It("Should panic", func() {
						Expect(constructNewController).To(Panic())
					})
				})
				Context("And TokenIssuer is not empty", func() {
					BeforeEach(func() {
						configureEnvironmentUnmarshalInfo()
					})
					It("Should be OK", func() {
						Expect(constructNewController).To(Not(Panic()))
					})
				})
			})
		})
	})

	Describe("Routes", func() {
		BeforeEach(func() {
			configureEnvironmentUnmarshalInfo()
		})
		It("Returns one route", func() {
			routes := controller.Routes()
			Expect(len(routes)).To(Equal(1))

			route := routes[0]
			Expect(route.Endpoint.Path).To(Equal(URL))
			Expect(route.Endpoint.Method).To(Equal(http.MethodGet))
		})
	})
})
