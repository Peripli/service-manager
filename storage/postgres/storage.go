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
	"sync"
	"time"

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

const Storage = "postgres"

type PostgresStorage struct {
	pgDB          PGDB
	db            *sqlx.DB
	state         *storageState
	encryptionKey []byte
	scheme        *storage.Scheme

	mutex sync.Mutex
}

func (ps *PostgresStorage) Introduce(entity storage.Entity) {
	ps.scheme.Introduce(entity)
}

func (ps *PostgresStorage) Credentials() storage.Credentials {
	ps.checkOpen()
	return &credentialStorage{db: ps.pgDB}
}

func (ps *PostgresStorage) Security() storage.Security {
	ps.checkOpen()
	return &securityStorage{ps.pgDB, ps.encryptionKey, false, &sync.Mutex{}}
}

func (ps *PostgresStorage) Open(options *storage.Settings, scheme *storage.Scheme) error {
	var err error
	if err = options.Validate(); err != nil {
		return err
	}
	if len(options.MigrationsURL) == 0 {
		return fmt.Errorf("validate Settings: StorageMigrationsURL missing")
	}
	ps.mutex.Lock()
	defer ps.mutex.Unlock()
	if ps.db == nil {
		sslModeParam := ""
		if options.SkipSSLValidation {
			sslModeParam = "?sslmode=disable"
		}
		ps.db, err = sqlx.Connect(Storage, options.URI+sslModeParam)
		if err != nil {
			log.D().Panicln("Could not connect to PostgreSQL:", err)
		}
		ps.state = &storageState{
			lastCheckTime:        time.Now(),
			mutex:                &sync.RWMutex{},
			db:                   ps.db,
			storageCheckInterval: time.Second * 5,
		}
		ps.encryptionKey = []byte(options.EncryptionKey)
		log.D().Debugf("Updating database schema using migrations from %s", options.MigrationsURL)
		if err := ps.updateSchema(options.MigrationsURL); err != nil {
			log.D().Panicln("Could not update database schema:", err)
		}
		ps.pgDB = ps.db
		ps.scheme = scheme
		ps.scheme.Introduce(&Broker{})
		ps.scheme.Introduce(&Platform{})
		ps.scheme.Introduce(&ServiceOffering{})
		ps.scheme.Introduce(&ServicePlan{})
		ps.scheme.Introduce(&Visibility{})
	}
	return err
}

func (ps *PostgresStorage) Close() error {
	ps.checkOpen()
	ps.mutex.Lock()
	defer ps.mutex.Unlock()
	defer func() {
		ps.db = nil
	}()
	return ps.db.Close()
}

func (ps *PostgresStorage) checkOpen() {
	if ps.pgDB == nil {
		log.D().Panicln("TransactionalRepository is not yet Open")
	}
}

