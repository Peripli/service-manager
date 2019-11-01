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

package postgres

import (
	"fmt"
	"strings"

	"github.com/Peripli/service-manager/pkg/query"
)

type logicalOperator string

const (
	AND logicalOperator = "AND"
	OR  logicalOperator = "OR"
)

// whereClauseTree represents an sql where clause as tree structure with AND/OR on the nodes
type whereClauseTree struct {
	criterion query.Criterion
	dbTags    []tagType
	tableName string

	operator logicalOperator
	children []*whereClauseTree
}

func (t *whereClauseTree) isLeaf() bool {
	return len(t.children) == 0
}

func (t *whereClauseTree) isEmpty() bool {
	return t.criterion.Operator == nil && len(t.children) == 0
}

func (t *whereClauseTree) compileSQL() (string, []interface{}) {
	if t.isEmpty() {
		return "", []interface{}{}
	}
	if t.isLeaf() {
		sql, queryParam := criterionSQL(t.criterion, t.dbTags, t.tableName)
		return sql, []interface{}{queryParam}
	}
	queryParams := make([]interface{}, 0)
	childrenSQL := make([]string, 0)
	for _, child := range t.children {
		childSQL, childQueryParams := child.compileSQL()
		if len(childSQL) != 0 {
			childrenSQL = append(childrenSQL, childSQL)
			queryParams = append(queryParams, childQueryParams...)
		}
	}
	var sql string
	childrenCount := len(childrenSQL)
	switch childrenCount {
	case 0:
		sql = ""
	case 1:
		sql = childrenSQL[0]
	default:
		sql = fmt.Sprintf("(%s)", strings.Join(childrenSQL, fmt.Sprintf(" %s ", t.operator)))
	}

	return sql, queryParams
}

func criterionSQL(c query.Criterion, dbTags []tagType, tableAlias string) (string, interface{}) {
	rightOpBindVar, rightOpQueryValue := buildRightOp(c.Operator, c.RightOp)
	sqlOperation := translateOperationToSQLEquivalent(c.Operator)

	ttype := findTagType(dbTags, c.LeftOp)
	dbCast := determineCastByType(ttype)
	var clause string
	if tableAlias != "" {
		clause = fmt.Sprintf("%s.%s%s %s %s", tableAlias, c.LeftOp, dbCast, sqlOperation, rightOpBindVar)
	} else {
		clause = fmt.Sprintf("%s%s %s %s", c.LeftOp, dbCast, sqlOperation, rightOpBindVar)
	}
	if c.Operator.IsNullable() {
		clause = fmt.Sprintf("(%s OR %s IS NULL)", clause, c.LeftOp)
	}
	return clause, rightOpQueryValue
}

func buildRightOp(operator query.Operator, rightOp []string) (string, interface{}) {
	rightOpBindVar := "?"
	var rhs interface{}
	if operator.Type() == query.MultivariateOperator {
		rightOpBindVar = "(?)"
		rhs = rightOp
	} else {
		rhs = rightOp[0]
	}
	return rightOpBindVar, rhs
}

func translateOperationToSQLEquivalent(operator query.Operator) string {
	switch operator {
	case query.LessThanOperator:
		return "<"
	case query.LessThanOrEqualOperator:
		return "<="
	case query.GreaterThanOperator:
		return ">"
	case query.GreaterThanOrEqualOperator:
		return ">="
	case query.NotInOperator:
		return "NOT IN"
	case query.EqualsOperator:
		fallthrough
	case query.EqualsOrNilOperator:
		return "="
	case query.NotEqualsOperator:
		return "!="
	default:
		return strings.ToUpper(operator.String())
	}
}
