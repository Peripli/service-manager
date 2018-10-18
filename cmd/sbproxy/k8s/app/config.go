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

package app

import (
	"errors"
	"fmt"
	"time"

	"github.com/Peripli/service-manager/pkg/env"
	svcatclient "github.com/kubernetes-incubator/service-catalog/pkg/client/clientset_generated/clientset"
	"github.com/kubernetes-incubator/service-catalog/pkg/svcat/service-catalog"

	k8sclient "k8s.io/client-go/kubernetes"

	"github.com/spf13/pflag"
)

// LibraryConfig configurations for the k8s library
type LibraryConfig struct {
	Timeout time.Duration `mapstructure:"timeout"`
}

// SecretRef reference to secret used for broker registration
type SecretRef struct {
	Namespace string
	Name      string
}

// RegistrationDetails type represents the credentials and secret name used to register a broker at the k8s cluster
type RegistrationDetails struct {
	User     string
	Password string
	Secret   *SecretRef
}

// ClientConfiguration type holds config info for building the k8s service catalog client
type ClientConfiguration struct {
	Client              *LibraryConfig       `mapstructure:"client"`
	Reg                 *RegistrationDetails `mapstructure:"reg"`
	K8sClientCreateFunc func(*LibraryConfig) (*servicecatalog.SDK, error)
}

// Settings type wraps the K8S client configuration
type Settings struct {
	K8S *ClientConfiguration `mapstructure:"k8s"`
}

// newSvcatSDK creates a service-catalog client from configuration
func newSvcatSDK(libraryConfig *LibraryConfig) (*servicecatalog.SDK, error) {
	config, err := restInClusterConfig()
	if err != nil {
		return nil, fmt.Errorf("Failed to load cluster config: %s", err.Error())
	}

	config.Timeout = libraryConfig.Timeout

	svcatClient, err := svcatclient.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("Failed to create new svcat client: %s", err.Error())
	}

	k8sClient, err := k8sclient.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("Failed to create new k8sClient: %s", err.Error())
	}

	return &servicecatalog.SDK{
		K8sClient:            k8sClient,
		ServiceCatalogClient: svcatClient,
	}, nil
}

// defaultClientConfiguration creates a default config for the K8S client
func defaultClientConfiguration() *ClientConfiguration {
	return &ClientConfiguration{
		Client: &LibraryConfig{
			Timeout: time.Second * 10,
		},
		Reg: &RegistrationDetails{
			Secret: &SecretRef{},
		},
		K8sClientCreateFunc: newSvcatSDK,
	}
}

// CreatePFlagsForK8SClient adds pflags relevant to the K8S client config
func CreatePFlagsForK8SClient(set *pflag.FlagSet) {
	env.CreatePFlags(set, &Settings{K8S: defaultClientConfiguration()})
}

// Validate validates the configuration and returns appropriate errors in case it is invalid
func (c *ClientConfiguration) Validate() error {
	if c.K8sClientCreateFunc == nil {
		return errors.New("K8S ClientCreateFunc missing")
	}
	if c.Client == nil {
		return errors.New("K8S client configuration missing")
	}
	if err := c.Client.Validate(); err != nil {
		return err
	}
	if c.Reg == nil {
		return errors.New("K8S broker registration configuration missing")
	}
	if err := c.Reg.Validate(); err != nil {
		return err
	}
	return nil
}

// Validate validates the registration details and returns appropriate errors in case it is invalid
func (r *RegistrationDetails) Validate() error {
	if r.User == "" || r.Password == "" {
		return errors.New("K8S broker registration credentials missing")
	}
	if r.Secret == nil {
		return errors.New("K8S secret configuration for broker registration missing")
	}
	if r.Secret.Name == "" || r.Secret.Namespace == "" {
		return errors.New("Properties of K8S secret configuration for broker registration missing")
	}
	return nil
}

// Validate validates the library configurations and returns appropriate errors in case it is invalid
func (r *LibraryConfig) Validate() error {
	if r.Timeout == 0 {
		return errors.New("K8S client configuration timeout missing")
	}
	return nil
}

// NewConfig creates ClientConfiguration from the provided environment
func NewConfig(env env.Environment) (*ClientConfiguration, error) {
	k8sSettings := &Settings{K8S: defaultClientConfiguration()}

	if err := env.Unmarshal(k8sSettings); err != nil {
		return nil, err
	}

	return k8sSettings.K8S, nil
}
