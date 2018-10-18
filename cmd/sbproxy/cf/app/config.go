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

// Package cf contains the cf specific logic for the proxy
package app

import (
	"time"

	"github.com/cloudfoundry-community/go-cfclient"

	"errors"

	"github.com/Peripli/service-manager/pkg/env"
	"github.com/spf13/pflag"
)

// RegistrationDetails type represents the credentials used to register a broker at the cf
type RegistrationDetails struct {
	User     string
	Password string
}

// ClientConfiguration type holds config info for building the cf client
type ClientConfiguration struct {
	*cfclient.Config   `mapstructure:"client"`
	Reg                *RegistrationDetails
	CfClientCreateFunc func(*cfclient.Config) (*cfclient.Client, error)
}

// Settings type wraps the CF client configuration
type Settings struct {
	Cf *ClientConfiguration
}

// DefaultClientConfiguration creates a default config for the CF client
func DefaultClientConfiguration() *ClientConfiguration {
	cfClientConfig := cfclient.DefaultConfig()
	cfClientConfig.HttpClient.Timeout = 10 * time.Second

	return &ClientConfiguration{
		Config:             cfClientConfig,
		Reg:                &RegistrationDetails{},
		CfClientCreateFunc: cfclient.NewClient,
	}
}

// CreatePFlagsForCFClient adds pflags relevant to the CF client config
func CreatePFlagsForCFClient(set *pflag.FlagSet) {
	env.CreatePFlags(set, &Settings{Cf: DefaultClientConfiguration()})
}

// Validate validates the configuration and returns appropriate errors in case it is invalid
func (c *ClientConfiguration) Validate() error {
	if c.CfClientCreateFunc == nil {
		return errors.New("CF ClientCreateFunc missing")
	}
	if c.Config == nil {
		return errors.New("CF client configuration missing")
	}
	if len(c.ApiAddress) == 0 {
		return errors.New("CF client configuration ApiAddress missing")
	}
	if c.HttpClient != nil && c.HttpClient.Timeout == 0 {
		return errors.New("CF client configuration timeout missing")
	}
	if c.Reg == nil {
		return errors.New("CF client configuration Registration credentials missing")
	}
	if len(c.Reg.User) == 0 {
		return errors.New("CF client configuration Registration details user missing")
	}
	if len(c.Reg.Password) == 0 {
		return errors.New("CF client configuration Registration details password missing")
	}
	return nil
}

// NewConfig creates ClientConfiguration from the provided environment
func NewConfig(env env.Environment) (*ClientConfiguration, error) {
	cfSettings := &Settings{Cf: DefaultClientConfiguration()}

	if err := env.Unmarshal(cfSettings); err != nil {
		return nil, err
	}

	return cfSettings.Cf, nil
}
