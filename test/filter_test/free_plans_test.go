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

package filter_test

import (
	"net/http"

	"github.com/mitchellh/mapstructure"

	"github.com/Peripli/service-manager/pkg/types"

	"github.com/tidwall/gjson"

	"github.com/tidwall/sjson"

	"github.com/Peripli/service-manager/test/common"

	. "github.com/onsi/ginkgo"

	. "github.com/onsi/gomega"
)

const testCatalog = `
{
  "services": [{
    "name": "fake-service",
    "id": "acb56d7c-XXXX-XXXX-XXXX-feb140a59a66",
    "description": "A fake service.",
    "tags": ["no-sql", "relational"],
    "requires": ["route_forwarding"],
    "bindable": true,
    "instances_retrievable": true,
    "bindings_retrievable": true,
    "metadata": {
      "provider": {
        "name": "The name"
      },
      "listing": {
        "imageUrl": "http://example.com/cat.gif",
        "blurb": "Add a blurb here",
        "longDescription": "A long time ago, in a galaxy far far away..."
      },
      "displayName": "The Fake Service Broker"
    },
    "plan_updateable": true,
    "plans": [{
      "name": "paid-plan",
      "id": "11131751-XXXX-XXXX-XXXX-a42377d3320e",
      "description": "Shared fake Server, 5tb persistent disk, 40 max concurrent connections.",
      "free": false,
      "metadata": {
        "max_storage_tb": 5,
        "costs":[
            {
               "amount":{
                  "usd":99.0
               },
               "unit":"MONTHLY"
            },
            {
               "amount":{
                  "usd":0.99
               },
               "unit":"1GB of messages over 20GB"
            }
         ],
        "bullets": [
          "Shared fake server",
          "5 TB storage",
          "40 concurrent connections"
        ]
      },
      "schemas": {
        "service_instance": {
          "create": {
            "parameters": {
              "$schema": "http://json-schema.org/draft-04/schema#",
              "type": "object",
              "properties": {
                "billing-account": {
                  "description": "Billing account number used to charge use of shared fake server.",
                  "type": "string"
                }
              }
            }
          },
          "update": {
            "parameters": {
              "$schema": "http://json-schema.org/draft-04/schema#",
              "type": "object",
              "properties": {
                "billing-account": {
                  "description": "Billing account number used to charge use of shared fake server.",
                  "type": "string"
                }
              }
            }
          }
        },
        "service_binding": {
          "create": {
            "parameters": {
              "$schema": "http://json-schema.org/draft-04/schema#",
              "type": "object",
              "properties": {
                "billing-account": {
                  "description": "Billing account number used to charge use of shared fake server.",
                  "type": "string"
                }
              }
            }
          }
        }
      }
    }, 
	{
      "name": "free-plan",
      "id": "1c4008b5-XXXX-XXXX-XXXX-dace631cd648",
      "description": "Shared fake Server, 5tb persistent disk, 40 max concurrent connections. 100 async.",
      "free": true,
      "metadata": {
        "max_storage_tb": 5,
        "costs":[
            {
               "amount":{
                  "usd":199.0
               },
               "unit":"MONTHLY"
            },
            {
               "amount":{
                  "usd":0.99
               },
               "unit":"1GB of messages over 20GB"
            }
         ],
        "bullets": [
          "40 concurrent connections"
        ]
      }
    }]
  }]
}
`

const newFreePlan = `
	{
      "name": "new-free-plan",
      "id": "456008b5-XXXX-XXXX-XXXX-dace631cd648",
      "description": "Shared fake Server, 5tb persistent disk, 40 max concurrent connections. 100 async.",
      "free": true,
      "metadata": {
        "max_storage_tb": 5,
        "costs":[
            {
               "amount":{
                  "usd":199.0
               },
               "unit":"MONTHLY"
            },
            {
               "amount":{
                  "usd":0.99
               },
               "unit":"1GB of messages over 20GB"
            }
         ],
        "bullets": [
          "40 concurrent connections"
        ]
      }
    }
`

const newPaidPlan = `
	{
      "name": "new-paid-plan",
      "id": "789008b5-XXXX-XXXX-XXXX-dace631cd648",
      "description": "Shared fake Server, 5tb persistent disk, 40 max concurrent connections. 100 async.",
      "free": false,
      "metadata": {
        "max_storage_tb": 5,
        "costs":[
            {
               "amount":{
                  "usd":199.0
               },
               "unit":"MONTHLY"
            },
            {
               "amount":{
                  "usd":0.99
               },
               "unit":"1GB of messages over 20GB"
            }
         ],
        "bullets": [
          "40 concurrent connections"
        ]
      }
    }
`

