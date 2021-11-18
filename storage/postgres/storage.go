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
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/lib/pq"

	"github.com/Peripli/service-manager/pkg/log"
	"github.com/Peripli/service-manager/pkg/query"
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/pkg/util"
	"github.com/Peripli/service-manager/storage"
	"github.com/golang-migrate/migrate"
	migratepg "github.com/golang-migrate/migrate/database/postgres"
	_ "github.com/golang-migrate/migrate/source/file"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

const (
	postgresDriverName  = "pq-timeouts"
	foreignKeyViolation = "foreign_key_violation"
)

type Storage struct {
	ConnectFunc func(driver string, url string) (*sql.DB, error)

	pgDB                  pgDB
	db                    *sqlx.DB
	queryBuilder          *QueryBuilder
	state                 *storageState
	layerOneEncryptionKey []byte
	scheme                *scheme
	mutex                 sync.Mutex
}

func (ps *Storage) Introduce(entity storage.Entity) {
	ps.scheme.introduce(entity)
}

func (ps *Storage) SelectContext(ctx context.Context, dest interface{}, query string, args ...interface{}) error {
	ps.checkOpen()
	return ps.pgDB.SelectContext(ctx, dest, query, args...)
}

func (ps *Storage) Open(settings *storage.Settings) error {
	if err := settings.Validate(); err != nil {
		return err
	}

	ps.mutex.Lock()
	defer ps.mutex.Unlock()
	if ps.db == nil {
		parsedUrl, err := url.Parse(settings.URI)
		if err != nil {
			return fmt.Errorf("could not parse PostgreSQL URI: %s", err)
		}

		parsedQuery, err := url.ParseQuery(parsedUrl.RawQuery)
		if err != nil {
			return fmt.Errorf("could not parse PostgreSQL URL query: %s", err)
		}

		parsedQuery.Set("read_timeout", strconv.Itoa(settings.ReadTimeout))
		parsedQuery.Set("write_timeout", strconv.Itoa(settings.WriteTimeout))
		if len(settings.SSLMode) > 0 && len(settings.SSLRootCert) > 0 {
			log.D().Infof("ssl mode set to %s, sslrootcert detected", settings.SSLMode)
			parsedQuery.Set("sslmode", settings.SSLMode)
			parsedQuery.Set("sslrootcert", settings.SSLRootCert)
		}
		parsedUrl.RawQuery = parsedQuery.Encode()

		db, err := ps.ConnectFunc(postgresDriverName, parsedUrl.String())
		if err != nil {
			return fmt.Errorf("could not connect to PostgreSQL: %s", err)
		}
		ps.db = sqlx.NewDb(db, postgresDriverName)

		ps.state = &storageState{
			lastCheckTime:        time.Now(),
			mutex:                &sync.RWMutex{},
			db:                   ps.db,
			storageCheckInterval: time.Second * 5,
		}
		ps.layerOneEncryptionKey = []byte(settings.EncryptionKey)
		ps.db.SetMaxIdleConns(settings.MaxIdleConnections)
		ps.db.SetMaxOpenConns(settings.MaxOpenConnections)
		ps.pgDB = ps.db
		ps.queryBuilder = NewQueryBuilder(ps.pgDB)

		log.D().Debugf("Updating database schema using migrations from %s", settings.MigrationsURL)
		if err := ps.updateSchema(settings.MigrationsURL, postgresDriverName); err != nil {
			return fmt.Errorf("could not update database schema: %s", err)
		}
		ps.scheme = newScheme()
		ps.scheme.introduce(&Broker{})
		ps.scheme.introduce(&Platform{})
		ps.scheme.introduce(&ServiceOffering{})
		ps.scheme.introduce(&ServicePlan{})
		ps.scheme.introduce(&Visibility{})
		ps.scheme.introduce(&Notification{})
		ps.scheme.introduce(&Operation{})
		ps.scheme.introduce(&ServiceInstance{})
		ps.scheme.introduce(&ServiceBinding{})
		ps.scheme.introduce(&BrokerPlatformCredential{})
	}

	return nil
}

func (ps *Storage) Close() error {
	ps.mutex.Lock()
	defer ps.mutex.Unlock()
	if ps.db != nil {
		return ps.db.Close()
	}

	return nil
}

func (ps *Storage) checkOpen() {
	if ps.pgDB == nil {
		log.D().Panicln("Storage is not yet open")
	}
}

