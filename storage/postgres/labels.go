/*
 * Copyright 2018 The Service Manager Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/Peripli/service-manager/pkg/query"

	"github.com/gofrs/uuid"

	"github.com/Peripli/service-manager/pkg/log"
	"github.com/jmoiron/sqlx"
)

func updateLabels(ctx context.Context, newLabelFunc func(labelID string, labelKey string, labelValue string) interface{}, pgDB pgDB, labelsTableName, referenceColumnName, referenceID string, updateActions []query.LabelChange) error {
	for _, action := range updateActions {
		switch action.Operation {
		case query.AddLabelOperation:
			fallthrough
		case query.AddLabelValuesOperation:
			if err := addLabel(ctx, newLabelFunc, pgDB, labelsTableName, action.Key, action.Values...); err != nil {
				return err
			}
		case query.RemoveLabelOperation:
			fallthrough
		case query.RemoveLabelValuesOperation:
			if err := removeLabel(ctx, pgDB, labelsTableName, referenceColumnName, referenceID, action.Key, action.Values...); err != nil {
				return err
			}
		}
	}
	return nil
}

func addLabel(ctx context.Context, f func(labelID string, labelKey string, labelValue string) interface{}, db pgDB, labelsTable string, key string, values ...string) error {
	for _, labelValue := range values {
		uuids, err := uuid.NewV4()
		if err != nil {
			return fmt.Errorf("could not generate id for new label: %v", err)
		}
		labelID := uuids.String()
		newLabel := f(labelID, key, labelValue)
		if _, err := create(ctx, db, labelsTable, newLabel); err != nil {
			return err
		}
	}
	return nil
}

func removeLabel(ctx context.Context, execer sqlx.ExecerContext, labelsTableName, referenceColumnName, referenceID, labelKey string, labelValues ...string) error {
	baseQuery := fmt.Sprintf("DELETE FROM %s WHERE key=$1 AND %s=$2", labelsTableName, referenceColumnName)
	labelValuesCount := len(labelValues)
	// remove all labels with this key
	if labelValuesCount == 0 {
		return execute(ctx, baseQuery, func() (sql.Result, error) {
			return execer.ExecContext(ctx, baseQuery, labelKey, referenceID)
		})
	}
	// remove labels with a specific key and a value which is in the provided list
	baseQuery += " AND val IN $3"
	sqlQuery, queryParams, err := sqlx.In(baseQuery, referenceID, labelKey, labelValues)
	if err != nil {
		return err
	}
	return execute(ctx, sqlQuery, func() (sql.Result, error) {
		return execer.ExecContext(ctx, sqlQuery, queryParams)
	})
}

func buildListQueryWithParams(sqlQuery string, baseTableName string, labelsTableName string, criteria []query.Criterion) (string, []interface{}, error) {
	if len(criteria) == 0 {
		return sqlQuery, nil, nil
	}

	var queryParams []interface{}
	var queries []string

	sqlQuery += " WHERE "
	for _, option := range criteria {
		rightOpBindVar, rightOpQueryValue := buildRightOp(option)
		sqlOperation := translateOperationToSQLEquivalent(option.Operator)
		if option.Type == query.LabelQuery {
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
	sqlQuery += strings.Join(queries, " AND ") + ";"

	if hasMultiVariateOp(criteria) {
		var err error
		// sqlx.In requires question marks(?) instead of positional arguments (the ones pgsql uses) in order to map the list argument to the IN operation
		if sqlQuery, queryParams, err = sqlx.In(sqlQuery, queryParams...); err != nil {
			return "", nil, err
		}
	}
	return sqlQuery, queryParams, nil
}

func buildRightOp(criterion query.Criterion) (string, interface{}) {
	rightOpBindVar := "?"
	var rhs interface{} = criterion.RightOp[0]
	if criterion.Operator.IsMultiVariate() {
		rightOpBindVar = "(?)"
		rhs = criterion.RightOp
	}
	return rightOpBindVar, rhs
}

func hasMultiVariateOp(criteria []query.Criterion) bool {
	for _, opt := range criteria {
		if opt.Operator.IsMultiVariate() {
			return true
		}
	}
	return false
}

func translateOperationToSQLEquivalent(operator query.Operator) string {
	switch operator {
	case query.LessThanOperator:
		return "<"
	case query.GreaterThanOperator:
		return ">"
	case query.NotInOperator:
		return "NOT IN"
	case query.EqualsOrNilOperator:
		return "="
	default:
		return strings.ToUpper(string(operator))
	}
}

func execute(ctx context.Context, query string, f func() (sql.Result, error)) error {
	log.C(ctx).Debugf("Executing query %s", query)
	result, err := f()
	if err != nil {
		return err
	}
	return checkRowsAffected(result)
}
