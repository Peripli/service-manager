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
	"fmt"
	"strings"

	"github.com/Peripli/service-manager/pkg/util"

	"github.com/Peripli/service-manager/pkg/query"

	"github.com/gofrs/uuid"

	"github.com/Peripli/service-manager/pkg/log"
	"github.com/jmoiron/sqlx"
)

func updateLabelsAbstract(ctx context.Context, newLabelFunc func(labelID string, labelKey string, labelValue string) Labelable, pgDB pgDB, referenceID string, updateActions []*query.LabelChange) error {
	for _, action := range updateActions {
		switch action.Operation {
		case query.AddLabelOperation:
			fallthrough
		case query.AddLabelValuesOperation:
			for _, labelValue := range action.Values {
				if err := addLabel(ctx, newLabelFunc, pgDB, action.Key, labelValue, referenceID); err != nil {
					return err
				}
			}
		case query.RemoveLabelOperation:
			fallthrough
		case query.RemoveLabelValuesOperation:
			pgLabel := newLabelFunc("", "", "")
			if err := removeLabel(ctx, pgDB, pgLabel, referenceID, action.Key, action.Values...); err != nil {
				return err
			}
		}
	}
	return nil
}

func addLabel(ctx context.Context, newLabelFunc func(labelID string, labelKey string, labelValue string) Labelable, db pgDB, key string, value string, referenceID string) error {
	uuids, err := uuid.NewV4()
	if err != nil {
		return fmt.Errorf("could not generate id for new label: %v", err)
	}
	labelID := uuids.String()
	newLabel := newLabelFunc(labelID, key, value)
	labelTable, referenceColumnName, _ := newLabel.Label()

	query := fmt.Sprintf("SELECT * FROM %s WHERE key=$1 and val=$2 and %s=$3", labelTable, referenceColumnName)
	log.C(ctx).Debugf("Executing query %s", query)

	err = db.GetContext(ctx, newLabel, query, key, value, referenceID)
	if checkSQLNoRows(err) == util.ErrNotFoundInStorage {
		if _, err := create(ctx, db, labelTable, newLabel); err != nil {
			return err
		}
	} else {
		log.C(ctx).Debugf("Nothing to create. Label with key=%s value=%s %s=%s already exists in table %s", key, value, referenceColumnName, referenceID, labelTable)
	}
	return nil
}

func removeLabel(ctx context.Context, execer sqlx.ExtContext, labelable Labelable, referenceID, labelKey string, labelValues ...string) error {
	labelTableName, referenceColumnName, _ := labelable.Label()
	baseQuery := fmt.Sprintf("DELETE FROM %s WHERE key=? AND %s=?", labelTableName, referenceColumnName)
	args := []interface{}{labelKey, referenceID}
	// remove all labels with this key
	if len(labelValues) == 0 {
		if err := executeNew(ctx, execer, baseQuery, args); err != nil {
			if err == util.ErrNotFoundInStorage {
				log.C(ctx).Debugf("Nothing to delete. Label with key=%s %s=%s not found in table %s", labelKey, referenceColumnName, referenceID, labelTableName)
				return nil
			}
			return err
		}
		return nil
	}
	// remove labels with a specific key and a value which is in the provided list
	args = append(args, labelValues)
	baseQuery += " AND val IN (?)"
	sqlQuery, queryParams, err := sqlx.In(baseQuery, args...)
	if err != nil {
		return err
	}

	if err := executeNew(ctx, execer, sqlQuery, queryParams); err != nil {
		if err == util.ErrNotFoundInStorage {
			log.C(ctx).Debugf("Nothing to delete. Label with key=%s values in %s %s=%s not found in table %s", labelKey, labelValues, referenceColumnName, referenceID, labelTableName)
			return nil
		}
		return err
	}
	return nil
}

func buildQueryWithParams(extContext sqlx.ExtContext, sqlQuery string, baseTableName string, labelable Labelable, criteria []query.Criterion) (string, []interface{}, error) {
	if len(criteria) == 0 {
		return sqlQuery + ";", nil, nil
	}

	var queryParams []interface{}
	var fieldQueries []string
	var labelQueries []string

	labelCriteria, fieldCriteria := splitCriteriaByType(criteria)

	if len(labelCriteria) > 0 {
		labelTableName, referenceColumnName, _ := labelable.Label()
		labelSubQuery := fmt.Sprintf("(SELECT * FROM %[1]s WHERE %[2]s IN (SELECT %[2]s FROM %[1]s WHERE ", labelTableName, referenceColumnName)
		for _, option := range labelCriteria {
			rightOpBindVar, rightOpQueryValue := buildRightOp(option)
			sqlOperation := translateOperationToSQLEquivalent(option.Operator)
			labelQueries = append(labelQueries, fmt.Sprintf("(%[1]s.key = ? AND %[1]s.val %[2]s %s)", labelTableName, sqlOperation, rightOpBindVar))
			queryParams = append(queryParams, option.LeftOp, rightOpQueryValue)
		}
		labelSubQuery += strings.Join(labelQueries, " OR ")
		labelSubQuery += "))"

		sqlQuery = strings.Replace(sqlQuery, "LEFT JOIN", "JOIN "+labelSubQuery, 1)
	}

	if len(fieldCriteria) > 0 {
		sqlQuery += " WHERE "
		for _, option := range fieldCriteria {
			rightOpBindVar, rightOpQueryValue := buildRightOp(option)
			sqlOperation := translateOperationToSQLEquivalent(option.Operator)
			clause := fmt.Sprintf("%s.%s::text %s %s", baseTableName, option.LeftOp, sqlOperation, rightOpBindVar)
			if option.Operator.IsNullable() {
				clause = fmt.Sprintf("(%s OR %s.%s IS NULL)", clause, baseTableName, option.LeftOp)
			}
			fieldQueries = append(fieldQueries, clause)
			queryParams = append(queryParams, rightOpQueryValue)
		}
		sqlQuery += strings.Join(fieldQueries, " AND ")
	}
	sqlQuery += ";"

	if hasMultiVariateOp(criteria) {
		var err error
		// sqlx.In requires question marks(?) instead of positional arguments (the ones pgsql uses) in order to map the list argument to the IN operation
		if sqlQuery, queryParams, err = sqlx.In(sqlQuery, queryParams...); err != nil {
			return "", nil, err
		}
	}
	sqlQuery = extContext.Rebind(sqlQuery)
	return sqlQuery, queryParams, nil
}

func splitCriteriaByType(criteria []query.Criterion) ([]query.Criterion, []query.Criterion) {
	var labelQueries []query.Criterion
	var fieldQueries []query.Criterion

	for _, criterion := range criteria {
		if criterion.Type == query.FieldQuery {
			fieldQueries = append(fieldQueries, criterion)
		} else {
			labelQueries = append(labelQueries, criterion)
		}
	}

	return labelQueries, fieldQueries
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

func executeNew(ctx context.Context, extContext sqlx.ExtContext, query string, args []interface{}) error {
	query = extContext.Rebind(query)
	log.C(ctx).Debugf("Executing query %s", query)
	result, err := extContext.ExecContext(ctx, query, args...)
	if err != nil {
		return err
	}
	return checkRowsAffected(ctx, result)
}
