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

var testPaidPlan = `
	{
      "name": "another-paid-plan-name-%[1]s",
      "id": "%[1]s",
      "description": "test-description",
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

var testService = `
{
    "name": "another-fake-service-%[1]s",
    "id": "%[1]s",
    "description": "test-description",
    "requires": ["another-route_forwarding"],
    "tags": ["another-no-sql", "another-relational"],
    "bindable": true,	
    "instances_retrievable": true,	
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
	s, err := sjson.Set(string(*sbc), fmt.Sprintf("services.%d.-1", serviceIndex), JSONToMap(plan))
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
	service1 := GenerateTestServiceWithPlans(plan1, plan2)

	catalog := NewEmptySBCatalog()
	catalog.AddService(service1)

	return catalog
}

// NewEmptySBCatalog returns an empty service broker catalog tha contains no services and no plans
func NewEmptySBCatalog() SBCatalog {
	catalog := SBCatalog(emptyCatalog)
	return catalog
}

func GenerateTestServiceWithPlans(plans ...string) string {
	UUID, err := uuid.NewV4()
	if err != nil {
		panic(err)
	}

	catalogService := fmt.Sprintf(testService, UUID.String())
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

func GenerateTestPlan() string {
	return GenerateTestPlanFromTemplate(testPaidPlan)
}

func GenerateFreeTestPlan() string {
	return GenerateTestPlanFromTemplate(testFreePlan)
}

func GeneratePaidTestPlan() string {
	return GenerateTestPlanFromTemplate(testPaidPlan)
}

func GenerateTestPlanFromTemplate(planTemplate string) string {
	UUID, err := uuid.NewV4()
	if err != nil {
		panic(err)
	}
	return fmt.Sprintf(planTemplate, UUID.String())
}