var _ = Describe("Service Manager Free Plans Filter", func() {
	var ctx *common.TestContext
	var existingBrokerID string
	var existingBrokerServer *common.BrokerServer

	var oldFreePlanCatalogID string
	var oldFreePlanCatalogName string

	var oldPaidPlanCatalogID string
	var oldPaidPlanCatalogName string

	var newFreePlanCatalogID string
	var newFreePlanCatalogName string

	var newPaidPlanCatalogID string
	var newPaidPlanCatalogName string

	var serviceCatalogID string
	var serviceCatalogName string

	findVisibilityForServicePlanID := func(servicePlanID string) []*types.Visibility {
		result := make([]*types.Visibility, 0, 0)
		visibilities := ctx.SMWithOAuth.GET("/v1/visibilities").
			Expect().
			Status(http.StatusOK).JSON().Object().Value("visibilities").Array().Iter()

		for _, visibility := range visibilities {
			if visibility.Object().Value("service_plan_id").String().Raw() == servicePlanID {
				v := &types.Visibility{}
				err := mapstructure.Decode(visibility.Object().Raw(), v)
				Expect(err).ToNot(HaveOccurred())
				result = append(result, v)
			}
		}
		return result
	}

	BeforeSuite(func() {
		ctx = common.NewTestContext(nil)
	})

	AfterSuite(func() {
		ctx.Cleanup()
	})

	BeforeEach(func() {
		ctx.SMWithOAuth.GET("/v1/service_plans").
			Expect().
			Status(http.StatusOK).JSON().Path("$.service_plans[*].catalog_id").Array().NotContains(newFreePlanCatalogID, newPaidPlanCatalogID)

		existingBrokerID, existingBrokerServer = ctx.RegisterBrokerWithCatalog(testCatalog)
		Expect(existingBrokerID).ToNot(BeEmpty())

		serviceCatalogID = gjson.Get(testCatalog, "services.0.id").Str
		Expect(serviceCatalogID).ToNot(BeEmpty())

		serviceCatalogName = gjson.Get(testCatalog, "services.0.name").Str
		Expect(serviceCatalogName).ToNot(BeEmpty())

		oldFreePlanCatalogID = gjson.Get(testCatalog, "services.0.plans.1.id").Str
		Expect(oldFreePlanCatalogID).ToNot(BeEmpty())

		oldFreePlanCatalogName = gjson.Get(testCatalog, "services.0.plans.1.name").Str
		Expect(oldFreePlanCatalogName).ToNot(BeEmpty())

		oldPaidPlanCatalogID = gjson.Get(testCatalog, "services.0.plans.0.id").Str
		Expect(oldPaidPlanCatalogID).ToNot(BeEmpty())

		oldPaidPlanCatalogName = gjson.Get(testCatalog, "services.0.plans.0.name").Str
		Expect(oldPaidPlanCatalogName).ToNot(BeEmpty())

		newFreePlanCatalogID = gjson.Get(newFreePlan, "id").Str
		Expect(newFreePlanCatalogID).ToNot(BeEmpty())

		newFreePlanCatalogName = gjson.Get(newFreePlan, "name").Str
		Expect(newFreePlanCatalogName).ToNot(BeEmpty())

		newPaidPlanCatalogID = gjson.Get(newPaidPlan, "id").Str
		Expect(newPaidPlanCatalogID).ToNot(BeEmpty())

		newPaidPlanCatalogName = gjson.Get(newPaidPlan, "name").Str
		Expect(newPaidPlanCatalogName).ToNot(BeEmpty())

		existingBrokerServer.Reset()
	})

	AfterEach(func() {
		ctx.CleanupBroker(existingBrokerID)
	})

	Specify("plans and visibilities for the registered brokers are known to SM", func() {
		freePlanID := ctx.SMWithOAuth.GET("/v1/service_plans").WithQuery("catalog_name", oldFreePlanCatalogName).
			Expect().
			Status(http.StatusOK).JSON().Object().Value("service_plans").Array().First().Object().Value("id").String().Raw()

		Expect(freePlanID).ToNot(BeEmpty())

		visibilities := findVisibilityForServicePlanID(freePlanID)
		Expect(len(visibilities)).To(Equal(1))
		Expect(visibilities[0].PlatformID).To(Equal(""))

		paidPlanID := ctx.SMWithOAuth.GET("/v1/service_plans").WithQuery("catalog_name", oldPaidPlanCatalogName).
			Expect().
			Status(http.StatusOK).JSON().Object().Value("service_plans").Array().First().Object().Value("id").String().Raw()

		Expect(paidPlanID).ToNot(BeEmpty())

		visibilities = findVisibilityForServicePlanID(paidPlanID)
		Expect(len(visibilities)).To(Equal(0))
	})

	Context("when no modifications to the plans occurs", func() {
		It("does not change the state of the visibilities for the existing plans", func() {

		})
	})

	Context("when a new free plan is added", func() {
		BeforeEach(func() {
			s, err := sjson.Set(testCatalog, "services.0.plans.-1", common.JSONToMap(newFreePlan))
			Expect(err).ShouldNot(HaveOccurred())
			existingBrokerServer.Catalog = common.JSONToMap(s)
		})

		It("creates the plan and creates a public visibility for it", func() {
			ctx.SMWithOAuth.GET("/v1/service_plans").
				Expect().
				Status(http.StatusOK).JSON().Path("$.service_plans[*].catalog_id").Array().NotContains(newFreePlanCatalogID)

			ctx.SMWithOAuth.PATCH("/v1/service_brokers/" + existingBrokerID).
				WithJSON(common.Object{}).
				Expect().
				Status(http.StatusOK)

			planID := ctx.SMWithOAuth.GET("/v1/service_plans").WithQuery("catalog_name", newFreePlanCatalogName).
				Expect().
				Status(http.StatusOK).JSON().Object().Value("service_plans").Array().First().Object().Value("id").String().Raw()

			Expect(planID).ToNot(BeEmpty())

			visibilities := findVisibilityForServicePlanID(planID)
			Expect(len(visibilities)).To(Equal(1))
			Expect(visibilities[0].PlatformID).To(Equal(""))
		})
	})

	Context("when a new paid plan is added", func() {
		BeforeEach(func() {
			s, err := sjson.Set(testCatalog, "services.0.plans.-1", common.JSONToMap(newPaidPlan))
			Expect(err).ShouldNot(HaveOccurred())
			existingBrokerServer.Catalog = common.JSONToMap(s)
		})

		It("creates the plan and does not create a new public visibility for it", func() {
			ctx.SMWithOAuth.GET("/v1/service_plans").
				Expect().
				Status(http.StatusOK).JSON().Path("$.service_plans[*].catalog_id").Array().NotContains(newPaidPlanCatalogID)

			ctx.SMWithOAuth.PATCH("/v1/service_brokers/" + existingBrokerID).
				WithJSON(common.Object{}).
				Expect().
				Status(http.StatusOK)

			planID := ctx.SMWithOAuth.GET("/v1/service_plans").WithQuery("catalog_name", newPaidPlanCatalogName).
				Expect().
				Status(http.StatusOK).JSON().Object().Value("service_plans").Array().First().Object().Value("id").String().Raw()

			Expect(planID).ToNot(BeEmpty())

			visibilities := findVisibilityForServicePlanID(planID)
			Expect(len(visibilities)).To(Equal(0))
		})
	})

	Context("when an existing free plan is made paid", func() {
		BeforeEach(func() {
			tempCatalog, err := sjson.Set(testCatalog, "services.0.plans.0.free", false)
			Expect(err).ToNot(HaveOccurred())

			catalog, err := sjson.Set(tempCatalog, "services.0.plans.1.free", false)
			Expect(err).ToNot(HaveOccurred())

			existingBrokerServer.Catalog = common.JSONToMap(catalog)
		})

		It("deletes the public visibility associated with the plan", func() {
			plan := ctx.SMWithOAuth.GET("/v1/service_plans").WithQuery("catalog_name", oldFreePlanCatalogName).
				Expect().
				Status(http.StatusOK).JSON()

			plan.Path("$.service_plans[*].free").Array().Contains(true)
			plan.Object().Value("service_plans").Array().Length().Equal(1)
			planID := plan.Object().Value("service_plans").Array().First().Object().Value("id").String().Raw()
			Expect(planID).ToNot(BeEmpty())

			visibilities := findVisibilityForServicePlanID(planID)
			Expect(len(visibilities)).To(Equal(1))
			Expect(visibilities[0].PlatformID).To(Equal(""))

			ctx.SMWithOAuth.PATCH("/v1/service_brokers/" + existingBrokerID).
				WithJSON(common.Object{}).
				Expect().
				Status(http.StatusOK)

			visibilities = findVisibilityForServicePlanID(planID)
			Expect(len(visibilities)).To(Equal(0))
		})
	})

	Context("when an existing paid plan is made free", func() {
		BeforeEach(func() {
			tempCatalog, err := sjson.Set(testCatalog, "services.0.plans.0.free", true)
			Expect(err).ToNot(HaveOccurred())

			catalog, err := sjson.Set(tempCatalog, "services.0.plans.1.free", true)
			Expect(err).ToNot(HaveOccurred())

			existingBrokerServer.Catalog = common.JSONToMap(catalog)
		})

		It("deletes all non-public visibilities that were associated with the plan", func() {

		})

		It("creates a public visibility associated with the plan", func() {
			plan := ctx.SMWithOAuth.GET("/v1/service_plans").WithQuery("catalog_name", oldPaidPlanCatalogName).
				Expect().
				Status(http.StatusOK).JSON()

			plan.Path("$.service_plans[*].free").Array().Contains(false)
			plan.Object().Value("service_plans").Array().Length().Equal(1)
			planID := plan.Object().Value("service_plans").Array().First().Object().Value("id").String().Raw()
			Expect(planID).ToNot(BeEmpty())

			visibilities := findVisibilityForServicePlanID(planID)
			Expect(len(visibilities)).To(Equal(0))

			ctx.SMWithOAuth.PATCH("/v1/service_brokers/" + existingBrokerID).
				WithJSON(common.Object{}).
				Expect().
				Status(http.StatusOK)

			visibilities = findVisibilityForServicePlanID(planID)
			Expect(len(visibilities)).To(Equal(1))
			Expect(visibilities[0].PlatformID).To(Equal(""))
		})
	})
})
