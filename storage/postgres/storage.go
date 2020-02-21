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
	postgresDriverName  = "postgres"
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
		sslModeParam := ""
		if settings.SkipSSLValidation {
			sslModeParam = "?sslmode=disable"
		}
		db, err := ps.ConnectFunc(postgresDriverName, settings.URI+sslModeParam)
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

func (ps *Storage) PingContext(ctx context.Context) error {
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

	createdObj := result.ToObject()
	createdObj.SetLabels(obj.GetLabels())

	var labels []storage.Label
	if labels, err = pgEntity.BuildLabels(createdObj.GetLabels(), pgEntity.NewLabel); err != nil {
		return nil, err
	}

	if err = ps.createLabels(ctx, createdObj.GetID(), labels); err != nil {
		return nil, err
	}

	return createdObj, nil
}

func (ps *Storage) createLabels(ctx context.Context, entityID string, labels []storage.Label) error {
	if err := validateLabels(labels); err != nil {
		return err
	}

	for _, label := range labels {
		pgLabel, ok := label.(PostgresLabel)
		if !ok {
			return fmt.Errorf("postgres storage requires labels to implement LabelEntity, got %T", label)
		}
		if err := create(ctx, ps.pgDB, pgLabel.LabelsTableName(), pgLabel, pgLabel); err != nil {
			return err
		}
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

func (ps *Storage) List(ctx context.Context, objType types.ObjectType, criteria ...query.Criterion) (types.ObjectList, error) {
	entity, err := ps.scheme.provide(objType)
	if err != nil {
		return nil, err
	}

	rows, err := ps.queryBuilder.NewQuery(entity).WithCriteria(criteria...).WithLock().List(ctx)
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
	if err != nil {
		return nil, err
	}
	return entity.RowsToList(rows)
}

func (ps *Storage) Count(ctx context.Context, objType types.ObjectType, criteria ...query.Criterion) (int, error) {
	entity, err := ps.scheme.provide(objType)
	if err != nil {
		return 0, err
	}
	return ps.queryBuilder.NewQuery(entity).WithCriteria(criteria...).WithLock().Count(ctx)
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

func (ps *Storage) Update(ctx context.Context, obj types.Object, labelChanges query.LabelChanges, _ ...query.Criterion) (types.Object, error) {
	obj.SetUpdatedAt(time.Now().UTC())
	entity, err := ps.scheme.convert(obj)
	if err != nil {
		return nil, err
	}
	if err = update(ctx, ps.pgDB, entity.TableName(), entity); err != nil {
		return nil, err
	}
	if err = ps.updateLabels(ctx, entity.GetID(), entity, labelChanges); err != nil {
		return nil, err
	}

	result := entity.ToObject()
	return result, nil
}

func (ps *Storage) updateLabels(ctx context.Context, entityID string, entity PostgresEntity, updateActions []*query.LabelChange) error {
	newLabelFunc := func(labelID string, labelKey string, labelValue string) (PostgresLabel, error) {
		label := entity.NewLabel(labelID, labelKey, labelValue)
		pgLabel, ok := label.(PostgresLabel)
		if !ok {
			return nil, fmt.Errorf("postgres storage requires labels to implement LabelEntity, got %T", label)
		}
		return pgLabel, nil
	}
	return updateLabelsAbstract(ctx, newLabelFunc, ps.pgDB, entityID, updateActions)
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
