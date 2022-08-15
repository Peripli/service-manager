package storage

import (
	"context"
	"crypto/rand"
	"fmt"
	"time"

	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/log"

	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/security"

	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/query"
	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/types"
)

// LockerCreatorFunc is a function building a storage.Locker with a specific advisory index
type LockerCreatorFunc func(advisoryIndex int) Locker

// Locker provides basic Lock/Unlock functionality
type Locker interface {
	// Lock locks the storage so that only one process can acquire it. Returns an error if the process has already acquired the lock, but waits if the lock is acquired by different connection
	Lock(ctx context.Context) error

	// TryLock tries to lock the storage, if it is already locked it returns an error, that it is already locked and does not wait
	TryLock(ctx context.Context) error

	// Unlock releases the acquired lock.
	Unlock(ctx context.Context) error
}

// KeyStore interface for encryption key operations
type KeyStore interface {
	// GetEncryptionKey returns the encryption key from the storage after applying the specified transformation function
	GetEncryptionKey(ctx context.Context, transformationFunc func(context.Context, []byte, []byte) ([]byte, error)) ([]byte, error)

	// SetEncryptionKey sets the provided encryption key in the KeyStore after applying the specified transformation function
	SetEncryptionKey(ctx context.Context, key []byte, transformationFunc func(context.Context, []byte, []byte) ([]byte, error)) error
}

// EncryptingDecorator creates a TransactionalRepositoryDecorator that can be used to add encrypting/decrypting logic to a TransactionalRepository
func EncryptingDecorator(ctx context.Context, encrypter security.Encrypter, keyStore KeyStore, locker Locker) TransactionalRepositoryDecorator {
	return func(next TransactionalRepository) (TransactionalRepository, error) {
		ctx, cancelFunc := context.WithTimeout(ctx, 2*time.Second)
		defer cancelFunc()

		if err := locker.Lock(ctx); err != nil {
			return nil, err
		}
		defer func() {
			if err := locker.Unlock(ctx); err != nil {
				log.C(ctx).WithError(err).Error("error while unlocking keystore")
			}
		}()

		encryptionKey, err := keyStore.GetEncryptionKey(ctx, encrypter.Decrypt)
		if err != nil {
			return nil, err
		}

		if len(encryptionKey) == 0 {
			logger := log.C(ctx)
			logger.Info("No encryption key is present. Generating new one...")
			newEncryptionKey := make([]byte, 32)
			if _, err = rand.Read(newEncryptionKey); err != nil {
				return nil, fmt.Errorf("could not generate encryption key: %v", err)
			}

			encryptionKey = newEncryptionKey
			if err = keyStore.SetEncryptionKey(ctx, newEncryptionKey, encrypter.Encrypt); err != nil {
				return nil, err
			}
			logger.Info("Successfully generated new encryption key")
		}

		return NewEncryptingRepository(next, encrypter, encryptionKey)
	}
}

//NewEncryptingRepository creates a new TransactionalEncryptingRepository using the specified encrypter and encryption key
func NewEncryptingRepository(repository TransactionalRepository, encrypter security.Encrypter, key []byte) (*TransactionalEncryptingRepository, error) {
	encryptingRepository := &TransactionalEncryptingRepository{
		encryptingRepository: &encryptingRepository{
			repository:    repository,
			encrypter:     encrypter,
			encryptionKey: key,
		},
		repository: repository,
	}

	return encryptingRepository, nil
}

type encryptingRepository struct {
	repository Repository
	encrypter  security.Encrypter

	encryptionKey []byte
}

//TransactionalEncryptingRepository is a TransactionalRepository with that also encrypts credentials of Secured objects
// before storing in the database them and decrypts credentials of Secured objects when reading them from the database
type TransactionalEncryptingRepository struct {
	*encryptingRepository

	repository TransactionalRepository
}

func (er *encryptingRepository) QueryForList(ctx context.Context, objectType types.ObjectType, queryName NamedQuery, queryParams map[string]interface{}) (types.ObjectList, error) {
	objList, err := er.repository.QueryForList(ctx, objectType, queryName, queryParams)
	if err != nil {
		return nil, err
	}
	for i := 0; i < objList.Len(); i++ {
		if err := er.decrypt(ctx, objList.ItemAt(i)); err != nil {
			return nil, err
		}
	}
	return objList, nil
}

func (er *encryptingRepository) Create(ctx context.Context, obj types.Object) (types.Object, error) {
	if err := er.encrypt(ctx, obj); err != nil {
		return nil, err
	}

	newObj, err := er.repository.Create(ctx, obj)
	if err != nil {
		return nil, err
	}

	if err := er.decrypt(ctx, newObj); err != nil {
		return nil, err
	}

	return newObj, nil
}

func (er *encryptingRepository) Get(ctx context.Context, objectType types.ObjectType, criteria ...query.Criterion) (types.Object, error) {
	obj, err := er.repository.Get(ctx, objectType, criteria...)
	if err != nil {
		return nil, err
	}

	if err := er.decrypt(ctx, obj); err != nil {
		return nil, err
	}

	return obj, nil
}