func (ps *Storage) updateSchema(migrationsURL, pgDriverName string) error {
	driver, err := migratepg.WithInstance(ps.db.DB, &migratepg.Config{})
	if err != nil {
		return err
	}
	m, err := migrate.NewWithDatabaseInstance(migrationsURL, pgDriverName, driver)
	if err != nil {
		return err
	}
	m.Log = migrateLogger{}
	err = m.Up()
	if err == migrate.ErrNoChange {
		log.D().Debug("Database schema already up to date")
		err = nil
	}
	return err
}

func (ps *Storage) PingContext(_ context.Context) error {
	ps.checkOpen()
	return ps.state.Get()
}

func (ps *Storage) Create(ctx context.Context, obj types.Object) (types.Object, error) {
	pgEntity, err := ps.scheme.convert(obj)
	if err != nil {
		return nil, err
	}
	result, err := ps.scheme.provide(obj.GetType())
	if err != nil {
		return nil, err
	}

	if err := create(ctx, ps.pgDB, pgEntity.TableName(), result, pgEntity); err != nil {
		return nil, err
	}

	createdObj, err := result.ToObject()
	if err != nil {
		return nil, fmt.Errorf("could not convert created pg entity to object: %s", err)
	}
	createdObj.SetLabels(obj.GetLabels())

	var pgLabels []PostgresLabel
	for key, values := range obj.GetLabels() {
		for _, labelValue := range values {
			pgLabel, err := ps.scheme.provideLabel(obj.GetType(), createdObj.GetID(), key, labelValue)
			if err != nil {
				return nil, err
			}
			pgLabels = append(pgLabels, pgLabel)
		}
	}

	if err = ps.createLabels(ctx, pgLabels); err != nil {
		return nil, err
	}

	return createdObj, nil
}

