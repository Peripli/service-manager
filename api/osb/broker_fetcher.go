package osb

import (
	"context"

	"github.com/Peripli/service-manager/pkg/log"
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/pkg/util"
	"github.com/Peripli/service-manager/security"
	"github.com/Peripli/service-manager/storage"
)

// StorageBrokerFetcher provides logic for fetching the broker coordinates from the storage
type StorageBrokerFetcher struct {
	BrokerStorage storage.Broker
	Encrypter     security.Encrypter
}

var _ BrokerFetcher = &StorageBrokerFetcher{}

// FetchBroker obtains the broker coordinates (auth and URL)
func (sbf *StorageBrokerFetcher) FetchBroker(ctx context.Context, brokerID string) (*types.Broker, error) {
	broker, err := sbf.BrokerStorage.Get(ctx, brokerID)
	if err != nil {
		log.C(ctx).Debugf("FetchBroker with id %s not found in storage", brokerID)
		return nil, util.HandleStorageError(err, "broker", brokerID)
	}

	password := broker.Credentials.Basic.Password
	plaintextPassword, err := sbf.Encrypter.Decrypt(ctx, []byte(password))
	if err != nil {
		return nil, err
	}

	broker.Credentials.Basic.Password = string(plaintextPassword)

	return broker, nil
}
