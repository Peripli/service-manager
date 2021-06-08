package schemas

import (
	"encoding/json"
	"fmt"
	"github.com/Peripli/service-manager/pkg/instance_sharing"
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/gofrs/uuid"
	"time"
)

func BuildReferencePlanSchema() string {
	return fmt.Sprintf(`{
  "name": "%[1]s",
  "catalog_name": "%[1]s",
  "description": "%[2]s",
  "bindable": true,
  "ready": true,
  "metadata": {
    "supportedPlatforms": []
  },
  "schemas": {
    "service_instance": {
      "create": {
        "parameters": {
          "$schema": "http://json-schema.org/draft-04/schema#",
          "type": "object",
          "additionalProperties": false,
          "_show_form_view": true,
		  "_controlsOrder": ["%[3]s", "%[6]s", "%[9]s", "%[12]s"]
          "properties": {
            "%[3]s": {
              "title": "%[4]s",
              "description": "%[5]s",
              "type": "string",
              "minLength": 0,
              "maxLength": 100
            },
            "%[6]s": {
              "title": "%[7]s",
              "description": "%[8]s",
              "type": "string",
              "minLength": 0,
              "maxLength": 100
            },
            "%[9]s": {
              "title": "%[10]s",
              "description": "%[11]s",
              "type": "string",
              "minLength": 0,
              "maxLength": 100
            },
            "%[12]s": {
              "title": "%[13]s",
              "description": "%[14]s",
              "type": "string",
              "minLength": 0,
              "maxLength": 100
            }
          }
        }
      }
    }
  }
}`,
		instance_sharing.ReferencePlanName,                        // 1
		instance_sharing.ReferencePlanDescription,                 // 2
		instance_sharing.ReferencedInstanceIDKey,                  // 3
		instance_sharing.ReferencedInstanceIDTitle,                // 4
		instance_sharing.ReferencedInstanceIDDescription,          // 5
		instance_sharing.ReferenceInstanceNameSelectorKey,         // 6
		instance_sharing.ReferenceInstanceNameSelectorTitle,       // 7
		instance_sharing.ReferenceInstanceNameSelectorDescription, // 8
		instance_sharing.ReferencePlanNameSelectorKey,             // 9
		instance_sharing.ReferencePlanNameSelectorTitle,           // 10
		instance_sharing.ReferencePlanNameSelectorDescription,     // 11
		instance_sharing.ReferenceLabelSelectorKey,                // 12
		instance_sharing.ReferenceLabelSelectorTitle,              // 13
		instance_sharing.ReferenceLabelSelectorDescription,        // 14
	)
}

func CreatePlanOutOfSchema(schema string, serviceOfferingId string) (*types.ServicePlan, error) {
	var plan types.ServicePlan
	err := json.Unmarshal([]byte(schema), &plan)
	if err != nil {
		return &plan, fmt.Errorf("error creating plan from schema: %s", err)
	}
	UUID, err := uuid.NewV4()
	if err != nil {
		return nil, fmt.Errorf("could not generate GUID for ServicePlan: %s", err)
	}
	plan.ID = UUID.String()
	plan.CatalogID = UUID.String()
	plan.CreatedAt = time.Now()
	plan.UpdatedAt = time.Now()
	plan.ServiceOfferingID = serviceOfferingId
	return &plan, nil

}
