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
	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/web"
	"reflect"
	"time"

	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/util"
)

const CFPlatformType = "cloudfoundry"
const K8sPlatformType = "kubernetes"
const SMPlatform = "service-manager"
const Monitored = "monitored"

var smSupportedPlatformType = []string{SMPlatform}

// Returns aliases of service manager platform
func GetSMSupportedPlatformTypes() []string {
	return smSupportedPlatformType
}

func SetSMSupportedPlatformTypes(typee []string) {
	smSupportedPlatformType = typee
}

//go:generate smgen api Platform
// Platform platform struct
type Platform struct {
	Base
	Secured           `json:"-"`
	Strip             `json:"-"`
	Type              string       `json:"type"`
	Name              string       `json:"name"`
	Description       string       `json:"description"`
	Credentials       *Credentials `json:"credentials,omitempty"`
	OldCredentials    *Credentials `json:"old_credentials,omitempty"`
	Version           string       `json:"-"`
	Active            bool         `json:"-"`
	Suspended         bool         `json:"suspended,omitempty"`
	LastActive        time.Time    `json:"-"`
	Integrity         []byte       `json:"-"`
	CredentialsActive bool         `json:"credentials_active,omitempty"`
	Technical         bool         `json:"technical,omitempty"` //technical platforms are only used for managing visibilities, and are excluded in notification and credential management flows
}

func (e *Platform) Equals(obj Object) bool {
	if !Equals(e, obj) {
		return false
	}

	platform := obj.(*Platform)
	if e.Description != platform.Description ||
		e.Type != platform.Type ||
		e.Name != platform.Name ||
		e.Active != platform.Active ||
		e.Version != platform.Version ||
		!e.LastActive.Equal(platform.LastActive) ||
		!reflect.DeepEqual(e.Credentials, platform.Credentials) {
		return false
	}

	return true
}

func (e *Platform) Sanitize(ctx context.Context) {
	if !web.IsGeneratePlatformCredentialsRequired(ctx) {
		e.Credentials = nil
	}
	e.OldCredentials = nil
	e.CredentialsActive = false
	e.Technical = false
}

func (e *Platform) Encrypt(ctx context.Context, encryptionFunc func(context.Context, []byte) ([]byte, error)) error {
	if !e.Technical {
		return e.transform(ctx, encryptionFunc)
	}
	return nil
}

func (e *Platform) Decrypt(ctx context.Context, decryptionFunc func(context.Context, []byte) ([]byte, error)) error {
	if !e.Technical {
		return e.transform(ctx, decryptionFunc)
	}
	return nil
}

func (e *Platform) transform(ctx context.Context, transformationFunc func(context.Context, []byte) ([]byte, error)) error {
	var credentialsExist bool
	var oldCredentialsExist bool
	if e.Credentials != nil && e.Credentials.Basic != nil {
		credentialsExist = true
	}
	if e.OldCredentials != nil && e.OldCredentials.Basic != nil {
		oldCredentialsExist = true
	}

	if credentialsExist {
		transformedPassword, err := transformationFunc(ctx, []byte(e.Credentials.Basic.Password))
		if err != nil {
			return err
		}
		e.Credentials.Basic.Password = string(transformedPassword)
	}
	if oldCredentialsExist {
		transformedOldPassword, err := transformationFunc(ctx, []byte(e.OldCredentials.Basic.Password))
		if err != nil {
			return err
		}
		e.OldCredentials.Basic.Password = string(transformedOldPassword)
	}
	return nil
}

func (e *Platform) IntegralData() []byte {
	integrity := ""
	if !e.Technical {
		oldCredentials := ""
		if e.OldCredentials != nil && e.OldCredentials.Basic != nil {
			oldCredentials = fmt.Sprintf(":%s:%s", e.OldCredentials.Basic.Username, e.OldCredentials.Basic.Password)
		}

		if e.Credentials != nil {
			integrity = fmt.Sprintf("%s:%s%s", e.Credentials.Basic.Username, e.Credentials.Basic.Password, oldCredentials)
		}
	}
	return []byte(integrity)
}

func (e *Platform) SetIntegrity(integrity []byte) {
	e.Integrity = integrity
}

func (e *Platform) GetIntegrity() []byte {
	return e.Integrity
}

// Validate implements InputValidator and verifies all mandatory fields are populated
func (e *Platform) Validate() error {
	if e.Type == "" {
		return errors.New("missing platform type")
	}
	if e.Name == "" {
		return errors.New("missing platform name")
	}
	if util.HasRFC3986ReservedSymbols(e.ID) {
		return fmt.Errorf("%s contains invalid character(s)", e.ID)
	}
	return nil
}
