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

func EncryptingDecorator(ctx context.Context, encrypter security.Encrypter, keyStore KeyStore) RepositoryDecorator {
	return func(next TransactionalRepository) (TransactionalRepository, error) {
		ctx, cancelFunc := context.WithTimeout(ctx, 2*time.Second)
		defer cancelFunc()
		if err := keyStore.Lock(ctx); err != nil {
			return nil, err
		}

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

		if err := keyStore.Unlock(ctx); err != nil {
			return nil, err
		}

		return NewEncryptingRepository(next, encrypter, encryptionKey)
	}
}

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

type TransactionalEncryptingRepository struct {
	*encryptingRepository

	repository TransactionalRepository
}

func (er *encryptingRepository) Create(ctx context.Context, obj types.Object) (types.Object, error) {
	if err := er.transformCredentials(ctx, obj, er.encrypter.Encrypt); err != nil {
		return nil, err
	}

	newObj, err := er.repository.Create(ctx, obj)
	if err != nil {
		return nil, err
	}

	if err := er.transformCredentials(ctx, newObj, er.encrypter.Decrypt); err != nil {
		return nil, err
	}

	return newObj, nil
}

func (er *encryptingRepository) Get(ctx context.Context, objectType types.ObjectType, id string) (types.Object, error) {
	obj, err := er.repository.Get(ctx, objectType, id)
	if err != nil {
		return nil, err
	}

	if err := er.transformCredentials(ctx, obj, er.encrypter.Decrypt); err != nil {
		return nil, err
	}

	return obj, nil
}

func (er *encryptingRepository) List(ctx context.Context, objectType types.ObjectType, criteria ...query.Criterion) (types.ObjectList, error) {
	objList, err := er.repository.List(ctx, objectType, criteria...)
	if err != nil {
		return nil, err
	}

	for i := 0; i < objList.Len(); i++ {
		if err := er.transformCredentials(ctx, objList.ItemAt(i), er.encrypter.Decrypt); err != nil {
			return nil, err
		}
	}

	return objList, nil
}

func (er *encryptingRepository) Update(ctx context.Context, obj types.Object, labelChanges ...*query.LabelChange) (types.Object, error) {
	if err := er.transformCredentials(ctx, obj, er.encrypter.Encrypt); err != nil {
		return nil, err
	}

	updatedObj, err := er.repository.Update(ctx, obj, labelChanges...)
	if err != nil {
		return nil, err
	}

	if err := er.transformCredentials(ctx, updatedObj, er.encrypter.Decrypt); err != nil {
		return nil, err
	}

	return updatedObj, nil
}

func (er *encryptingRepository) Delete(ctx context.Context, objectType types.ObjectType, criteria ...query.Criterion) (types.ObjectList, error) {
	objList, err := er.repository.Delete(ctx, objectType, criteria...)
	if err != nil {
		return nil, err
	}

	for i := 0; i < objList.Len(); i++ {
		if err := er.transformCredentials(ctx, objList.ItemAt(i), er.encrypter.Decrypt); err != nil {
			return nil, err
		}
	}

	return objList, nil
}

func (er *encryptingRepository) transformCredentials(ctx context.Context, obj types.Object, transformationFunc func(context.Context, []byte, []byte) ([]byte, error)) error {
	securedObj, isSecured := obj.(types.Secured)
	if isSecured {
		credentials := securedObj.GetCredentials()
		if credentials != nil {
			transformedPassword, err := transformationFunc(ctx, []byte(credentials.Basic.Password), er.encryptionKey)
			if err != nil {
				return err
			}
			credentials.Basic.Password = string(transformedPassword)
			securedObj.SetCredentials(credentials)
		}
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
