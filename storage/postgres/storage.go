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
)

const Storage = "postgres"

func init() {
	storage.Register(Storage, &postgresStorage{})
}

type postgresStorage struct {
	pdDB          pgDB
	db            *sqlx.DB
	state         *storageState
	encryptionKey []byte
	scheme        *storage.Scheme
}

func (ps *postgresStorage) Credentials() storage.Credentials {
	ps.checkOpen()
	return &credentialStorage{db: ps.db}
}

func (ps *postgresStorage) Security() storage.Security {
	ps.checkOpen()
	return &securityStorage{ps.db, ps.encryptionKey, false, &sync.Mutex{}}
}

func (ps *postgresStorage) ServiceOffering() storage.ServiceOffering {
	ps.checkOpen()
	return &serviceOfferingStorage{db: ps.db}
}

func (ps *postgresStorage) Open(options *storage.Settings, scheme *storage.Scheme) error {
	var err error
	if err = options.Validate(); err != nil {
		return err
	}
	if len(options.MigrationsURL) == 0 {
		return fmt.Errorf("validate Settings: StorageMigrationsURL missing")
	}
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
		ps.scheme = scheme
		InstallBroker(ps.scheme)
		InstallServiceOffering(ps.scheme)
	}
	return err
}

func (ps *postgresStorage) Close() error {
	ps.checkOpen()
	return ps.db.Close()
}

func (ps *postgresStorage) checkOpen() {
	if ps.db == nil {
		log.D().Panicln("Repository is not yet Open")
	}
}

func (ps *postgresStorage) updateSchema(migrationsURL string) error {
	driver, err := migratepg.WithInstance(ps.db.DB, &migratepg.Config{})
	if err != nil {
		return err
	}
	m, err := migrate.NewWithDatabaseInstance(migrationsURL, "postgres", driver)
	if err != nil {
		return err
	}
	err = m.Up()
	if err == migrate.ErrNoChange {
		log.D().Debug("Database schema already up to date")
		err = nil
	}
	return err
}

func (ps *postgresStorage) Ping() error {
	ps.checkOpen()
	return ps.state.Get()
}

func (ps *postgresStorage) provide(objectType types.ObjectType) (Entity, error) {
	entity, ok := ps.scheme.Provide(objectType)
	if !ok {
		return nil, fmt.Errorf("object of type %s is not supported by the storage", objectType)
	}
	pgEntity, ok := entity.(Entity)
	if !ok {
		return nil, fmt.Errorf("registered storage entity for type %s is not compatible with postgres storage", objectType)
	}
	return pgEntity, nil
}

func (ps *postgresStorage) Create(ctx context.Context, obj types.Object) (string, error) {
	e, ok := ps.scheme.ObjectToEntity(obj)
	if !ok {
		return "", fmt.Errorf("object of type %s is not introduced to the storage", obj.GetType())
	}
	pgEntity := e.(Entity)
	id, err := create(ctx, ps.pdDB, pgEntity.TableName(), pgEntity)
	if err != nil {
		return "", err
	}
	labels, err := ps.scheme.TypeLabelsToEntity(obj.GetID(), obj.GetType(), obj.GetLabels())
	if err != nil {
		return "", err
	}
	if err = ps.createLabels(ctx, id, labels); err != nil {
		return "", err
	}
	return id, nil
}

func (ps *postgresStorage) createLabels(ctx context.Context, entityID string, labels []storage.Label) error {
	if err := validateLabels(labels); err != nil {
		return err
	}
	for _, label := range labels {
		pgLabel := label.(LabelEntity)
		if _, err := create(ctx, ps.db, pgLabel.LabelsTableName(), pgLabel); err != nil {
			return err
		}
	}
	return nil
}

func (ps *postgresStorage) Get(ctx context.Context, id string, objType types.ObjectType) (types.Object, error) {
	entity, err := ps.provide(objType)
	if err != nil {
		return nil, err
	}
	primaryColumn := entity.PrimaryColumn()
	byPrimaryColumn := query.ByField(query.EqualsOperator, primaryColumn, id)

	result, err := ps.List(ctx, objType, byPrimaryColumn)
	if err != nil {
		return nil, err
	}
	if result.Len() == 0 {
		return nil, util.ErrNotFoundInStorage
	}
	return result.ItemAt(0), nil
}

func (ps *postgresStorage) List(ctx context.Context, objType types.ObjectType, criteria ...query.Criterion) (types.ObjectList, error) {
	entity, err := ps.provide(objType)
	if err != nil {
		return nil, err
	}
	var rows *sqlx.Rows
	labelsInfo := entity.LabelEntity()
	rows, err = listWithLabelsByCriteria(ctx, ps.db, entity, labelsInfo, entity.TableName(), criteria)
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

func (ps *postgresStorage) Delete(ctx context.Context, objType types.ObjectType, criteria ...query.Criterion) (types.ObjectList, error) {
	entity, err := ps.provide(objType)
	if err != nil {
		return nil, err
	}
	rows, err := deleteAllByFieldCriteria(ctx, ps.db, entity.TableName(), entity, criteria)
	if err != nil {
		return nil, err
	}
	deletedObjects, err := entity.RowsToList(rows)
	if err != nil {
		return nil, err
	}
	return deletedObjects, nil
}

func (ps *postgresStorage) Update(ctx context.Context, obj types.Object, labelChanges ...*query.LabelChange) (types.Object, error) {
	e, ok := ps.scheme.ObjectToEntity(obj)
	if !ok {
		return nil, fmt.Errorf("not supported")
	}
	entity := e.(Entity)
	var err error
	if err = update(ctx, ps.db, entity.TableName(), entity); err != nil {
		return nil, err
	}
	if err = ps.updateLabels(ctx, entity.GetID(), obj.GetType(), labelChanges); err != nil {
		return nil, err
	}
	labelsInfo := entity.LabelEntity()
	typesLabels := obj.GetLabels()
	labels, err := ps.scheme.TypeLabelsToEntity(obj.GetID(), obj.GetType(), typesLabels)

	byEntityID := query.ByField(query.EqualsOperator, labelsInfo.ReferenceColumn(), entity.GetID())
	if err = listByFieldCriteria(ctx, ps.db, labelsInfo.LabelsTableName(), labels, []query.Criterion{byEntityID}); err != nil {
		return nil, err
	}
	typeLabels := ps.scheme.StorageLabelsToType(labels)
	result, ok := ps.scheme.EntityToObject(entity)
	result.SetLabels(typeLabels)
	return result, nil
}

func (ps *postgresStorage) updateLabels(ctx context.Context, entityID string, objType types.ObjectType, updateActions []*query.LabelChange) error {
	entity, err := ps.provide(objType)
	if err != nil {
		return err
	}
	newLabelFunc := func(labelID string, labelKey string, labelValue string) LabelEntity {
		return entity.LabelEntity().New(labelID, labelKey, labelValue, entityID).(LabelEntity)
	}
	return updateLabelsAbstract(ctx, newLabelFunc, ps.db, entityID, updateActions)
}

func (ps *postgresStorage) InTransaction(ctx context.Context, f func(ctx context.Context, storage storage.Warehouse) error) error {
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

	transactionalStorage := &postgresStorage{
		pdDB:   tx,
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
