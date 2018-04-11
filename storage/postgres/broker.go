package postgres

import (
	"github.com/Peripli/service-manager/types"
	"github.com/jmoiron/sqlx"
)

type brokerStorage struct {
	db *sqlx.DB
}

func (storage *brokerStorage) Create(broker *types.Broker) error {
	return nil
}

func (storage *brokerStorage) Find(id string) (*types.Broker, error) {
	return &types.Broker{
		Name:     "brokerName",
		ID:       "brokerID",
		URL:      "http://localhost:8080/broker",
		User:     "brokerAdmin",
		Password: "brokerAdmin",
	}, nil
}

func (storage *brokerStorage) FindAll() ([]*types.Broker, error) {
	return []*types.Broker{}, nil
}

func (storage *brokerStorage) Delete(id string) error {
	return nil
}

func (storage *brokerStorage) Update(broker *types.Broker) error {
	return nil
}