func (ps *PostgresStorage) updateSchema(migrationsURL string) error {
	driver, err := migratepg.WithInstance(ps.db.DB, &migratepg.Config{})
	if err != nil {
		return err
	}
	m, err := migrate.NewWithDatabaseInstance(migrationsURL, "postgres", driver)
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

func (ps *PostgresStorage) Ping() error {
	ps.checkOpen()
	return ps.state.Get()
}

func (ps *PostgresStorage) provide(objectType types.ObjectType) (PostgresEntity, error) {
	entity, ok := ps.scheme.Provide(objectType)
	if !ok {
		return nil, fmt.Errorf("object of type %s is not supported by the storage", objectType)
	}
	return toPgEntity(entity)
}

func (ps *PostgresStorage) convert(object types.Object) (PostgresEntity, error) {
	entity, ok := ps.scheme.ObjectToEntity(object)
	if !ok {
		return nil, fmt.Errorf("object of type %s is not introduced to the storage", object.GetType())
	}
	return toPgEntity(entity)
}

func toPgEntity(entity storage.Entity) (PostgresEntity, error) {
	pgEntity, ok := entity.(PostgresEntity)
	if !ok {
		return nil, fmt.Errorf("postgres storage requires entities to implement postgres.Entity, got %T", entity)
	}
	return pgEntity, nil
}

func (ps *PostgresStorage) Create(ctx context.Context, obj types.Object) (string, error) {
	pgEntity, err := ps.convert(obj)
	if err != nil {
		return "", err
	}
	var id string
	if id, err = create(ctx, ps.pgDB, pgEntity.TableName(), pgEntity); err != nil {
		return "", err
	}
	var labels []storage.Label
	if labels, err = pgEntity.BuildLabels(obj.GetLabels(), pgEntity.NewLabel); err != nil {
		return "", err
	}
	if err = ps.createLabels(ctx, id, labels); err != nil {
		return "", err
	}
	return id, nil
}

func (ps *PostgresStorage) createLabels(ctx context.Context, entityID string, labels []storage.Label) error {
	if err := validateLabels(labels); err != nil {
		return err
	}
	for _, label := range labels {
		pgLabel, ok := label.(PostgresLabel)
		if !ok {
			return fmt.Errorf("postgres storage requires labels to implement postgres.LabelEntity, got %T", label)
		}
		if _, err := create(ctx, ps.pgDB, pgLabel.LabelsTableName(), pgLabel); err != nil {
			return err
		}
	}
	return nil
}

func (ps *PostgresStorage) Get(ctx context.Context, objectType types.ObjectType, id string) (types.Object, error) {
	byPrimaryColumn := query.ByField(query.EqualsOperator, "id", id)

	result, err := ps.List(ctx, objectType, byPrimaryColumn)
	if err != nil {
		return nil, err
	}
	if result.Len() == 0 {
		return nil, util.ErrNotFoundInStorage
	}
	return result.ItemAt(0), nil
}

func (ps *PostgresStorage) List(ctx context.Context, objType types.ObjectType, criteria ...query.Criterion) (types.ObjectList, error) {
	entity, err := ps.provide(objType)
	if err != nil {
		return nil, err
	}
	rows, err := listWithLabelsByCriteria(ctx, ps.pgDB, entity, entity.LabelEntity(), entity.TableName(), criteria)
	defer func() {
		if rows == nil {
			return
		}
		if err := rows.Close(); err != nil {
			log.C(ctx).Errorf("Could not release connection when checking database. Error: %s", err)
		}
	}()
	if err != nil {
		return nil, err
	}
	return entity.RowsToList(rows)
}

func (ps *PostgresStorage) Delete(ctx context.Context, objType types.ObjectType, criteria ...query.Criterion) (types.ObjectList, error) {
	entity, err := ps.provide(objType)
	if err != nil {
		return nil, err
	}
	rows, err := deleteAllByFieldCriteria(ctx, ps.pgDB, entity.TableName(), entity, criteria)
	defer func() {
		if rows == nil {
			return
		}
		if err := rows.Close(); err != nil {
			log.C(ctx).Errorf("Could not release connection when checking database. Error: %s", err)
		}
	}()
	if err != nil {
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

func (ps *PostgresStorage) Update(ctx context.Context, obj types.Object, labelChanges ...*query.LabelChange) (types.Object, error) {
	entity, err := ps.convert(obj)
	if err != nil {
		return nil, err
	}
	if err = update(ctx, ps.pgDB, entity.TableName(), entity); err != nil {
		return nil, err
	}
	if err = ps.updateLabels(ctx, entity.GetID(), entity, labelChanges); err != nil {
		return nil, err
	}
	labelsInfo := entity.LabelEntity()
	byEntityID := query.ByField(query.EqualsOperator, labelsInfo.ReferenceColumn(), entity.GetID())
	rows, err := listByFieldCriteria(ctx, ps.pgDB, labelsInfo.LabelsTableName(), []query.Criterion{byEntityID})
	if err != nil {
		return nil, err
	}
	labels, err := labelsRowsToList(rows, func() PostgresLabel { return entity.LabelEntity() })
	if err != nil {
		return nil, err
	}
	typeLabels := ps.scheme.StorageLabelsToType(labels)
	result := entity.ToObject()
	result.SetLabels(typeLabels)
	return result, nil
}

func (ps *PostgresStorage) updateLabels(ctx context.Context, entityID string, entity PostgresEntity, updateActions []*query.LabelChange) error {
	newLabelFunc := func(labelID string, labelKey string, labelValue string) (PostgresLabel, error) {
		label := entity.NewLabel(labelID, labelKey, labelValue)
		pgLabel, ok := label.(PostgresLabel)
		if !ok {
			return nil, fmt.Errorf("postgres storage requires labels to implement postgres.LabelEntity, got %T", label)
		}
		return pgLabel, nil
	}
	return updateLabelsAbstract(ctx, newLabelFunc, ps.pgDB, entityID, updateActions)
}

func (ps *PostgresStorage) InTransaction(ctx context.Context, f func(ctx context.Context, storage storage.Repository) error) error {
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

	transactionalStorage := &PostgresStorage{
		pgDB:   tx,
		db:     ps.db,
		scheme: ps.scheme,
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
