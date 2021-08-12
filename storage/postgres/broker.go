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
	"database/sql"
	"fmt"

	"github.com/Peripli/service-manager/storage"
	sqlxtypes "github.com/jmoiron/sqlx/types"

	"github.com/Peripli/service-manager/pkg/types"
)

// Broker entity
//go:generate smgen storage broker github.com/Peripli/service-manager/pkg/types:ServiceBroker
type Broker struct {
	BaseEntity
	Name                  string             `db:"name"`
	Description           sql.NullString     `db:"description"`
	BrokerURL             string             `db:"broker_url"`
	Username              string             `db:"username"`
	Password              string             `db:"password"`
	Integrity             []byte             `db:"integrity"`
	TlsClientKey          string             `db:"tls_client_key"`
	TlsClientCertificate  string             `db:"tls_client_certificate"`
	Catalog               sqlxtypes.JSONText `db:"catalog"`
	SMProvidedCredentials bool               `db:"sm_provided_tls_credentials"`
	Services              []*ServiceOffering `db:"-"`
}

func (e *Broker) ToObject() (types.Object, error) {
	var services []*types.ServiceOffering
	for _, service := range e.Services {
		serviceObject, err := service.ToObject()
		if err != nil {
			return nil, fmt.Errorf("converting broker to object failed while converting services: %s", err)
		}
		services = append(services, serviceObject.(*types.ServiceOffering))
	}

	var tls *types.TLS
	if e.TlsClientCertificate != "" || e.TlsClientKey != "" {
		tls = &types.TLS{Certificate: e.TlsClientCertificate, Key: e.TlsClientKey}
	}
	tls.UseSMCertificate = e.SMProvidedCredentials

	var basic *types.Basic
	if e.Username != "" || e.Password != "" {
		basic = &types.Basic{
			Username: e.Username,
			Password: e.Password,
		}
	}

	broker := &types.ServiceBroker{
		Base: types.Base{
			ID:             e.ID,
			CreatedAt:      e.CreatedAt,
			UpdatedAt:      e.UpdatedAt,
			Labels:         map[string][]string{},
			PagingSequence: e.PagingSequence,
			Ready:          e.Ready,
		},
		Name:        e.Name,
		Description: e.Description.String,
		BrokerURL:   e.BrokerURL,
		Credentials: &types.Credentials{
			Basic:     basic,
			TLS:       tls,
			Integrity: e.Integrity,
		},
		Catalog:  getJSONRawMessage(e.Catalog),
		Services: services,
	}
	return broker, nil
}

func (*Broker) FromObject(object types.Object) (storage.Entity, error) {
	broker, ok := object.(*types.ServiceBroker)
	if !ok {
		return nil, nil
	}
	serviceOfferingDTO := &ServiceOffering{}
	var services []*ServiceOffering
	for _, service := range broker.Services {
		entity, err := serviceOfferingDTO.FromObject(service)
		if err != nil {
			return nil, fmt.Errorf("converting broker from object failed while converting services: %s", err)
		}

		services = append(services, entity.(*ServiceOffering))
	}
	b := &Broker{
		BaseEntity: BaseEntity{
			ID:             broker.ID,
			CreatedAt:      broker.CreatedAt,
			UpdatedAt:      broker.UpdatedAt,
			PagingSequence: broker.PagingSequence,
			Ready:          broker.Ready,
		},
		Name:        broker.Name,
		Description: toNullString(broker.Description),
		BrokerURL:   broker.BrokerURL,
		Catalog:     getJSONText(broker.Catalog),
		Services:    services,
	}
	if broker.Credentials != nil {
		b.Integrity = broker.Credentials.Integrity
		if broker.Credentials.Basic != nil {
			b.Username = broker.Credentials.Basic.Username
			b.Password = broker.Credentials.Basic.Password
		}

		if broker.Credentials.TLS != nil {
			b.TlsClientCertificate = broker.Credentials.TLS.Certificate
			b.TlsClientKey = broker.Credentials.TLS.Key
			b.SMProvidedCredentials = broker.Credentials.TLS.UseSMCertificate
		}

	}
	return b, nil
}
