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
	"errors"
	"fmt"

	"github.com/Peripli/service-manager/pkg/util"
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
		e.BrokerID != instance.BrokerID {
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
