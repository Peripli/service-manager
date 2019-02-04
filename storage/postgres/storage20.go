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
	"github.com/jmoiron/sqlx"
)

func init() {
	storage.Register(Storage, &postgresStorage2{})
}

type postgresStorage2 struct {
	pdDB          pgDB
	db            *sqlx.DB
	state         *storageState
	encryptionKey []byte
}

type Entity interface {
	TableName() string
	GetID() string
	PrimaryColumn() string
	DTOMapper
}

func (ps *postgresStorage2) Close() error {
	panic("implement me")
}

func (ps *postgresStorage2) Ping() error {
	panic("implement me")
}

func (ps *postgresStorage2) Create(ctx context.Context, obj types.Object) (string, error) {
	dto := ps.FromDTO(obj)
	id, err := create(ctx, ps.pdDB, dto.TableName(), dto)
	if err != nil {
		return "", err
	}
	return id, ps.createLabels(ctx, id, obj.GetType(), obj.GetLabels())
}

func (ps *postgresStorage2) createLabels(ctx context.Context, entityID string, objType types.ObjectType, labels types.Labels) error {
	e := ps.LabelsForType(objType)
	entities, err := e.FromDTO(entityID, labels)
	if err != nil {
		return err
	}
	if err := e.Validate(entities); err != nil {
		return err
	}
	for _, label := range entities {
		if _, err := create(ctx, ps.db, e.TableName(), label); err != nil {
			return err
		}
	}
	return nil
}

func (ps *postgresStorage2) Get(ctx context.Context, id string, objectType types.ObjectType) (types.Object, error) {
	byID := query.ByField(query.EqualsOperator, "id", id)

	dto := ps.EmptyEntityForType(objectType).ToDTO(nil)
	result := dto.EmptyList()
	if err := ps.List(ctx, result, byID); err != nil {
		return nil, err
	}
	if result.Len() == 0 {
		return nil, util.ErrNotFoundInStorage
	}
	return result.ItemAt(0), nil
}

func (ps *postgresStorage2) List(ctx context.Context, obj types.ObjectList, criteria ...query.Criterion) error {
	panic("implement me")
}

func (ps *postgresStorage2) Delete(ctx context.Context, objectType types.ObjectType, criteria ...query.Criterion) error {
	entityForType := ps.EmptyEntityForType(objectType)
	return deleteAllByFieldCriteria(ctx, ps.db, entityForType.TableName(), entityForType, criteria)
}

func (ps *postgresStorage2) Update(ctx context.Context, obj types.Object, labelChanges ...*query.LabelChange) (types.Object, error) {
	entity := ps.FromDTO(obj)
	if err := update(ctx, ps.db, entity.TableName(), entity); err != nil {
		return nil, err
	}
	if err := ps.updateLabels(ctx, entity.GetID(), obj.GetType(), labelChanges); err != nil {
		return nil, err
	}
	entityLabels := ps.LabelsForType(obj.GetType())
	byBrokerID := query.ByField(query.EqualsOperator, entityLabels.ReferenceColumn(), entity.GetID())
	if err := listByFieldCriteria(ctx, ps.db, entityLabels.TableName(), &entityLabels, []query.Criterion{byBrokerID}); err != nil {
		return nil, err
	}
	labels := entityLabels.ToDTO()
	result := entity.ToDTO(entity)
	return result.WithLabels(labels), nil
}

func (ps *postgresStorage2) updateLabels(ctx context.Context, entityID string, objType types.ObjectType, updateActions []*query.LabelChange) error {
	now := time.Now()
	newLabelFunc := func(labelID string, labelKey string, labelValue string) Labelable {
		return typeLabelEntity[objType].NewLabelable(labelID, labelKey, labelValue, entityID, &now, &now)
	}
	return updateLabelsAbstract(ctx, newLabelFunc, ps.db, entityID, updateActions)
}

func (ps *postgresStorage2) InTransaction(ctx context.Context, f func(ctx context.Context, storage storage.Warehouse) error) error {
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

	transactionalStorage := &postgresStorage2{
		pdDB: tx,
	}

	if err := f(ctx, transactionalStorage); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}
	ok = true
	return nil
}

func (ps *postgresStorage2) FromDTO(object types.Object) Entity {
	return dtos[object.GetType()].FromDTO(object)
}

func (ps *postgresStorage2) EmptyEntityForType(objectType types.ObjectType) Entity {
	return dtos[objectType].FromDTO(nil)
}

type EntityLabels interface {
	FromDTO(entityID string, labels types.Labels) ([]Label, error)
	Validate(entities []Label) error
	TableName() string
	ReferenceColumn() string
	ToDTO() types.Labels
}

func (ps *postgresStorage2) LabelsForType(objectType types.ObjectType) EntityLabels {
	return brokerLabels{}
}

type LabelableCreator interface {
	NewLabelable(id, key, val, entityID string, createdAt, updatedAt *time.Time) Labelable
}

type BrokerLabelProvider struct {
}

func (*BrokerLabelProvider) NewLabelable(id, key, val, entityID string, createdAt, updatedAt *time.Time) Labelable {
	return &BrokerLabel{
		ID:        toNullString(id),
		Key:       toNullString(key),
		Val:       toNullString(val),
		BrokerID:  toNullString(entityID),
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
	}
}

var typeLabelEntity = map[types.ObjectType]LabelableCreator{types.BrokerType: &BrokerLabelProvider{}}

func (ps *postgresStorage2) Open(options *storage.Settings) error {
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
		ps.pdDB = ps.db
		ps.state = &storageState{
			lastCheckTime:        time.Now(),
			mutex:                &sync.RWMutex{},
			db:                   ps.pdDB,
			storageCheckInterval: time.Second * 5,
		}
		ps.encryptionKey = []byte(options.EncryptionKey)
		log.D().Debugf("Updating database schema using migrations from %s", options.MigrationsURL)
		if err := ps.updateSchema(options.MigrationsURL); err != nil {
			log.D().Panicln("Could not update database schema:", err)
		}
	}
	return err
}

func (ps *postgresStorage2) updateSchema(migrationsURL string) error {
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

func (ps *postgresStorage2) checkOpen() {
	if ps.db == nil {
		log.D().Panicln("Repository is not yet Open")
	}
}
