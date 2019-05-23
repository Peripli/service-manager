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

func NewEncryptingRepository(ctx context.Context, repository securedTransactionalRepository, encrypter security.Encrypter) (*EncryptingRepository, error) {
	encryptingRepository := &EncryptingRepository{
		repository: repository,
		encrypter:  encrypter,
	}

	if err := encryptingRepository.setupEncryptionKey(ctx); err != nil {
		return nil, err
	}

	return encryptingRepository, nil
}

type EncryptingRepository struct {
	repository securedTransactionalRepository
	encrypter  security.Encrypter

	encryptionKey []byte
}

func (er *EncryptingRepository) Create(ctx context.Context, obj types.Object) (types.Object, error) {
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

func (er *EncryptingRepository) Get(ctx context.Context, objectType types.ObjectType, id string) (types.Object, error) {
	obj, err := er.repository.Get(ctx, objectType, id)
	if err != nil {
		return nil, err
	}

	if err := er.transformCredentials(ctx, obj, er.encrypter.Decrypt); err != nil {
		return nil, err
	}

	return obj, nil
}

func (er *EncryptingRepository) List(ctx context.Context, objectType types.ObjectType, criteria ...query.Criterion) (types.ObjectList, error) {
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

func (er *EncryptingRepository) Update(ctx context.Context, obj types.Object, labelChanges ...*query.LabelChange) (types.Object, error) {
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

func (er *EncryptingRepository) Delete(ctx context.Context, objectType types.ObjectType, criteria ...query.Criterion) (types.ObjectList, error) {
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

// InTransaction wraps repository passed in the transaction to also encypt/decrypt credentials
func (er *EncryptingRepository) InTransaction(ctx context.Context, f func(ctx context.Context, storage Repository) error) error {
	return er.repository.InTransaction(ctx, func(ctx context.Context, storage Repository) error {
		return f(ctx, er)
	})
}

func (er *EncryptingRepository) transformCredentials(ctx context.Context, obj types.Object, transformationFunc func(context.Context, []byte, []byte) ([]byte, error)) error {
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

func (er *EncryptingRepository) setupEncryptionKey(ctx context.Context) error {
	ctx, cancelFunc := context.WithTimeout(ctx, 2*time.Second)
	defer cancelFunc()
	if err := er.repository.Lock(ctx); err != nil {
		return err
	}

	encryptionKey, err := er.repository.GetEncryptionKey(ctx, er.encrypter.Decrypt)
	if err != nil {
		return err
	}

	if len(encryptionKey) == 0 {
		logger := log.C(ctx)
		logger.Info("No encryption key is present. Generating new one...")
		newEncryptionKey := make([]byte, 32)
		if _, err = rand.Read(newEncryptionKey); err != nil {
			return fmt.Errorf("could not generate encryption key: %v", err)
		}

		if err = er.repository.SetEncryptionKey(ctx, newEncryptionKey, er.encrypter.Encrypt); err != nil {
			return err
		}
		logger.Info("Successfully generated new encryption key")
	}

	er.encryptionKey = encryptionKey

	return er.repository.Unlock(ctx)
}
