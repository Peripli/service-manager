package schemas

const ReferencePlan = `{
  "metadata": {
    "supportedPlatforms": [],
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