func (ps *Storage) createLabels(ctx context.Context, labels []PostgresLabel) error {
	if len(labels) == 0 {
		return nil
	}
	if err := validateLabels(labels); err != nil {
		return err
	}
	pgLabel := labels[0]
	setTagType := getDBTags(pgLabel, isAutoIncrementable)
	dbTags := make([]string, 0, len(setTagType))
	for _, tagType := range setTagType {
		dbTags = append(dbTags, tagType.Tag)
	}

	if len(dbTags) == 0 {
		return fmt.Errorf("%s insert: No fields to insert", pgLabel.LabelsTableName())
	}

	tableName := labels[0].(PostgresLabel).LabelsTableName()

	sqlQuery := fmt.Sprintf(
		"INSERT INTO %s (%s) VALUES(:%s) ON CONFLICT DO NOTHING;",
		tableName,
		strings.Join(dbTags, ", "),
		strings.Join(dbTags, ", :"),
	)

	// break into batches so the PostgreSQL limit of 65535 parameters is not exceeded
	const maxParams = 65535
	maxRows := maxParams / len(dbTags)
	for len(labels) > 0 {
		rows := min(len(labels), maxRows)
		log.C(ctx).Debugf("Executing query %s", sqlQuery)
		result, err := ps.pgDB.NamedExecContext(ctx, sqlQuery, labels[:rows])
		if err != nil {
			return checkIntegrityViolation(ctx, err)
		}
		rowsAffected, err := result.RowsAffected()
		if err != nil {
			log.C(ctx).Debugf("Could not get number of affected rows: %v", err)
		} else {
			log.C(ctx).Debugf("%d rows inserted", rowsAffected)
		}
		labels = labels[rows:]
	}
	return nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func (ps *Storage) deleteLabels(ctx context.Context, objectType types.ObjectType, entityID string, removedLabels types.Labels) error {
	if len(removedLabels) == 0 {
		return nil
	}
	emptyLabel, err := ps.scheme.provideLabel(objectType, entityID, "", "")
	if err != nil {
		return err
	}
	labelTableName := emptyLabel.LabelsTableName()
	referenceColumnName := emptyLabel.ReferenceColumn()
	isExpansionRequired := false
	segments := make([]string, 0)
	args := make([]interface{}, 0)
	for key, vals := range removedLabels {
		segment := fmt.Sprintf("(key=? AND %s=?", referenceColumnName)
		args = append(args, key, entityID)
		if len(vals) != 0 {
			isExpansionRequired = true
			segment += " AND val IN (?)"
			args = append(args, vals)
		}
		segment += ")"
		segments = append(segments, segment)
	}
	baseQuery := fmt.Sprintf("DELETE FROM %s WHERE %s", labelTableName, strings.Join(segments, " OR "))

	if isExpansionRequired {
		var err error
		baseQuery, args, err = sqlx.In(baseQuery, args...)
		if err != nil {
			return err
		}
	}

	baseQuery = ps.pgDB.Rebind(baseQuery)
	log.C(ctx).Debugf("Executing query %s", baseQuery)
	_, err = ps.pgDB.ExecContext(ctx, baseQuery, args...)
	if err != nil {
		return err
	}

	return nil
}

func (ps *Storage) Get(ctx context.Context, objectType types.ObjectType, criteria ...query.Criterion) (types.Object, error) {
	result, err := ps.List(ctx, objectType, criteria...)
	if err != nil {
		return nil, err
	}
	if result.Len() == 0 {
		return nil, util.ErrNotFoundInStorage
	}
	return result.ItemAt(0), nil
}

func (ps *Storage) QueryForList(ctx context.Context, objectType types.ObjectType, queryName storage.NamedQuery, queryParams map[string]interface{}) (types.ObjectList, error) {
	entity, err := ps.scheme.provide(objectType)
	if err != nil {
		return nil, err
	}

	rows, err := ps.queryBuilder.NewQuery(entity).Query(ctx, queryName, queryParams)
	if err != nil {
		return nil, err
	}

	defer func() {
		if rows == nil {
			return
		}
		if err := rows.Close(); err != nil {
			log.C(ctx).WithError(err).Error("Could not release connection when checking database")
		}
	}()
	return entity.RowsToList(rows)
}

func (ps *Storage) GetForUpdate(ctx context.Context, objectType types.ObjectType, criteria ...query.Criterion) (types.Object, error) {
	result, err := ps.list(ctx, objectType, true, true, criteria...)
	if err != nil {
		return nil, err
	}
	if result.Len() == 0 {
		return nil, util.ErrNotFoundInStorage
	}
	return result.ItemAt(0), nil
}

func (ps *Storage) List(ctx context.Context, objType types.ObjectType, criteria ...query.Criterion) (types.ObjectList, error) {
	return ps.list(ctx, objType, false, true, criteria...)
}

func (ps *Storage) ListNoLabels(ctx context.Context, objType types.ObjectType, criteria ...query.Criterion) (types.ObjectList, error) {
	return ps.list(ctx, objType, false, false, criteria...)
}

func (ps *Storage) list(ctx context.Context, objType types.ObjectType, forUpdate, withLabels bool, criteria ...query.Criterion) (types.ObjectList, error) {
	entity, err := ps.scheme.provide(objType)
	if err != nil {
		return nil, err
	}

	queryBuilder := ps.queryBuilder.NewQuery(entity).WithCriteria(criteria...)
	if forUpdate {
		queryBuilder = queryBuilder.WithLock()
	}

	var rows *sqlx.Rows
	if withLabels {
		rows, err = queryBuilder.List(ctx)
	} else {
		rows, err = queryBuilder.ListNoLabels(ctx)
	}
	if err != nil {
		return nil, err
	}

	defer func() {
		if rows == nil {
			return
		}
		if err := rows.Close(); err != nil {
			log.C(ctx).WithError(err).Error("Could not release connection when checking database")
		}
	}()
	return entity.RowsToList(rows)
}

func (ps *Storage) Count(ctx context.Context, objType types.ObjectType, criteria ...query.Criterion) (int, error) {
	entity, err := ps.scheme.provide(objType)
	if err != nil {
		return 0, err
	}
	return ps.queryBuilder.NewQuery(entity).WithCriteria(criteria...).Count(ctx)
}

func (ps *Storage) CountLabelValues(ctx context.Context, objType types.ObjectType, criteria ...query.Criterion) (int, error) {
	entity, err := ps.scheme.provide(objType)
	if err != nil {
		return 0, err
	}
	return ps.queryBuilder.NewQuery(entity).WithCriteria(criteria...).CountLabelValues(ctx)
}

func (ps *Storage) DeleteReturning(ctx context.Context, objType types.ObjectType, criteria ...query.Criterion) (types.ObjectList, error) {
	entity, err := ps.scheme.provide(objType)
	if err != nil {
		return nil, err
	}

	rows, err := ps.queryBuilder.NewQuery(entity).WithCriteria(criteria...).DeleteReturning(ctx, "*")
	defer closeRows(ctx, rows)
	if err != nil {
		pqError, ok := err.(*pq.Error)
		if ok && pqError.Code.Name() == foreignKeyViolation {
			entityName := objType.String()
			referenceEntityName := ps.scheme.entityToObjectTypeConverter[pqError.Table]

			return nil, &util.ErrForeignKeyViolation{
				Entity:          entityName,
				ReferenceEntity: referenceEntityName,
			}
		}

		return nil, err
	}
	objectList, err := entity.RowsToList(rows)
	if err != nil {
		return nil, err
	}
	if objectList.Len() < 1 {
		return nil, util.ErrNotFoundInStorage
	}
	return objectList, nil
}

func (ps *Storage) Delete(ctx context.Context, objType types.ObjectType, criteria ...query.Criterion) error {
	entity, err := ps.scheme.provide(objType)
	if err != nil {
		return err
	}

	result, err := ps.queryBuilder.NewQuery(entity).WithCriteria(criteria...).Delete(ctx)
	if err != nil {
		pqError, ok := err.(*pq.Error)
		if ok && pqError.Code.Name() == foreignKeyViolation {
			entityName := objType.String()
			referenceEntityName := ps.scheme.entityToObjectTypeConverter[pqError.Table]

			return &util.ErrForeignKeyViolation{
				Entity:          entityName,
				ReferenceEntity: referenceEntityName,
			}
		}
		return err
	}

	return checkRowsAffected(ctx, result)
}

func (ps *Storage) Update(ctx context.Context, obj types.Object, labelChanges types.LabelChanges, _ ...query.Criterion) (types.Object, error) {
	obj.SetUpdatedAt(time.Now().UTC())

	entity, err := ps.scheme.convert(obj)
	if err != nil {
		return nil, err
	}
	if err = update(ctx, ps.pgDB, entity.TableName(), entity); err != nil {
		return nil, err
	}
	if err = ps.updateLabels(ctx, obj.GetType(), entity.GetID(), labelChanges); err != nil {
		return nil, err
	}

	result, err := entity.ToObject()
	if err != nil {
		return nil, fmt.Errorf("could not convert updated pg entity for object: %s", err)
	}
	return result, nil
}

func (ps *Storage) UpdateLabels(ctx context.Context, objectType types.ObjectType, objectID string, labelChanges types.LabelChanges, _ ...query.Criterion) error {
	return ps.updateLabels(ctx, objectType, objectID, labelChanges)
}

func (ps *Storage) GetEntities() []storage.EntityMetadata {
	entities := make([]storage.EntityMetadata, 0)
	for entityTableName, entityName := range ps.scheme.entityToObjectTypeConverter {
		entity := storage.EntityMetadata{}
		entity.TableName = entityTableName
		entity.Name = entityName

		entities = append(entities, entity)
	}
	return entities
}

func (ps *Storage) updateLabels(ctx context.Context, objectType types.ObjectType, entityID string, updateActions []*types.LabelChange) error {
	_, addedLabels, removedLabels := query.ApplyLabelChangesToLabels(updateActions, types.Labels{})
	var pgAddedLabels []PostgresLabel
	for key, values := range addedLabels {
		for _, val := range values {
			pgLabel, err := ps.scheme.provideLabel(objectType, entityID, key, val)
			if err != nil {
				return err
			}
			pgAddedLabels = append(pgAddedLabels, pgLabel)
		}
	}

	if err := ps.createLabels(ctx, pgAddedLabels); err != nil {
		return err
	}

	if err := ps.deleteLabels(ctx, objectType, entityID, removedLabels); err != nil {
		return err
	}

	return nil
}

func (ps *Storage) InTransaction(ctx context.Context, f func(ctx context.Context, storage storage.Repository) error) error {
	ok := false
	tx, err := ps.db.Beginx()
	if err != nil {
		return err
	}
	defer func() {
		if !ok {
			if txError := tx.Rollback(); txError != nil {
				log.C(ctx).Error("Could not rollback transaction", txError)
			}
		}
	}()

	transactionalStorage := &Storage{
		pgDB:                  tx,
		db:                    ps.db,
		queryBuilder:          NewQueryBuilder(tx),
		scheme:                ps.scheme,
		layerOneEncryptionKey: ps.layerOneEncryptionKey,
	}

	if err = f(ctx, transactionalStorage); err != nil {
		return err
	}

	if err = tx.Commit(); err != nil {
		return err
	}
	ok = true
	return nil
}

type migrateLogger struct{}

func (migrateLogger) Printf(format string, v ...interface{}) {
	log.D().Debugf(format, v...)
}

func (migrateLogger) Verbose() bool {
	return true
}
