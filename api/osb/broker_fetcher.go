package osb

import (
	"context"

	"github.com/Peripli/service-manager/pkg/log"
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/pkg/util"
	"github.com/Peripli/service-manager/security"
	"github.com/Peripli/service-manager/storage"
)

// BrokerDetails provides handler for the Service Manager OSB business logic
type BrokerDetails struct {
	BrokerStorage storage.Broker
	Encrypter     security.Encrypter
}

var _ BrokerFetcher = &BrokerDetails{}

// FetchBroker obtains the broker coordinates (auth and URL)
func (bd *BrokerDetails) FetchBroker(ctx context.Context, brokerID string) (*types.Broker, error) {
	broker, err := bd.BrokerStorage.Get(ctx, brokerID)
	if err != nil {
		log.D().Debugf("FetchBroker with id %s not found in storage", brokerID)
		return nil, util.HandleStorageError(err, "broker", brokerID)
	}

	password := broker.Credentials.Basic.Password
	plaintextPassword, err := bd.Encrypter.Decrypt(ctx, []byte(password))
	if err != nil {
		return nil, err
	}

	broker.Credentials.Basic.Password = string(plaintextPassword)

	return broker, nil
}
