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

package common

import (
	"fmt"

	"github.com/tidwall/gjson"

	"github.com/gofrs/uuid"
	"github.com/tidwall/sjson"
)

var emptyCatalog = `
{
  "services": []
}
`

var testFreePlan = `
	{
      "name": "another-free-plan-name-%[1]s",
      "id": "%[1]s",
      "description": "test-description",
	  "free": true,
	  "bindable": true,
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

var testShareableFreePlan = `
	{
      "name": "another-free-plan-name-%[1]s",
      "id": "%[1]s",
      "description": "test-description",
	  "free": true,
	  "bindable": true,
      "metadata": {
		"supportsInstanceSharing": true,
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

var testShareablePaidPlan = `
	{
      "name": "shareable-plan-name-%[1]s",
      "id": "%[1]s",
      "description": "test-description",
	  "free": false,
	  "bindable": true,
      "metadata": {
        "max_storage_tb": 5,
		"supportedPlatforms": [],
		"supportsInstanceSharing": true,
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

var testShareableNonBindablePlan = `
	{
      "name": "shareable-plan-name-%[1]s",
      "id": "%[1]s",
      "description": "test-description",
	  "free": false,
	  "bindable": false,
      "metadata": {
        "max_storage_tb": 5,
		"supportedPlatforms": [],
		"supportsInstanceSharing": true,
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

var testPaidPlan = `
	{
      "name": "another-paid-plan-name-%[1]s",
      "id": "%[1]s",
      "description": "test-description",
      "free": false,
      "bindable": true,
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

var testService = `
{
    "name": "another-fake-service-%[1]s",
    "id": "%[1]s",
    "description": "test-description",
    "requires": ["another-route_forwarding"],
    "tags": ["another-no-sql", "another-relational"],
    "bindable": true,	
    "instances_retrievable": true,	
    "allow_context_updates": true,	
    "bindings_retrievable": true,	
    "metadata": {	
      "provider": {	
        "name": "another name"	
      },	
      "listing": {	
        "imageUrl": "http://example.com/cat.gif",	
        "blurb": "another blurb here",	
        "longDescription": "A long time ago, in a another galaxy far far away..."	
      },	
      "displayName": "another Fake Service Broker"	
    },	
    "plan_updateable": true,	
    "plans": []
}
`

var testServiceNonBindable = `
{
    "name": "another-fake-service-%[1]s",
    "id": "%[1]s",
    "description": "test-description",
    "requires": ["another-route_forwarding"],
    "tags": ["another-no-sql", "another-relational"],
    "bindable": false,	
    "instances_retrievable": true,	
    "allow_context_updates": true,	
    "bindings_retrievable": true,	
    "metadata": {	
      "provider": {	
        "name": "another name"	
      },	
      "listing": {	
        "imageUrl": "http://example.com/cat.gif",	
        "blurb": "another blurb here",	
        "longDescription": "A long time ago, in a another galaxy far far away..."	
      },	
      "displayName": "another Fake Service Broker"	
    },	
    "plan_updateable": true,	
    "plans": []
}
`

type SBCatalog string

func (sbc *SBCatalog) AddService(service string) {
	s, err := sjson.Set(string(*sbc), "services.-1", JSONToMap(service))
	if err != nil {
		panic(err)
	}

	*sbc = SBCatalog(s)
}

func (sbc *SBCatalog) AddPlanToService(plan string, serviceIndex int) {
	s, err := sjson.Set(string(*sbc), fmt.Sprintf("services.%d.plans.-1", serviceIndex), JSONToMap(plan))
	if err != nil {
		panic(err)
	}

	*sbc = SBCatalog(s)
}

func (sbc *SBCatalog) RemoveService(index int) (string, string) {
	service := gjson.Get(string(*sbc), fmt.Sprintf("services.%d", index)).Raw
	catalogID := gjson.Get(string(*sbc), "id").Raw
	s, err := sjson.Delete(string(*sbc), fmt.Sprintf("services.%d", index))
	if err != nil {
		panic(err)
	}
	*sbc = SBCatalog(s)

	return catalogID, service
}

func (sbc *SBCatalog) RemovePlan(serviceIndex, planIndex int) (string, string) {
	plan := gjson.Get(string(*sbc), fmt.Sprintf("services.%d.plans.%d", serviceIndex, planIndex)).Raw
	id := gjson.Get(string(*sbc), "id").Raw
	s, err := sjson.Delete(string(*sbc), fmt.Sprintf("services.%d.plans.%d", serviceIndex, planIndex))
	if err != nil {
		panic(err)
	}
	*sbc = SBCatalog(s)

	return id, plan
}

// NewRandomSBCatalog returns a service broker catalog containg one random service with one free and one paid random plans
func NewRandomSBCatalog() SBCatalog {
	plan1 := GeneratePaidTestPlan()
	plan2 := GenerateFreeTestPlan()
	plan3 := GenerateFreeTestPlan()
	plan4 := GenerateFreeTestPlan()
	plan5 := GenerateFreeTestPlan()
	var err error
	plan4, err = sjson.Set(plan4, "bindable", false)
	if err != nil {
		panic(err)
	}

	service1 := GenerateTestServiceWithPlans(plan1, plan2, plan3, plan4)
	service2 := GenerateTestServiceWithPlans(plan5)
	service2, err = sjson.Set(service2, "bindings_retrievable", false)
	if err != nil {
		panic(err)
	}

	catalog := NewEmptySBCatalog()
	catalog.AddService(service1)
	catalog.AddService(service2)

	return catalog
}

// NewRandomSBCatalog returns a service broker catalog containg one random service with one free and one paid random plans
func NewShareableCatalog() (SBCatalog, string, string, string, string, string) {
	plan1, _ := GenerateShareablePaidTestPlan()
	plan2, _ := GenerateShareablePaidTestPlan()
	plan3, _ := GenerateShareablePaidTestPlan()

	service1 := GenerateTestServiceWithPlans(plan1, plan2)
	service2 := GenerateTestServiceWithPlans(plan3)

	catalog := NewEmptySBCatalog()
	catalog.AddService(service1)
	catalog.AddService(service2)

	return catalog, service1, service2, plan1, plan2, plan3
}

// NewEmptySBCatalog returns an empty service broker catalog tha contains no services and no plans
func NewEmptySBCatalog() SBCatalog {
	catalog := SBCatalog(emptyCatalog)
	return catalog
}

func GenerateTestServiceWithPlansWithID(serviceID string, plans ...string) string {
	var err error
	catalogService := fmt.Sprintf(testService, serviceID)
	for _, plan := range plans {
		catalogService, err = sjson.Set(catalogService, "plans.-1", JSONToMap(plan))
		if err != nil {
			panic(err)
		}
	}

	catalogID := gjson.Get(catalogService, "id").Str
	if catalogID == "" {
		panic("catalog_id cannot be empty")
	}
	catalogName := gjson.Get(catalogService, "name").Str
	if catalogName == "" {
		panic("catalog_name cannot be empty")
	}

	return catalogService
}

func GenerateTestServiceWithPlansWithIDNonBindable(serviceID string, plans ...string) string {
	var err error
	catalogService := fmt.Sprintf(testServiceNonBindable, serviceID)
	for _, plan := range plans {
		catalogService, err = sjson.Set(catalogService, "plans.-1", JSONToMap(plan))
		if err != nil {
			panic(err)
		}
	}

	catalogID := gjson.Get(catalogService, "id").Str
	if catalogID == "" {
		panic("catalog_id cannot be empty")
	}
	catalogName := gjson.Get(catalogService, "name").Str
	if catalogName == "" {
		panic("catalog_name cannot be empty")
	}

	return catalogService
}

func GenerateTestServiceWithPlans(plans ...string) string {
	UUID, err := uuid.NewV4()
	if err != nil {
		panic(err)
	}

	return GenerateTestServiceWithPlansWithID(UUID.String(), plans...)

}
func GenerateTestServiceWithPlansNonBindable(plans ...string) string {
	UUID, err := uuid.NewV4()
	if err != nil {
		panic(err)
	}

	return GenerateTestServiceWithPlansWithIDNonBindable(UUID.String(), plans...)

}

func GenerateTestPlanWithID(planID string) string {
	return GenerateTestPlanFromTemplate(planID, testPaidPlan)
}
func GenerateShareableTestPlanWithID(planID string) string {
	return GenerateTestPlanFromTemplate(planID, testShareablePaidPlan)
}

func GenerateTestPlan() string {
	UUID, err := uuid.NewV4()
	if err != nil {
		panic(err)
	}
	id := UUID.String()
	return GenerateTestPlanWithID(id)
}

func GenerateFreeTestPlan() string {
	UUID, err := uuid.NewV4()
	if err != nil {
		panic(err)
	}
	return GenerateTestPlanFromTemplate(UUID.String(), testFreePlan)
}
func GenerateShareableFreeTestPlan() string {
	UUID, err := uuid.NewV4()
	if err != nil {
		panic(err)
	}
	return GenerateTestPlanFromTemplate(UUID.String(), testShareableFreePlan)
}
func GenerateShareablePaidTestPlan() (string, string) {
	UUID, err := uuid.NewV4()
	if err != nil {
		panic(err)
	}
	return GenerateTestPlanFromTemplate(UUID.String(), testShareablePaidPlan), UUID.String()
}

func GenerateShareableNonBindablePlan() string {
	UUID, err := uuid.NewV4()
	if err != nil {
		panic(err)
	}
	return GenerateTestPlanFromTemplate(UUID.String(), testShareableNonBindablePlan)
}

func GeneratePaidTestPlan() string {
	UUID, err := uuid.NewV4()
	if err != nil {
		panic(err)
	}
	return GenerateTestPlanFromTemplate(UUID.String(), testPaidPlan)
}

func GenerateTestPlanFromTemplate(id, planTemplate string) string {
	if len(id) == 0 {
		UUID, err := uuid.NewV4()
		if err != nil {
			panic(err)
		}
		id = UUID.String()
	}
	return fmt.Sprintf(planTemplate, id)
}
