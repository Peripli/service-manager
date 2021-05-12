package schemas

import (
	"encoding/json"
	"io/ioutil"
)

const schemasPath string = "./schemas"

func SchemasLoader(schemaName string) (json.RawMessage, error) {

	path := schemasPath + "/" + schemaName
	schema, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return schema, nil

}
