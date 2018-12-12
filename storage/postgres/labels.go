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

	"github.com/gofrs/uuid"

	"github.com/Peripli/service-manager/pkg/log"
	"github.com/jmoiron/sqlx"
)

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

func removeLabel(ctx context.Context, execer sqlx.ExecerContext, labelsTableName string, labelKey string, labelValues ...string) error {
	baseQuery := fmt.Sprintf("DELETE FROM %s WHERE key=$1", labelsTableName)
	if len(labelValues) == 0 {
		return removeAllLabelsWithKey(ctx, execer, baseQuery, labelKey)
	}
	baseQuery += " AND val=$2"
	return removeLabelValues(ctx, execer, baseQuery, labelKey, labelValues)
}

func removeLabelValues(ctx context.Context, execerContext sqlx.ExecerContext, query string, labelKey string, labelValues []string) error {
	for _, labelValue := range labelValues {
		if err := execute(ctx, query, func() (sql.Result, error) {
			return execerContext.ExecContext(ctx, query, labelKey, labelValue)
		}); err != nil {
			return err
		}
	}
	return nil
}

func removeAllLabelsWithKey(ctx context.Context, execerContext sqlx.ExecerContext, query string, key string) error {
	return execute(ctx, query, func() (sql.Result, error) {
		return execerContext.ExecContext(ctx, query, key)
	})
}

func execute(ctx context.Context, query string, f func() (sql.Result, error)) error {
	log.C(ctx).Debugf("Executing query %s", query)
	result, err := f()
	if err != nil {
		return err
	}
	return checkRowsAffected(result)
}
