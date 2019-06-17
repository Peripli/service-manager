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
package healthcheck

import (
	"fmt"
	h "github.com/InVisionApp/go-health"
	"github.com/Peripli/service-manager/pkg/health"
	"github.com/Peripli/service-manager/pkg/health/healthfakes"
	"net/http"
	"testing"

	"github.com/Peripli/service-manager/pkg/web"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestServer(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Healthcheck controller Suite")
}

var _ = Describe("Healthcheck controller", func() {

	statusText := func(status health.Status) string {
		return fmt.Sprintf(`status":"%s"`, status)
	}

	assertResponse := func(status health.Status, httpStatus int) {
		resp, err := createController(status).healthCheck(&web.Request{Request: &http.Request{}})
		Expect(err).ToNot(HaveOccurred())
		Expect(resp.StatusCode).To(Equal(httpStatus))
		Expect(string(resp.Body)).To(ContainSubstring(statusText(status)))
	}

	Describe("healthCheck", func() {
		When("health returns down", func() {
			It("should respond with 503", func() {
				assertResponse(health.StatusDown, http.StatusServiceUnavailable)
			})
		})

		When("health returns unknown", func() {
			It("should respond with 503", func() {
				assertResponse(health.StatusUnknown, http.StatusServiceUnavailable)
			})
		})

		When("health returns up", func() {
			It("should respond with 200", func() {
				assertResponse(health.StatusUp, http.StatusOK)
			})
		})
	})
})

func createController(status health.Status) *controller {
	aggPolicy := &healthfakes.FakeAggregationPolicy{}
	aggPolicy.ApplyReturns(&health.Health{Status: status})
	return &controller{
		health:           HealthFake{},
		aggPolicy:        aggPolicy,
		failuresTreshold: 3,
	}
}

type HealthFake struct {
	state  map[string]h.State
	failed bool
	err    error
}

func (hf HealthFake) AddChecks(cfgs []*h.Config) error {
	return nil
}

func (hf HealthFake) AddCheck(cfg *h.Config) error {
	return nil
}

func (hf HealthFake) Start() error {
	return nil
}

func (hf HealthFake) Stop() error {
	return nil
}

func (hf HealthFake) State() (map[string]h.State, bool, error) {
	return hf.state, hf.failed, hf.err
}
func (hf HealthFake) Failed() bool {
	return hf.failed
}
