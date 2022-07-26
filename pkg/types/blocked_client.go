/*
 *    Copyright 2018 The Service Manager Authors
 *
 *    Licensed under the Apache License, Version 2.0 (the "License");
 *    you may not use this file except in compliance with the License.
 *    You may obtain a copy of the License at
 *
 *        http://www.apache.org/licenses/LICENSE-2.0
 *
 *    Unless required by applicable law or agreed to in writing, software
 *    distributed under the License is distributed on an "AS IS" BASIS,
 *    WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *    See the License for the specific language governing permissions and
 *    limitations under the License.
 */

package types

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/Peripli/service-manager/pkg/util"
	"github.com/Peripli/service-manager/pkg/web"
	"reflect"
	"time"
)

const BlockedClientsType ObjectType = web.BlockedClientsConfigURL

//go:generate smgen api BlockedClient
// BlockedClient struct
type BlockedClient struct {
	Base
	ClientID       string   `json:"client_id"`
	SubaccountID   string   `json:"subaccount_id"`
	BlockedMethods []string `json:"blocked_methods,omitempty"`
}

type BlockedClients struct {
	BlockedClients []*BlockedClient `json:"blocked_clients"`
}

func (e *BlockedClients) Validate() error {
	panic("implement me")
}

func (e *BlockedClients) Equals(object Object) bool {
	panic("implement me")
}

func (e *BlockedClients) SetID(id string) {
	panic("implement me")
}

func (e *BlockedClients) GetID() string {
	panic("implement me")
}

func (e *BlockedClients) GetLabels() Labels {
	panic("implement me")
}

func (e *BlockedClients) SetLabels(labels Labels) {
	panic("implement me")
}

func (e *BlockedClients) SetCreatedAt(time time.Time) {
	panic("implement me")
}

func (e *BlockedClients) GetCreatedAt() time.Time {
	panic("implement me")
}

func (e *BlockedClients) SetUpdatedAt(time time.Time) {
	panic("implement me")
}

func (e *BlockedClients) GetUpdatedAt() time.Time {
	panic("implement me")
}

func (e *BlockedClients) GetPagingSequence() int64 {
	panic("implement me")
}

func (e *BlockedClients) SetReady(b bool) {
	panic("implement me")
}

func (e *BlockedClients) GetReady() bool {
	panic("implement me")
}

func (e *BlockedClients) SetLastOperation(operation *Operation) {
	panic("implement me")
}

func (e *BlockedClients) GetLastOperation() *Operation {
	panic("implement me")
}

func (e *BlockedClient) GetType() ObjectType {
	return BlockedClientsType
}

func (e *BlockedClient) Equals(obj Object) bool {
	if !Equals(e, obj) {
		return false
	}

	blockedClient := obj.(*BlockedClient)
	if e.ClientID != blockedClient.ClientID ||
		e.SubaccountID != blockedClient.SubaccountID ||
		!reflect.DeepEqual(e.BlockedMethods, blockedClient.BlockedMethods) {
		return false
	}

	return true
}

// Validate implements InputValidator and verifies all mandatory fields are populated
func (e *BlockedClient) Validate() error {
	if util.HasRFC3986ReservedSymbols(e.ID) {
		return fmt.Errorf("%s contains invalid character(s)", e.ID)
	}
	if e.ClientID == "" {
		return errors.New("missing blocked client ID")
	}
	if e.SubaccountID == "" {
		return errors.New("missing blocked subaccount ID")
	}
	if err := e.Labels.Validate(); err != nil {
		return err
	}

	return nil
}

func (e *BlockedClients) Add(object Object) {
	e.BlockedClients = append(e.BlockedClients, object.(*BlockedClient))
}

func (e *BlockedClients) ItemAt(index int) Object {
	return e.BlockedClients[index]
}

func (e *BlockedClients) Len() int {
	return len(e.BlockedClients)
}

func (e *BlockedClients) GetType() ObjectType {
	return BlockedClientsType
}

// MarshalJSON override json serialization for http response
func (e *BlockedClient) MarshalJSON() ([]byte, error) {
	type E BlockedClient
	toMarshal := struct {
		*E
		CreatedAt *string `json:"created_at,omitempty"`
		UpdatedAt *string `json:"updated_at,omitempty"`
		Labels    Labels  `json:"labels,omitempty"`
	}{
		E:      (*E)(e),
		Labels: e.Labels,
	}
	if !e.CreatedAt.IsZero() {
		str := util.ToRFCNanoFormat(e.CreatedAt)
		toMarshal.CreatedAt = &str
	}
	if !e.UpdatedAt.IsZero() {
		str := util.ToRFCNanoFormat(e.UpdatedAt)
		toMarshal.UpdatedAt = &str
	}
	hasNoLabels := true
	for key, values := range e.Labels {
		if key != "" && len(values) != 0 {
			hasNoLabels = false
			break
		}
	}
	if hasNoLabels {
		toMarshal.Labels = nil
	}
	return json.Marshal(toMarshal)
}
