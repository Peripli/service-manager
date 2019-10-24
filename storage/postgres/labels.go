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
	"reflect"
	"strings"
	"time"

	"github.com/Peripli/service-manager/pkg/util"

	"github.com/Peripli/service-manager/pkg/query"

	"github.com/gofrs/uuid"

	"github.com/Peripli/service-manager/pkg/log"
	"github.com/jmoiron/sqlx"
)

func updateLabelsAbstract(ctx context.Context, newLabelFunc func(labelID string, labelKey string, labelValue string) (PostgresLabel, error), pgDB pgDB, referenceID string, updateActions []*query.LabelChange) error {
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
			pgLabel, err := newLabelFunc("", "", "")
			if err != nil {
				return err
			}
			if err := removeLabel(ctx, pgDB, pgLabel, referenceID, action.Key, action.Values...); err != nil {
				return err
			}
		}
	}
	return nil
}
func addLabel(ctx context.Context, newLabelFunc func(labelID string, labelKey string, labelValue string) (PostgresLabel, error), db pgDB, key string, value string, referenceID string) error {
	uuids, err := uuid.NewV4()
	if err != nil {
		return fmt.Errorf("could not generate id for new label: %v", err)
	}
	labelID := uuids.String()
	newLabel, err := newLabelFunc(labelID, key, value)
	if err != nil {
		return err
	}
	labelTable := newLabel.LabelsTableName()
	referenceColumnName := newLabel.ReferenceColumn()

	query := fmt.Sprintf("SELECT * FROM %s WHERE key=$1 and val=$2 and %s=$3", labelTable, referenceColumnName)
	log.C(ctx).Debugf("Executing query %s", query)

	err = db.GetContext(ctx, newLabel, query, key, value, referenceID)
	if checkSQLNoRows(err) == util.ErrNotFoundInStorage {
		if err := create(ctx, db, labelTable, newLabel, newLabel); err != nil {
			return err
		}
	} else {
		log.C(ctx).Debugf("Nothing to create. Label with key=%s value=%s %s=%s already exists in table %s", key, value, referenceColumnName, referenceID, labelTable)
	}
	return nil
}

func removeLabel(ctx context.Context, execer sqlx.ExtContext, label PostgresLabel, referenceID, labelKey string, labelValues ...string) error {
	labelTableName := label.LabelsTableName()
	referenceColumnName := label.ReferenceColumn()
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

func findTagType(tags []tagType, tagName string) reflect.Type {
	for _, tag := range tags {
		if strings.Split(tag.Tag, ",")[0] == tagName {
			return tag.Type
		}
	}
	return nil
}

var (
	intType   = reflect.TypeOf(int(1))
	int64Type = reflect.TypeOf(int64(1))
	timeType  = reflect.TypeOf(time.Time{})
)

func determineCastByType(tagType reflect.Type) string {
	dbCast := ""
	switch tagType {
	case intType:
		fallthrough
	case int64Type:
		fallthrough
	case timeType:
		dbCast = ""

	default:
		dbCast = "::text"
	}
	return dbCast
}

func hasMultiVariateOp(criteria []query.Criterion) bool {
	for _, opt := range criteria {
		if opt.Operator.Type() == query.MultivariateOperator {
			return true
		}
	}
	return false
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
