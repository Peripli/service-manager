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
	"context"
	"database/sql"
	"fmt"
	"reflect"
	"strings"

	"github.com/jmoiron/sqlx"

	"github.com/Peripli/service-manager/pkg/log"
	"github.com/Peripli/service-manager/pkg/util"
	"github.com/fatih/structs"
	"github.com/lib/pq"
)

type prepareNamedContext interface {
	PrepareNamedContext(ctx context.Context, query string) (*sqlx.NamedStmt, error)
}

type namedExecerContext interface {
	NamedExecContext(ctx context.Context, query string, arg interface{}) (sql.Result, error)
}

type namedQuerierContext interface {
	NamedQuery(query string, arg interface{}) (*sqlx.Rows, error)
}

type selecterContext interface {
	SelectContext(ctx context.Context, dest interface{}, query string, args ...interface{}) error
}

type getterContext interface {
	GetContext(ctx context.Context, dest interface{}, query string, args ...interface{}) error
}

type pgDB interface {
	prepareNamedContext
	namedExecerContext
	namedQuerierContext
	selecterContext
	getterContext
	sqlx.ExtContext
}

func create(ctx context.Context, db pgDB, table string, dto interface{}) (string, error) {
	var lastInsertId string
	set := getDBTags(dto)

	if len(set) == 0 {
		return lastInsertId, fmt.Errorf("%s insert: No fields to insert", table)
	}

	query := fmt.Sprintf(
		"INSERT INTO %s (%s) VALUES(:%s)",
		table,
		strings.Join(set, ", "),
		strings.Join(set, ", :"),
	)

	id, ok := structs.New(dto).FieldOk("ID")
	if ok {
		queryReturningID := fmt.Sprintf("%s Returning %s", query, id.Tag("db"))
		log.C(ctx).Debugf("Executing query %s", queryReturningID)
		stmt, err := db.PrepareNamedContext(ctx, queryReturningID)
		if err != nil {
			return "", err
		}
		err = stmt.GetContext(ctx, &lastInsertId, dto)
		return lastInsertId, checkIntegrityViolation(ctx, checkUniqueViolation(ctx, err))
	}
	log.C(ctx).Debugf("Executing query %s", query)
	_, err := db.NamedExecContext(ctx, query, dto)

	return lastInsertId, checkIntegrityViolation(ctx, checkUniqueViolation(ctx, err))
}

func get(ctx context.Context, db getterContext, id string, table string, dto interface{}) error {
	query := "SELECT * FROM " + table + " WHERE id=$1"
	log.C(ctx).Debugf("Executing query %s", query)
	err := db.GetContext(ctx, dto, query, &id)
	return checkSQLNoRows(err)
}

func list(ctx context.Context, db selecterContext, table string, filter map[string][]string, dtos interface{}) error {
	query := "SELECT * FROM " + table
	if len(filter) != 0 {
		andPairs := make([]string, 0)
		for key, values := range filter {
			orPairs := make([]string, 0)
			for _, value := range values {
				if value != "" {
					orPairs = append(orPairs, fmt.Sprintf("%s='%s'", key, value))
				} else {
					orPairs = append(orPairs, fmt.Sprintf("%s IS NULL", key))
				}
			}
			orPair := " (" + strings.Join(orPairs, " OR ") + ") "
			andPairs = append(andPairs, orPair)
		}
		query += " WHERE " + strings.Join(andPairs, " AND ")
	}
	log.C(ctx).Debugf("Executing query %s", query)
	return db.SelectContext(ctx, dtos, query)
}

func remove(ctx context.Context, db sqlx.ExecerContext, id string, table string) error {
	query := "DELETE FROM " + table + " WHERE id=$1"
	log.C(ctx).Debugf("Executing query %s", query)
	result, err := db.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}
	return checkRowsAffected(result)
}

func update(ctx context.Context, db namedExecerContext, table string, dto interface{}) error {
	updateQueryString := updateQuery(table, dto)
	if updateQueryString == "" {
		log.C(ctx).Debugf("%s update: Nothing to update", table)
		return nil
	}
	log.C(ctx).Debugf("Executing query %s", updateQueryString)
	result, err := db.NamedExecContext(ctx, updateQueryString, dto)
	if err = checkIntegrityViolation(ctx, checkUniqueViolation(ctx, err)); err != nil {
		return err
	}
	return checkRowsAffected(result)
}

func getDBTags(structure interface{}) []string {
	s := structs.New(structure)
	fields := s.Fields()
	set := make([]string, 0, len(fields))

	for _, field := range fields {
		if field.IsEmbedded() || (field.Kind() == reflect.Ptr && field.IsZero()) {
			continue
		}
		dbTag := field.Tag("db")
		if dbTag == "-" {
			continue
		}
		if dbTag == "" {
			dbTag = strings.ToLower(field.Name())
		}
		set = append(set, dbTag)
	}
	return set
}

func updateQuery(tableName string, structure interface{}) string {
	dbTags := getDBTags(structure)
	set := make([]string, 0, len(dbTags))
	for _, dbTag := range dbTags {
		set = append(set, fmt.Sprintf("%s = :%s", dbTag, dbTag))
	}
	if len(set) == 0 {
		return ""
	}
	return fmt.Sprintf("UPDATE "+tableName+" SET %s WHERE id = :id",
		strings.Join(set, ", "))
}

func checkUniqueViolation(ctx context.Context, err error) error {
	if err == nil {
		return nil
	}
	sqlErr, ok := err.(*pq.Error)
	if ok && sqlErr.Code.Name() == "unique_violation" {
		log.C(ctx).Debug(sqlErr)
		return util.ErrAlreadyExistsInStorage
	}
	return err
}

func checkIntegrityViolation(ctx context.Context, err error) error {
	if err == nil {
		return nil
	}
	sqlErr, ok := err.(*pq.Error)
	if ok && (sqlErr.Code.Class() == "42" || sqlErr.Code.Class() == "44" || sqlErr.Code.Class() == "23") {
		log.C(ctx).Debug(sqlErr)
		return util.ErrBadRequestStorage(err)
	}
	return err
}

func checkRowsAffected(result sql.Result) error {
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected < 1 {
		return util.ErrNotFoundInStorage
	}
	return nil
}

func checkSQLNoRows(err error) error {
	if err == sql.ErrNoRows {
		return util.ErrNotFoundInStorage
	}
	return err
}
