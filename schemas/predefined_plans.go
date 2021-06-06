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
	sr := fmt.Sprintf(`{
  "name": "%[1]s",
  "catalog_name": "%[1]s",
  "description": "%[2]s",
  "bindable": true,
  "ready": true,
  "metadata": {
    "supportedPlatforms": [],
    "translations": {
      "en-US": {
        "displayName": "%[1]s",
        "description": "%[2]s"
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
            "%[3]s": {
              "title": "Referenced Instance ID",
              "description": "%[2]s",
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
}`, instance_sharing.ReferencePlanName, instance_sharing.ReferencePlanDescription, instance_sharing.ReferencedInstanceIDKey)
	return sr
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
