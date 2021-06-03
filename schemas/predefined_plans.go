package schemas

import (
	"encoding/json"
	"fmt"
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/gofrs/uuid"
	"time"
)

const ReferencePlan = `{
 "name": "reference-instance",
  "catalog_name": "reference-instance",
  "description": "Allows to create a reference to a shared service instance from any environment in a subaccount and manage service bindings to that service instance.",
  "bindable": true,
  "ready": true,
 "metadata": {
    "supportedPlatforms": [
      
    ],
    "translations": {
      "en-US": {
        "displayName": "reference-instance",
        "description": "Allows to create a reference to a shared service instance from any environment in a subaccount and manage service bindings to that service instance."
      }
    }
  },
  "schemas": {
    "service_instance": {
      "create": {
        "parameters": {
          "$schema": "http://json-schema.org/draft-04/schema#",
          "type": "object",
          "additionalProperties": false,
          "_show_form_view": true,
          "properties": {
            "referenced_instance_id": {
              "title": "Referenced Instance ID",
              "description": "Referenced instance ID is the instance_id of the shared instance from the other platform.",
              "_title": "TITLE_XTIT",
              "_description": "DESCRIPTION_XMSG",
              "type": "string",
              "minLength": 1,
              "maxLength": 100
            }
          }
        }
      }
    }
  }
}`

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
