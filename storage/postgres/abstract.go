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
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"github.com/Peripli/service-manager/pkg/query"

	"github.com/jmoiron/sqlx"
	sqlxtypes "github.com/jmoiron/sqlx/types"

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

//go:generate counterfeiter . pgDB
// pgDB represents a PG database API
type pgDB interface {
	prepareNamedContext
	namedExecerContext
	namedQuerierContext
	selecterContext
	getterContext
	sqlx.ExtContext
}

func create(ctx context.Context, db pgDB, table string, dto interface{}) (string, error) {
	var lastInsertID string
	setTagType := getDBTags(dto, isAutoIncrementable)
	dbTags := make([]string, 0, len(setTagType))
	for _, tagType := range setTagType {
		dbTags = append(dbTags, tagType.Tag)
	}

	if len(dbTags) == 0 {
		return lastInsertID, fmt.Errorf("%s insert: No fields to insert", table)
	}

	sqlQuery := fmt.Sprintf(
		"INSERT INTO %s (%s) VALUES(:%s)",
		table,
		strings.Join(dbTags, ", "),
		strings.Join(dbTags, ", :"),
	)

	id, ok := structs.New(dto).FieldOk("ID")
	if ok {
		queryReturningID := fmt.Sprintf("%s Returning %s", sqlQuery, id.Tag("db"))
		log.C(ctx).Debugf("Executing query %s", queryReturningID)
		stmt, err := db.PrepareNamedContext(ctx, queryReturningID)
		if err != nil {
			return "", err
		}
		err = stmt.GetContext(ctx, &lastInsertID, dto)
		return lastInsertID, checkIntegrityViolation(ctx, checkUniqueViolation(ctx, err))
	}
	log.C(ctx).Debugf("Executing query %s", sqlQuery)
	_, err := db.NamedExecContext(ctx, sqlQuery, dto)
	return lastInsertID, checkIntegrityViolation(ctx, checkUniqueViolation(ctx, err))
}

func listWithLabelsByCriteria(ctx context.Context, db pgDB, baseEntity interface{}, label PostgresLabel, baseTableName string, criteria []query.Criterion) (*sqlx.Rows, error) {
	if err := validateFieldQueryParams(getDBTags(baseEntity, nil), criteria); err != nil {
		return nil, err
	}
	var baseQuery string
	if label == nil {
		baseQuery = constructBaseQueryForEntity(baseTableName)
	} else {
		baseQuery = constructBaseQueryForLabelable(label, baseTableName)
	}
	sqlQuery, queryParams, err := buildQueryWithParams(db, baseQuery, baseTableName, label, criteria, getDBTags(baseEntity, isAutoIncrementable), " ORDER BY created_at")
	if err != nil {
		return nil, err
	}

	// Lock the rows if we are in transaction so that update operations on those rows can rely on unchanged data
	// This allows us to handle concurrent updates on the same rows by executing them sequentially as
	// before updating we have to anyway select the rows and can therefore lock them
	if _, ok := db.(*sqlx.Tx); ok {
		sqlQuery = sqlQuery[:len(sqlQuery)-1]
		sqlQuery += fmt.Sprintf(" FOR SHARE of %s;", baseTableName)
	}

	log.C(ctx).Debugf("Executing query %s", sqlQuery)
	return db.QueryxContext(ctx, sqlQuery, queryParams...)
}

func listByFieldCriteria(ctx context.Context, db pgDB, table string, criteria []query.Criterion) (*sqlx.Rows, error) {
	baseQuery := constructBaseQueryForEntity(table)
	sqlQuery, queryParams, err := buildQueryWithParams(db, baseQuery, table, nil, criteria, nil, " ORDER BY created_at")
	if err != nil {
		return nil, err
	}
	return db.QueryxContext(ctx, sqlQuery, queryParams...)
}

func deleteAllByFieldCriteria(ctx context.Context, extContext sqlx.ExtContext, table string, dto interface{}, criteria []query.Criterion) (*sqlx.Rows, error) {
	for _, criterion := range criteria {
		if criterion.Type != query.FieldQuery {
			return nil, &util.UnsupportedQueryError{Message: "conditional delete is only supported for field queries"}
		}
	}
	if err := validateFieldQueryParams(getDBTags(dto, nil), criteria); err != nil {
		return nil, err
	}
	baseQuery := fmt.Sprintf("DELETE FROM %s", table)
	sqlQuery, queryParams, err := buildQueryWithParams(extContext, baseQuery, table, nil, criteria, getDBTags(dto, isAutoIncrementable))
	if err != nil {
		return nil, err
	}
	sqlQuery = sqlQuery[:len(sqlQuery)-1] + " RETURNING *;"
	return extContext.QueryxContext(ctx, sqlQuery, queryParams...)
}

