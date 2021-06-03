package schemas

import (
	"encoding/json"
	"fmt"
)

func SchemaLoader(schema string) (json.RawMessage, json.RawMessage, error) {
	var planSchema map[string]json.RawMessage
	err := json.Unmarshal([]byte(schema), &planSchema)
	if err != nil {
		return nil, nil, fmt.Errorf("error setting reference schema for the plan: %s", err)
	}
	return planSchema["schemas"], planSchema["metadata"], nil

}
