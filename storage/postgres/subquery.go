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
	"reflect"
)

type selectSubQuery struct {
	tableName        string
	referenceColumns string
	dbTags           []tagType
	sql              queryStringBuilder
	queryParams      []interface{}

	fieldCriteria []query.Criterion
	orderByFields []orderRule
	limit         string

	whereClauseTreeCreator func([]query.Criterion) *whereClauseTree

	err error
}

func newSelectSubQuery(tableName string, referenceColumns string, dbTags []tagType, creator func([]query.Criterion) *whereClauseTree) *selectSubQuery {
	return &selectSubQuery{
		tableName:              tableName,
		referenceColumns:       referenceColumns,
		dbTags:                 dbTags,
		whereClauseTreeCreator: creator,
	}
}

func (ssq *selectSubQuery) orderBySQL() *selectSubQuery {
	if sql, err := orderBySQL(ssq.orderByFields); err != nil {
		ssq.err = err
		return ssq
	} else {
		ssq.sql.WriteString(sql)
	}
	return ssq
}

func (ssq *selectSubQuery) limitSQL() *selectSubQuery {
	if len(ssq.limit) > 0 {
		ssq.sql.WriteString(fmt.Sprintf(" LIMIT %s", ssq.limit))
	}
	return ssq
}

func (ssq *selectSubQuery) fieldCriteriaSQL() *selectSubQuery {
	if len(ssq.fieldCriteria) > 0 {
		tree := ssq.whereClauseTreeCreator(ssq.fieldCriteria)
		whereSQL, queryParams, err := tree.compileSQL(ssq.dbTags)
		if err != nil {
			ssq.err = err
			return ssq
		}
		ssq.queryParams = append(ssq.queryParams, queryParams...)
		ssq.sql.WriteString(" WHERE " + whereSQL)
	}
	return ssq
}

func (ssq *selectSubQuery) compileSQL() (string, error) {
	columns := columnsByTags(ssq.dbTags)
	if err := validateFieldQueryParams(columns, ssq.fieldCriteria); err != nil {
		return "", err
	}
	if err := validateOrderFields(columns, ssq.orderByFields...); err != nil {
		return "", err
	}
	baseQuery := fmt.Sprintf("(SELECT %s FROM %s", ssq.referenceColumns, ssq.tableName)
	ssq.sql.WriteString(baseQuery)

	ssq.fieldCriteriaSQL().
		orderBySQL().
		limitSQL()

	ssq.sql.WriteString(")")

	if ssq.err != nil {
		return "", ssq.err
	}
	return ssq.sql.String(), nil
}

func criterionSQL(criterion query.Criterion, dbTags []tagType) (string, interface{}, error) {
	var ttype reflect.Type
	if dbTags != nil {
		var err error
		ttype, err = findTagType(dbTags, criterion.LeftOp)
		if err != nil {
			return "", nil, err
		}
	}
	rightOpBindVar, rightOpQueryValue := buildRightOp(criterion)
	sqlOperation := translateOperationToSQLEquivalent(criterion.Operator)

	dbCast := determineCastByType(ttype)
	clause := fmt.Sprintf("%s%s %s %s", criterion.LeftOp, dbCast, sqlOperation, rightOpBindVar)
	if criterion.Operator.IsNullable() {
		clause = fmt.Sprintf("(%s OR %s IS NULL)", clause, criterion.LeftOp)
	}
	return clause, rightOpQueryValue, nil
}

func fromSubQueryWhereSchema(criteria []query.Criterion) *whereClauseTree {
	return and(treesFromCriteria(criteria...))
}

func labelsJoinSubQueryWhereSchema(criteria []query.Criterion) *whereClauseTree {
	trees := make([]*whereClauseTree, 0)
	for i := 0; i < len(criteria)-1; i += 2 {
		trees = append(trees, and(treesFromCriteria(criteria[i], criteria[i+1])))
	}
	return or(trees)
}
