/*
 * Copyright 2018 The Service Manager Authors
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
	"context"
	"errors"
	"fmt"

	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/util"
)

//go:generate smgen api BrokerPlatformCredential
// BrokerPlatformCredential struct
type BrokerPlatformCredential struct {
	Base

	Username        string `json:"username"`
	PasswordHash    string `json:"password_hash"`
	OldUsername     string `json:"old_username"`
	OldPasswordHash string `json:"old_password_hash"`

	PlatformID string `json:"platform_id"`
	BrokerID   string `json:"broker_id"`

	NotificationID string `json:"notification_id,omitempty"`
	Integrity      []byte `json:"-"`

	Active bool `json:"active"`
}

func (e *BrokerPlatformCredential) Encrypt(ctx context.Context, encryptionFunc func(context.Context, []byte) ([]byte, error)) error {
	return e.transform(ctx, encryptionFunc)
}

func (e *BrokerPlatformCredential) Decrypt(ctx context.Context, decryptionFunc func(context.Context, []byte) ([]byte, error)) error {
	return e.transform(ctx, decryptionFunc)
}

func (e *BrokerPlatformCredential) transform(ctx context.Context, transformationFunc func(context.Context, []byte) ([]byte, error)) error {
	transformedPassword, err := transformationFunc(ctx, []byte(e.PasswordHash))
	if err != nil {
		return err
	}
	e.PasswordHash = string(transformedPassword)

	if e.OldPasswordHash != "" {
		transformedOldPassword, err := transformationFunc(ctx, []byte(e.OldPasswordHash))
		if err != nil {
			return err
		}
		e.OldPasswordHash = string(transformedOldPassword)
	}
	return nil
}

func (e *BrokerPlatformCredential) IntegralData() []byte {
	return []byte(fmt.Sprintf("%s:%s:%s:%s:%s:%s", e.Username, e.PasswordHash, e.OldUsername, e.OldPasswordHash, e.BrokerID, e.PlatformID))
}

func (e *BrokerPlatformCredential) SetIntegrity(integrity []byte) {
	e.Integrity = integrity
}

func (e *BrokerPlatformCredential) GetIntegrity() []byte {
	return e.Integrity
}

func (e *BrokerPlatformCredential) Equals(obj Object) bool {
	if !Equals(e, obj) {
		return false
	}

	instance := obj.(*BrokerPlatformCredential)
	if e.Username != instance.Username ||
		e.PasswordHash != instance.PasswordHash ||
		e.OldUsername != instance.OldUsername ||
		e.OldPasswordHash != instance.OldPasswordHash ||
		e.PlatformID != instance.PlatformID ||
		e.BrokerID != instance.BrokerID ||
		e.NotificationID != instance.NotificationID {
		return false
	}

	return true
}

// Validate implements InputValidator and verifies all mandatory fields are populated
func (e *BrokerPlatformCredential) Validate() error {
	if util.HasRFC3986ReservedSymbols(e.ID) {
		return fmt.Errorf("%s contains invalid character(s)", e.ID)
	}
	if e.Username == "" {
		return errors.New("missing username")
	}
	if e.PasswordHash == "" {
		return errors.New("missing password_hash")
	}
	if e.BrokerID == "" {
		return errors.New("missing broker id")
	}

	return nil
}
