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
	AND       logicalOperator = "AND"
	INTERSECT logicalOperator = "INTERSECT"
)

// treeSqlBuilder is a helper struct to allow for dynamic changing of the function that builds the sql template statements
type treeSqlBuilder struct {
	buildSQL func(childrenSQL []string) string
}

var defaultTreeSqlBuilder = &treeSqlBuilder{
	buildSQL: func(childrenSQL []string) string {
		return fmt.Sprintf("(%s)", strings.Join(childrenSQL, fmt.Sprintf(" %s ", AND)))
	},
}

// whereClauseTree represents an sql where clause as tree structure with AND/OR on the nodes
type whereClauseTree struct {
	criterion query.Criterion
	dbTags    []tagType
	tableName string

	children   []*whereClauseTree
	sqlBuilder *treeSqlBuilder
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
			if child.criterion.Type != query.ExistQuery {
				queryParams = append(queryParams, childQueryParams...)
			}
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
		if t.sqlBuilder == nil {
			t.sqlBuilder = defaultTreeSqlBuilder
		}
		sql = t.sqlBuilder.buildSQL(childrenSQL)
	}

	return sql, queryParams
}

func criterionSQL(c query.Criterion, dbTags []tagType, tableAlias string) (string, interface{}) {
	rightOpBindVar, rightOpQueryValue := buildRightOp(c.Operator, c.RightOp)
	sqlOperation := translateOperationToSQLEquivalent(c.Operator)
	column := strings.Split(c.LeftOp, "/")[0]
	ttype := findTagType(dbTags, column)
	dbCast := determineCastByType(ttype)
	if ttype == jsonType {
		var isCompound bool
		c.LeftOp, isCompound = convertToJsonKey(c.LeftOp)
		if isCompound {
			dbCast = ""
		}
	}
	var clause string
	if c.Type == query.ExistQuery {
		clause = fmt.Sprintf("%s (%s)", sqlOperation, rightOpQueryValue.(string))
	} else if tableAlias != "" {
		clause = fmt.Sprintf("%s.%s%s %s %s", tableAlias, c.LeftOp, dbCast, sqlOperation, rightOpBindVar)
	} else {
		clause = fmt.Sprintf("%s%s %s %s", c.LeftOp, dbCast, sqlOperation, rightOpBindVar)
	}
	if c.Operator.IsNullable() {
		clause = fmt.Sprintf("(%s OR %s IS NULL)", clause, c.LeftOp)
	}
	return clause, rightOpQueryValue
}

func convertToJsonKey(key string) (string, bool) {
	columnParts := strings.Split(key, "/")
	if len(columnParts) == 1 {
		return columnParts[0], false
	} else {
		result := columnParts[0]
		for i := 1; i < len(columnParts)-1; i++ {
			result += fmt.Sprintf("%s'%s'", "->", columnParts[i])
		}
		result += fmt.Sprintf("%s'%s'", "->>", columnParts[len(columnParts)-1])
		return result, true
	}
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
	case query.ExistsSubquery:
		return "EXISTS"
	case query.NotExistsSubquery:
		return "NOT EXISTS"
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
