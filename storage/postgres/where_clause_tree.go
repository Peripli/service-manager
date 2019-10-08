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
	"github.com/Peripli/service-manager/pkg/query"
	"strings"
)

type logicalOperator string

const (
	AND logicalOperator = "AND"
	OR  logicalOperator = "OR"
)

// whereClauseTree represents an sql where clause as tree structure with AND/OR on the nodes
type whereClauseTree struct {
	operator  logicalOperator
	criterion query.Criterion
	children  []*whereClauseTree
}

func (t *whereClauseTree) isLeaf() bool {
	return len(t.children) == 0
}

func (t *whereClauseTree) compileSQL(dbTags []tagType) (string, []interface{}, error) {
	if t.isLeaf() {
		sql, queryParam, err := criterionSQL(t.criterion, dbTags)
		if err != nil {
			return "", nil, err
		}
		return sql, []interface{}{queryParam}, nil
	}
	queryParams := make([]interface{}, 0)
	childrenSQL := make([]string, 0)
	for _, child := range t.children {
		childSQL, childQueryParams, err := child.compileSQL(dbTags)
		if err != nil {
			return "", nil, err
		}
		childrenSQL = append(childrenSQL, childSQL)
		queryParams = append(queryParams, childQueryParams...)
	}
	sep := " " + string(t.operator) + " "
	sql := fmt.Sprintf("(%s)", strings.Join(childrenSQL, sep))
	return sql, queryParams, nil
}

func treesFromCriteria(criteria ...query.Criterion) []*whereClauseTree {
	trees := make([]*whereClauseTree, 0, len(criteria))
	for _, criterion := range criteria {
		trees = append(trees, &whereClauseTree{criterion: criterion})
	}
	return trees
}

func and(trees []*whereClauseTree) *whereClauseTree {
	return &whereClauseTree{
		operator: AND,
		children: trees,
	}
}

func or(trees []*whereClauseTree) *whereClauseTree {
	return &whereClauseTree{
		operator: OR,
		children: trees,
	}
}
