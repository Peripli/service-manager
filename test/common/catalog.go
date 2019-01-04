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
func NewRandomSBCatalog() *SBCatalog {
	_, plan1, _ := GenerateFreeTestPlan()
	_, plan2, _ := GeneratePaidTestPlan()
	_, service1, _ := GenerateTestServiceWithPlans(plan1, plan2)

	catalog := NewEmptySBCatalog()
	catalog.AddService(service1)

	return catalog
}

// NewEmptySBCatalog returns an empty service broker catalog tha contains no services and no plans
func NewEmptySBCatalog() *SBCatalog {
	catalog := SBCatalog(emptyCatalog)
	return &catalog
}

func GenerateTestServiceWithPlans(plans ...string) (string, string, string) {
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

	s1, err := sjson.Set(catalogService, "catalog_id", catalogID)
	if err != nil {
		panic(err)
	}

	s2, err := sjson.Set(s1, "catalog_name", catalogName)
	if err != nil {
		panic(err)
	}

	smService, err := sjson.Delete(s2, "id")
	if err != nil {
		panic(err)
	}

	return catalogID, catalogService, smService
}

func GenerateFreeTestPlan() (string, string, string) {
	return GenerateTestPlan(testFreePlan)
}

func GeneratePaidTestPlan() (string, string, string) {
	return GenerateTestPlan(testPaidPlan)
}

func GenerateTestPlan(planTemplate string) (string, string, string) {
	UUID, err := uuid.NewV4()
	if err != nil {
		panic(err)
	}

	catalogPlan := fmt.Sprintf(planTemplate, UUID.String())

	catalogID := gjson.Get(catalogPlan, "id").Str
	if catalogID == "" {
		panic("catalog_id cannot be empty")
	}
	catalogName := gjson.Get(catalogPlan, "name").Str
	if catalogName == "" {
		panic("catalog_name cannot be empty")
	}
	p1, err := sjson.Set(catalogPlan, "catalog_id", catalogID)
	if err != nil {
		panic(err)
	}

	p2, err := sjson.Set(p1, "catalog_name", catalogName)
	if err != nil {
		panic(err)
	}

	smPlan, err := sjson.Delete(p2, "id")
	if err != nil {
		panic(err)
	}
	return catalogID, catalogPlan, smPlan
}
