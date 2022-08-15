package schemas

import (
	"encoding/json"
	"fmt"
	"github.com/gofrs/uuid"
	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/instance_sharing"
	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/types"
	"time"
)

func BuildReferencePlanSchema() string {
	return fmt.Sprintf(`{
  "ready": true,
  "name": "%[1]s",
  "catalog_name": "%[1]s",
  "description": "%[2]s",
  "free": true,
  "bindable": true,
  "plan_updateable": false,
  "binding_rotatable": true,
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
          "_controlsOrder": [
            "%[3]s",
            "%[6]s"
          ],
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
              "type": "object",
              "_controlsOrder": [
                "%[9]s",
                "%[12]s",
                "%[15]s"
              ],
              "properties": {
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
                },
                "%[15]s": {
                  "title": "%[16]s",
                  "description": "%[17]s",
                  "type": "array",
                  "minItems": 0,
                  "items": {
                    "type": "string"
                  }
                }
              }
            }
          }
        }
      },
      "update": {
        "parameters": {
          "$schema": "http://json-schema.org/draft-04/schema#",
          "type": "object",
          "_show_form_view": false,
          "_controlsOrder": [],
          "additionalProperties": false,
          "properties": {}
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
		instance_sharing.SelectorsKey,                             // 6
		instance_sharing.BySelectorsTitle,                         // 7
		instance_sharing.BySelectorsDescription,                   // 8
		instance_sharing.ReferenceInstanceNameSelectorKey,         // 9
		instance_sharing.ReferenceInstanceNameSelectorTitle,       // 10
		instance_sharing.ReferenceInstanceNameSelectorDescription, // 11
		instance_sharing.ReferencePlanNameSelectorKey,             // 12
		instance_sharing.ReferencePlanNameSelectorTitle,           // 13
		instance_sharing.ReferencePlanNameSelectorDescription,     // 14
		instance_sharing.ReferenceLabelSelectorKey,                // 15
		instance_sharing.ReferenceLabelSelectorTitle,              // 16
		instance_sharing.ReferenceLabelSelectorDescription,        // 17
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
