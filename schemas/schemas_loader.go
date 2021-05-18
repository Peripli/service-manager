package schemas

import (
	"encoding/json"
	"io/ioutil"
	"os"
)



func SchemasLoader(schemaName string) (json.RawMessage, error) {
	path, _ := os.Getwd()
	schemasPath := path + "/" + schemaName
	schema, err := ioutil.ReadFile(schemasPath)
	if err != nil {
		return nil, err
	}
	return schema, nil

}
