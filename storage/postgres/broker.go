package postgres

import (
	"github.com/Peripli/service-manager/types"
	"github.com/jmoiron/sqlx"
	"context"
)

type brokerStorage struct {
	db *sqlx.DB
}

func (storage *brokerStorage) Create(ctx context.Context, broker *types.Broker) error {
	return nil
}

func (storage *brokerStorage) Find(ctx context.Context, id string) (*types.Broker, error) {
	return &types.Broker{
		Name:     "brokerName",
		ID:       "brokerID",
		URL:      "http://localhost:8080/broker",
		User:     "brokerAdmin",
		Password: "brokerAdmin",
	}, nil
}

func (storage *brokerStorage) FindAll(ctx context.Context) ([]*types.Broker, error) {
	return []*types.Broker{}, nil
}

func (storage *brokerStorage) Delete(ctx context.Context, id string) error {
	return nil
}

func (storage *brokerStorage) Update(ctx context.Context, broker *types.Broker) error {
	return nil
}
