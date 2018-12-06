/*
 *    Copyright 2018 The Service Manager Authors
 *
 *    Licensed under the Apache License, Version 2.0 (the "License");
 *    you may not use this file except in compliance with the License.
 *    You may obtain a copy of the License at
 *
 *        http://www.apache.org/licenses/LICENSE-2.0
 *
 *    Unless required by applicable law or agreed to in writing, software
 *    distributed under the License is distributed on an "AS IS" BASIS,
 *    WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *    See the License for the specific language governing permissions and
 *    limitations under the License.
 */

package main

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/Peripli/service-manager/pkg/web"

	"github.com/Peripli/service-manager/pkg/selection"
)

func main() {
	// visibilityTable db table for visibilities
	visibilityTable := "visibilities"

	// visibilityLabelsTable db table for visibilities table
	visibilityLabelsTable := "visibility_labels"

	// service_visibilities?fieldQuery=platform_id=iosjd0-sdfjw-r290ef-sojifds&labelQuery=subaccount+IN+[asd,qwe]
	request, err := http.NewRequest("GET", "http://localhost:8080/service_visibilities?fieldQuery=platform_id=iosjd0-sdfjw-r290ef-sojifds&labelQuery=subaccount+IN+[asd,qwe]", nil)
	if err != nil {
		panic(err)
	}
	req := &web.Request{
		Request: request,
	}
	criteria, err := selection.BuildQuerySegmentsForRequest(req)
	if err != nil {
		panic(err)
	}
	fmt.Println(criteria)
	options := []selection.Criteria{
		{
			LeftOp:   "subaccount",
			Operator: selection.InOperator,
			RightOp:  []string{"asd", "qwe"},
			Type:     "labelQuery",
		},
		{
			LeftOp:   "platform_id",
			Operator: selection.EqualsOperator,
			RightOp:  []string{"iosjd0-sdfjw-r290ef-sojifds"},
			Type:     selection.FieldQuery,
		},
	}
	baseQuery := fmt.Sprintf(`SELECT 
		%[1]s.*,
		%[2]s.id "%[2]s.id",
		%[2]s.key "%[2]s.key",
		%[2]s.val "%[2]s.val",
		%[2]s.created_at "%[2]s.created_at",
		%[2]s.updated_at "%[2]s.updated_at",
		%[2]s.visibility_id "%[2]s.visibility_id",
	FROM %[1]s 
	JOIN %[2]s ON %[1]s.id = %[2]s.visibility_id`, visibilityTable, visibilityLabelsTable)

	if len(options) > 0 {
		baseQuery += " WHERE "
	}

	hasInOperator := false
	var queries []string
	var queryParams []interface{}
	j := 0
	for _, option := range options {
		// handle operations that need
		preRightOperand := ""
		postRightOperand := ""
		if option.Operator.IsMultiVariate() {
			hasInOperator = true
			preRightOperand = "("
			postRightOperand = ")"
		}
		if option.Type == selection.LabelQuery {
			//... visibility_labels.key <operator> (<value>) where ( and ) are optional
			queries = append(queries, fmt.Sprintf("%[1]s.key=$%[2]d AND %[1]s.val %[3]s %[4]s$%[5]d%[6]s", visibilityLabelsTable, j, option.Operator, preRightOperand, j+1, postRightOperand))
			queryParams = append(queryParams, option.LeftOp)
			queryParams = append(queryParams, option.RightOp)
			j += 2
		} else {
			//... visibilities.<column> <operator> (<value>) where ( and ) are optional
			queries = append(queries, fmt.Sprintf("%s.%s %s %s$%d%s", visibilityTable, option.LeftOp, option.Operator, preRightOperand, j, postRightOperand))
			queryParams = append(queryParams, option.RightOp)
			j++
		}
	}

	baseQuery += strings.Join(queries, " AND ")
	if hasInOperator {
		fmt.Println("SHOULD REBIND")
	}

	fmt.Println(baseQuery)
}
