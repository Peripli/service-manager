/*
 * Copyright 2018 The Service Manager Authors
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

package postgres

import (
	"fmt"
	"strings"

	"github.com/jmoiron/sqlx"

	"github.com/Peripli/service-manager/pkg/selection"
)

func buildListQueryWithParams(query string, baseTableName string, labelsTableName string, criteria []selection.Criterion) (string, []interface{}, error) {
	if criteria == nil || len(criteria) == 0 {
		return query, nil, nil
	}

	var queryParams []interface{}
	var queries []string

	query += " WHERE "
	for _, option := range criteria {
		rightOpBindVar, rightOpQueryValue := buildRightOp(option)
		sqlOperation := translateOperationToSQLEquivalent(option.Operator)
		if option.Type == selection.LabelQuery {
			queries = append(queries, fmt.Sprintf("%[1]s.key = ? AND %[1]s.val %[2]s %s", labelsTableName, sqlOperation, rightOpBindVar))
			queryParams = append(queryParams, option.LeftOp)
		} else {
			clause := fmt.Sprintf("%s.%s %s %s", baseTableName, option.LeftOp, sqlOperation, rightOpBindVar)
			if option.Operator.IsNullable() {
				clause = fmt.Sprintf("(%s OR %s.%s IS NULL)", clause, baseTableName, option.LeftOp)
			}
			queries = append(queries, clause)
		}
		queryParams = append(queryParams, rightOpQueryValue)
	}
	query += strings.Join(queries, " AND ") + ";"

	if hasMultiVariateOp(criteria) {
		var err error
		// sqlx.In requires question marks(?) instead of positional arguments (the ones pgsql uses) in order to map the list argument to the IN operation
		if query, queryParams, err = sqlx.In(query, queryParams...); err != nil {
			return "", nil, err
		}
	}
	return query, queryParams, nil
}

func buildRightOp(criterion selection.Criterion) (string, interface{}) {
	rightOpBindVar := "?"
	var rhs interface{} = criterion.RightOp[0]
	if criterion.Operator.IsMultiVariate() {
		rightOpBindVar = "(?)"
		rhs = criterion.RightOp
	}
	return rightOpBindVar, rhs
}

func hasMultiVariateOp(criteria []selection.Criterion) bool {
	for _, opt := range criteria {
		if opt.Operator.IsMultiVariate() {
			return true
		}
	}
	return false
}

func translateOperationToSQLEquivalent(operator selection.Operator) string {
	switch operator {
	case selection.LessThanOperator:
		return "<"
	case selection.GreaterThanOperator:
		return ">"
	case selection.NotInOperator:
		return "NOT IN"
	case selection.EqualsOrNilOperator:
		return "="
	default:
		return string(operator)
	}
}