func validateFieldQueryParams(tags []tagType, criteria []query.Criterion) error {
	availableColumns := make(map[string]bool)
	for _, dbTag := range tags {
		tagValues := strings.Split(dbTag.Tag, ",")
		availableColumns[tagValues[0]] = true
	}
	for _, criterion := range criteria {
		if criterion.Type == query.FieldQuery && !availableColumns[criterion.LeftOp] {
			return &util.UnsupportedQueryError{Message: fmt.Sprintf("unsupported field query key: %s", criterion.LeftOp)}
		}
	}
	return nil
}

func constructBaseQueryForEntity(tableName string) string {
	return fmt.Sprintf("SELECT * FROM %s", tableName)
}

func constructBaseQueryForLabelable(labelsEntity PostgresLabel, baseTableName string) string {
	baseQuery := `SELECT %[1]s.*,`
	for _, dbTag := range getDBTags(labelsEntity, isAutoIncrementable) {
		baseQuery += " %[2]s." + dbTag.Tag + " " + "\"%[2]s." + dbTag.Tag + "\"" + ","
	}
	baseQuery = baseQuery[:len(baseQuery)-1] //remove last comma
	labelsTableName := labelsEntity.LabelsTableName()
	referenceKeyColumn := labelsEntity.ReferenceColumn()
	primaryKeyColumn := labelsEntity.LabelsPrimaryColumn()
	baseQuery += " FROM %[1]s LEFT JOIN %[2]s ON %[1]s." + primaryKeyColumn + " = %[2]s." + referenceKeyColumn
	return fmt.Sprintf(baseQuery, baseTableName, labelsTableName)
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
	return checkRowsAffected(ctx, result)
}

func isAutoIncrementable(tagValue string) bool {
	// auto_increment states that the value will be calculated in the DB
	return strings.Contains(tagValue, "auto_increment")
}

type tagType struct {
	Tag  string
	Type reflect.Type
}

func getDBTags(structure interface{}, predicate func(string) bool) []tagType {
	s := structs.New(structure)
	fields := s.Fields()
	set := make([]tagType, 0, len(fields))
	if predicate == nil {
		predicate = func(string) bool { return false }
	}
	getTags(fields, &set, predicate)
	return set
}

func getTags(fields []*structs.Field, set *[]tagType, predicate func(string) bool) {
	for _, field := range fields {
		if field.Kind() == reflect.Ptr && field.IsZero() {
			continue
		}
		if field.IsEmbedded() {
			embedded := make([]tagType, 0)
			getTags(field.Fields(), &embedded, predicate)
			*set = append(*set, embedded...)
		} else {
			dbTag := field.Tag("db")
			if dbTag == "-" || predicate(dbTag) {
				continue
			}
			if dbTag == "" {
				dbTag = strings.ToLower(field.Name())
			}
			ttype := reflect.ValueOf(field.Value()).Type()
			*set = append(*set, tagType{
				Tag:  dbTag,
				Type: ttype,
			})
		}
	}
}

func updateQuery(tableName string, structure interface{}) string {
	dbTags := getDBTags(structure, isAutoIncrementable)
	set := make([]string, 0, len(dbTags))
	for _, dbTag := range dbTags {
		set = append(set, fmt.Sprintf("%s = :%s", dbTag.Tag, dbTag.Tag))
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

func closeRows(ctx context.Context, rows *sqlx.Rows) {
	if rows == nil {
		return
	}
	if err := rows.Close(); err != nil {
		log.C(ctx).WithError(err).Errorf("Could not release connection")
	}
}

func checkRowsAffected(ctx context.Context, result sql.Result) error {
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected < 1 {
		return util.ErrNotFoundInStorage
	}
	log.C(ctx).Debugf("Operation affected %d rows", rowsAffected)
	return nil
}

func checkSQLNoRows(err error) error {
	if err == sql.ErrNoRows {
		return util.ErrNotFoundInStorage
	}
	return err
}

func toNullString(s string) sql.NullString {
	return sql.NullString{String: s, Valid: s != ""}
}

func getJSONText(item json.RawMessage) sqlxtypes.JSONText {
	if len(item) == len("null") && string(item) == "null" {
		return sqlxtypes.JSONText("{}")
	}
	return sqlxtypes.JSONText(item)
}

func getJSONRawMessage(item sqlxtypes.JSONText) json.RawMessage {
	if len(item) <= len("null") {
		itemStr := string(item)
		if itemStr == "{}" || itemStr == "null" {
			return nil
		}
	}
	return json.RawMessage(item)
}
