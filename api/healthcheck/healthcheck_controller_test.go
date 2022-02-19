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
	"context"
	"fmt"
	"net/http"
	"testing"

	h "github.com/InVisionApp/go-health"
	"github.com/Peripli/service-manager/pkg/health"
	"github.com/Peripli/service-manager/pkg/web"

	. "github.com/onsi/ginkgo/v2"
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

		When("health returns up", func() {
			It("should respond with 200", func() {
				assertResponse(health.StatusUp, http.StatusOK)
			})
		})
	})

	Describe("aggregation", func() {
		var ctx context.Context
		var c *controller
		var healths map[string]h.State
		var thresholds map[string]int64

		BeforeEach(func() {
			ctx = context.TODO()
			healths = map[string]h.State{
				"test1":       {Status: "ok", Details: "details"},
				"test2":       {Status: "ok", Details: "details"},
				"failedState": {Status: "failed", Details: "details", Err: "err"},
			}
			thresholds = map[string]int64{
				"test1": 3,
				"test2": 3,
			}
			c = &controller{
				health:     HealthFake{},
				thresholds: thresholds,
			}
		})

		When("No healths are provided", func() {
			It("Returns UP", func() {
				aggregatedHealth := c.aggregate(ctx, nil)
				Expect(aggregatedHealth.Status).To(Equal(health.StatusUp))
			})
		})

		When("At least one health is DOWN more than threshold times and is Fatal", func() {
			It("Returns DOWN", func() {
				healths["test3"] = h.State{Status: "failed", Fatal: true, ContiguousFailures: 4}
				c.thresholds["test3"] = 3
				aggregatedHealth := c.aggregate(ctx, healths)
				Expect(aggregatedHealth.Status).To(Equal(health.StatusDown))
			})
		})

		When("At least one health is DOWN and is not Fatal", func() {
			It("Returns UP", func() {
				healths["test3"] = h.State{Status: "failed", Fatal: false, ContiguousFailures: 4}
				aggregatedHealth := c.aggregate(ctx, healths)
				Expect(aggregatedHealth.Status).To(Equal(health.StatusUp))
			})
		})

		When("There is DOWN healths but not more than threshold times in a row", func() {
			It("Returns UP", func() {
				healths["test3"] = h.State{Status: "failed"}
				c.thresholds["test3"] = 3
				aggregatedHealth := c.aggregate(ctx, healths)
				Expect(aggregatedHealth.Status).To(Equal(health.StatusUp))
			})
		})

		When("All healths are UP", func() {
			It("Returns UP", func() {
				aggregatedHealth := c.aggregate(ctx, healths)
				Expect(aggregatedHealth.Status).To(Equal(health.StatusUp))
			})
		})

		When("Aggregating health as unauthorized user", func() {
			It("should strip details and error", func() {
				aggregatedHealth := c.aggregate(ctx, healths)
				for name, h := range healths {
					h.Status = convertStatus(h.Status)
					h.Details = nil
					h.Err = ""
					Expect(aggregatedHealth.Details[name]).To(Equal(h))
				}
			})
		})

		When("Aggregating health as authorized user", func() {
			It("should include all details and errors", func() {
				ctx = web.ContextWithAuthorization(context.Background())
				aggregatedHealth := c.aggregate(ctx, healths)
				for name, h := range healths {
					h.Status = convertStatus(h.Status)
					Expect(aggregatedHealth.Details[name]).To(Equal(h))
				}
			})
		})
	})
})

func createController(status health.Status) *controller {
	stringStatus := "ok"
	var contiguousFailures int64 = 0
	if status == health.StatusDown {
		stringStatus = "failed"
		contiguousFailures = 1
	}

	return &controller{
		health: HealthFake{
			state: map[string]h.State{
				"test1": {Status: stringStatus, Fatal: true, ContiguousFailures: contiguousFailures},
			},
		},
		thresholds: map[string]int64{
			"test1": 1,
		},
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