func (er *encryptingRepository) GetForUpdate(ctx context.Context, objectType types.ObjectType, criteria ...query.Criterion) (types.Object, error) {
	obj, err := er.repository.Get(ctx, objectType, criteria...)
	if err != nil {
		return nil, err
	}

	if err := er.decrypt(ctx, obj); err != nil {
		return nil, err
	}

	return obj, nil
}

func (er *encryptingRepository) List(ctx context.Context, objectType types.ObjectType, criteria ...query.Criterion) (types.ObjectList, error) {
	return er.list(ctx, objectType, true, criteria...)
}

func (er *encryptingRepository) ListNoLabels(ctx context.Context, objectType types.ObjectType, criteria ...query.Criterion) (types.ObjectList, error) {
	return er.list(ctx, objectType, false, criteria...)
}

func (er *encryptingRepository) list(ctx context.Context, objectType types.ObjectType, withLabels bool, criteria ...query.Criterion) (types.ObjectList, error) {
	var (
		objList types.ObjectList
		err     error
	)
	if withLabels {
		objList, err = er.repository.List(ctx, objectType, criteria...)
	} else {
		objList, err = er.repository.ListNoLabels(ctx, objectType, criteria...)
	}
	if err != nil {
		return nil, err
	}

	for i := 0; i < objList.Len(); i++ {
		if err := er.decrypt(ctx, objList.ItemAt(i)); err != nil {
			return nil, err
		}
	}

	return objList, nil
}

func (er *encryptingRepository) Count(ctx context.Context, objectType types.ObjectType, criteria ...query.Criterion) (int, error) {
	return er.repository.Count(ctx, objectType, criteria...)
}

func (er *encryptingRepository) CountLabelValues(ctx context.Context, objectType types.ObjectType, criteria ...query.Criterion) (int, error) {
	return er.repository.CountLabelValues(ctx, objectType, criteria...)
}

func (er *encryptingRepository) Update(ctx context.Context, obj types.Object, labelChanges types.LabelChanges, _ ...query.Criterion) (types.Object, error) {
	if err := er.encrypt(ctx, obj); err != nil {
		return nil, err
	}

	updatedObj, err := er.repository.Update(ctx, obj, labelChanges)
	if err != nil {
		return nil, err
	}

	if err := er.decrypt(ctx, updatedObj); err != nil {
		return nil, err
	}

	return updatedObj, nil
}

func (cr *encryptingRepository) UpdateLabels(ctx context.Context, objectType types.ObjectType, objectID string, labelChanges types.LabelChanges, criteria ...query.Criterion) error {
	return cr.repository.UpdateLabels(ctx, objectType, objectID, labelChanges, criteria...)
}

func (er *encryptingRepository) DeleteReturning(ctx context.Context, objectType types.ObjectType, criteria ...query.Criterion) (types.ObjectList, error) {
	objList, err := er.repository.DeleteReturning(ctx, objectType, criteria...)
	if err != nil {
		return nil, err
	}

	for i := 0; i < objList.Len(); i++ {
		if err := er.decrypt(ctx, objList.ItemAt(i)); err != nil {
			return nil, err
		}
	}

	return objList, nil
}

func (er *encryptingRepository) Delete(ctx context.Context, objectType types.ObjectType, criteria ...query.Criterion) error {
	if err := er.repository.Delete(ctx, objectType, criteria...); err != nil {
		return err
	}

	return nil
}

func (er *encryptingRepository) encrypt(ctx context.Context, obj types.Object) error {
	if securedObject, isSecured := obj.(types.Secured); isSecured {
		return securedObject.Encrypt(ctx, func(ctx context.Context, bytes []byte) ([]byte, error) {
			return er.encrypter.Encrypt(ctx, bytes, er.encryptionKey)
		})
	}
	return nil
}

func (er *encryptingRepository) decrypt(ctx context.Context, obj types.Object) error {
	if securedObject, isSecured := obj.(types.Secured); isSecured {
		return securedObject.Decrypt(ctx, func(ctx context.Context, bytes []byte) ([]byte, error) {
			return er.encrypter.Decrypt(ctx, bytes, er.encryptionKey)
		})
	}
	return nil
}

// InTransaction wraps repository passed in the transaction to also encypt/decrypt credentials
func (er *TransactionalEncryptingRepository) InTransaction(ctx context.Context, f func(ctx context.Context, storage Repository) error) error {
	return er.repository.InTransaction(ctx, func(ctx context.Context, storage Repository) error {
		return f(ctx, &encryptingRepository{
			repository:    storage,
			encrypter:     er.encrypter,
			encryptionKey: er.encryptionKey,
		})
	})
}

func (er *encryptingRepository) GetEntities() []EntityMetadata {
	return er.repository.GetEntities()
}
