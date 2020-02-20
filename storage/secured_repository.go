package storage

import (
	"context"
	"crypto/rand"
	"fmt"
	"time"

	"github.com/Peripli/service-manager/pkg/log"

	"github.com/Peripli/service-manager/pkg/security"

	"github.com/Peripli/service-manager/pkg/query"
	"github.com/Peripli/service-manager/pkg/types"
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

// SecuringDecorator creates a TransactionalRepositoryDecorator that can be used to add encrypting/decrypting logic to a TransactionalRepository
func SecuringDecorator(ctx context.Context, encrypter security.Encrypter, keyStore KeyStore, locker Locker, checksumFunc func(data []byte) [32]byte) TransactionalRepositoryDecorator {
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

		return NewSecuredRepository(next, encrypter, encryptionKey, checksumFunc)
	}
}

//NewSecuredRepository creates a new TransactionalSecuredRepository using the specified encrypter and encryption key
func NewSecuredRepository(repository TransactionalRepository, encrypter security.Encrypter, key []byte, checksumFunc func(data []byte) [32]byte) (*TransactionalSecuredRepository, error) {
	encryptingRepository := &TransactionalSecuredRepository{
		securedRepository: &securedRepository{
			repository:    repository,
			encrypter:     encrypter,
			encryptionKey: key,
			checksumFunc:  checksumFunc,
		},
		repository: repository,
	}

	return encryptingRepository, nil
}

type securedRepository struct {
	repository Repository
	encrypter  security.Encrypter

	encryptionKey []byte
	checksumFunc  func(data []byte) [32]byte
}

//TransactionalSecuredRepository is a TransactionalRepository with that also encrypts credentials of Secured objects
// before storing in the database them and decrypts credentials of Secured objects when reading them from the database
// It also sets a checksum for secured objects and validates the checksum on secured objects when reading them from the database
type TransactionalSecuredRepository struct {
	*securedRepository

	repository TransactionalRepository
}

func (sr *securedRepository) Create(ctx context.Context, obj types.Object) (types.Object, error) {
	if err := sr.processModification(ctx, obj); err != nil {
		return nil, err
	}

	newObj, err := sr.repository.Create(ctx, obj)
	if err != nil {
		return nil, err
	}

	if err := sr.decrypt(ctx, newObj); err != nil {
		return nil, err
	}

	return newObj, nil
}

func (sr *securedRepository) Get(ctx context.Context, objectType types.ObjectType, criteria ...query.Criterion) (types.Object, error) {
	obj, err := sr.repository.Get(ctx, objectType, criteria...)
	if err != nil {
		return nil, err
	}

	if err := sr.decrypt(ctx, obj); err != nil {
		return nil, err
	}
	if err := sr.validateChecksum(obj); err != nil {
		return nil, err
	}
	return obj, nil
}

func (sr *securedRepository) List(ctx context.Context, objectType types.ObjectType, criteria ...query.Criterion) (types.ObjectList, error) {
	objList, err := sr.repository.List(ctx, objectType, criteria...)
	if err != nil {
		return nil, err
	}

	for i := 0; i < objList.Len(); i++ {
		item := objList.ItemAt(i)
		if err := sr.decrypt(ctx, item); err != nil {
			return nil, err
		}
		if err := sr.validateChecksum(item); err != nil {
			return nil, err
		}
	}

	return objList, nil
}

func (sr *securedRepository) Count(ctx context.Context, objectType types.ObjectType, criteria ...query.Criterion) (int, error) {
	return sr.repository.Count(ctx, objectType, criteria...)
}

func (sr *securedRepository) Update(ctx context.Context, obj types.Object, labelChanges query.LabelChanges, _ ...query.Criterion) (types.Object, error) {
	if err := sr.processModification(ctx, obj); err != nil {
		return nil, err
	}

	updatedObj, err := sr.repository.Update(ctx, obj, labelChanges)
	if err != nil {
		return nil, err
	}

	if err := sr.decrypt(ctx, updatedObj); err != nil {
		return nil, err
	}

	return updatedObj, nil
}

func (sr *securedRepository) DeleteReturning(ctx context.Context, objectType types.ObjectType, criteria ...query.Criterion) (types.ObjectList, error) {
	objList, err := sr.repository.DeleteReturning(ctx, objectType, criteria...)
	if err != nil {
		return nil, err
	}

	for i := 0; i < objList.Len(); i++ {
		if err := sr.decrypt(ctx, objList.ItemAt(i)); err != nil {
			return nil, err
		}
	}

	return objList, nil
}

func (sr *securedRepository) Delete(ctx context.Context, objectType types.ObjectType, criteria ...query.Criterion) error {
	if err := sr.repository.Delete(ctx, objectType, criteria...); err != nil {
		return err
	}

	return nil
}

func (sr *securedRepository) decrypt(ctx context.Context, obj types.Object) error {
	if securedObject, isSecured := obj.(types.Secured); isSecured {
		return securedObject.Decrypt(ctx, func(ctx context.Context, bytes []byte) ([]byte, error) {
			return sr.encrypter.Decrypt(ctx, bytes, sr.encryptionKey)
		})
	}
	return nil
}

func (sr *securedRepository) processModification(ctx context.Context, obj types.Object) error {
	if securedObject, isSecured := obj.(types.Secured); isSecured {
		securedObject.SetChecksum(sr.checksumFunc)
		return securedObject.Encrypt(ctx, func(ctx context.Context, bytes []byte) ([]byte, error) {
			return sr.encrypter.Encrypt(ctx, bytes, sr.encryptionKey)
		})
	}
	return nil
}

func (sr *securedRepository) validateChecksum(obj types.Object) error {
	if securedObject, isSecured := obj.(types.Secured); isSecured {
		if !securedObject.ValidateChecksum(sr.checksumFunc) {
			return fmt.Errorf("invalid checksum for %s with ID %s", obj.GetType(), obj.GetID())
		}
	}
	return nil
}

// InTransaction wraps repository passed in the transaction to also encypt/decrypt credentials
func (sr *TransactionalSecuredRepository) InTransaction(ctx context.Context, f func(ctx context.Context, storage Repository) error) error {
	return sr.repository.InTransaction(ctx, func(ctx context.Context, storage Repository) error {
		return f(ctx, &securedRepository{
			repository:    storage,
			encrypter:     sr.encrypter,
			encryptionKey: sr.encryptionKey,
			checksumFunc:  sr.checksumFunc,
		})
	})
}
