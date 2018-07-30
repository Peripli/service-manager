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
	"database/sql"
	"fmt"
	"strings"

	"github.com/Peripli/service-manager/pkg/util"
	"github.com/fatih/structs"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"github.com/sirupsen/logrus"
)

func create(db *sqlx.DB, table string, dto interface{}) error {
	set := getDBTags(dto)

	if len(set) == 0 {
		return fmt.Errorf("%s insert: No fields to insert", table)
	}

	query := fmt.Sprintf(
		"INSERT INTO %s (%s) VALUES(:%s)",
		table,
		strings.Join(set, ", "),
		strings.Join(set, ", :"),
	)
	logrus.Debugf("Insert query %s", query)
	_, err := db.NamedExec(query, dto)
	return checkUniqueViolation(err)
}

func get(db *sqlx.DB, id string, table string, dto interface{}) error {
	query := "SELECT * FROM " + table + " WHERE id=$1"
	err := db.Get(dto, query, &id)
	return checkSQLNoRows(err)
}

func getAll(db *sqlx.DB, table string, dtos interface{}) error {
	query := "SELECT * FROM " + table
	return db.Select(dtos, query)
}

func delete(db *sqlx.DB, id string, table string) error {
	query := "DELETE FROM " + table + " WHERE id=$1"

	result, err := db.Exec(query, &id)
	if err != nil {
		return err
	}
	return checkRowsAffected(result)
}

func update(db *sqlx.DB, table string, dto interface{}) error {
	updateQueryString := updateQuery(table, dto)
	if updateQueryString == "" {
		logrus.Debugf("%s update: Nothing to update", table)
		return nil
	}
	logrus.Debugf("Update query %s", updateQueryString)
	result, err := db.NamedExec(updateQueryString, dto)
	if err = checkUniqueViolation(err); err != nil {
		return err
	}
	return checkRowsAffected(result)
}

func getDBTags(structure interface{}) []string {
	s := structs.New(structure)
	fields := s.Fields()
	set := make([]string, 0, len(fields))

	for _, field := range fields {
		if field.IsEmbedded() || field.IsZero() {
			continue
		}
		dbTag := field.Tag("db")
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

func checkUniqueViolation(err error) error {
	if err == nil {
		return nil
	}
	sqlErr, ok := err.(*pq.Error)
	if ok && sqlErr.Code.Name() == "unique_violation" {
		logrus.Debug(sqlErr)
		return util.ErrAlreadyExistsInStorage
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
