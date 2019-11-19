package config

import (
	"errors"
	"fmt"
	"time"

	"k8s.io/client-go/tools/clientcmd"

	"github.com/Peripli/service-manager/pkg/agent"
	"k8s.io/client-go/rest"

	"github.com/Peripli/service-manager/pkg/env"
	svcatclient "github.com/kubernetes-sigs/service-catalog/pkg/client/clientset_generated/clientset"
	servicecatalog "github.com/kubernetes-sigs/service-catalog/pkg/svcat/service-catalog"

	k8sclient "k8s.io/client-go/kubernetes"

	"github.com/spf13/pflag"
)

// Settings type wraps the K8S client configuration
type Settings struct {
	agent.Settings `mapstructure:",squash"`
	K8S            *ClientConfiguration `mapstructure:"k8s"`
}

// DefaultSettings returns the default settings for the k8s agent
func DefaultSettings() *Settings {
	return &Settings{
		Settings: *agent.DefaultSettings(),
		K8S:      DefaultClientConfiguration(),
	}
}

// Validate validates the application settings
func (s *Settings) Validate() error {
	if err := s.K8S.Validate(); err != nil {
		return err
	}
	return s.Settings.Validate()
}

// ClientConfiguration type holds config info for building the k8s service catalog client
type ClientConfiguration struct {
	ClientSettings      *LibraryConfig                                    `mapstructure:"client"`
	Secret              *SecretRef                                        `mapstructure:"secret"`
	K8sClientCreateFunc func(*LibraryConfig) (*servicecatalog.SDK, error) `mapstructure:"-"`
}

// Validate validates the configuration and returns appropriate errors in case it is invalid
func (c *ClientConfiguration) Validate() error {
	if c.K8sClientCreateFunc == nil {
		return errors.New("K8S ClientCreateFunc missing")
	}
	if c.ClientSettings == nil {
		return errors.New("K8S client configuration missing")
	}
	if err := c.ClientSettings.Validate(); err != nil {
		return err
	}
	if c.Secret == nil {
		return errors.New("K8S broker secret missing")
	}
	if err := c.Secret.Validate(); err != nil {
		return err
	}
	return nil
}

// LibraryConfig configurations for the k8s library
type LibraryConfig struct {
	Host             string                             `mapstructure:"host"`
	Timeout          time.Duration                      `mapstructure:"timeout"`
	KubeConfigPath   string                             `mapstructure:"kube_config_path"`
	NewClusterConfig func(string) (*rest.Config, error) `mapstructure:"-"`
}

// Validate validates the library configurations and returns appropriate errors in case it is invalid
func (r *LibraryConfig) Validate() error {
	if r.Timeout == 0 {
		return errors.New("K8S client configuration timeout missing")
	}
	if r.NewClusterConfig == nil {
		return errors.New("K8S client cluster configuration missing")
	}
	return nil
}

// SecretRef reference to secret used for broker registration
type SecretRef struct {
	Namespace string
	Name      string
}

// Validate validates the registration details and returns appropriate errors in case it is invalid
func (r *SecretRef) Validate() error {
	if r.Name == "" || r.Namespace == "" {
		return errors.New("properties of K8S secret configuration for broker registration missing")
	}
	return nil
}

// NewSvcatSDK creates a service-catalog client from configuration
func NewSvcatSDK(libraryConfig *LibraryConfig) (*servicecatalog.SDK, error) {
	config, err := libraryConfig.NewClusterConfig(libraryConfig.KubeConfigPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load cluster config: %s", err.Error())
	}

	if len(libraryConfig.Host) > 0 {
		config.Host = libraryConfig.Host
	}
	config.Timeout = libraryConfig.Timeout

	svcatClient, err := svcatclient.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create new svcat client: %s", err.Error())
	}

	k8sClient, err := k8sclient.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create new k8sClient: %s", err.Error())
	}

	return &servicecatalog.SDK{
		K8sClient:            k8sClient,
		ServiceCatalogClient: svcatClient,
	}, nil
}

// DefaultClientConfiguration creates a default config for the K8S client
func DefaultClientConfiguration() *ClientConfiguration {
	return &ClientConfiguration{
		ClientSettings: &LibraryConfig{
			Timeout: time.Second * 10,
			NewClusterConfig: func(kubeConfigPath string) (config *rest.Config, e error) {
				return clientcmd.BuildConfigFromFlags("", kubeConfigPath) // if kubeConfigPath is empty fallbacks to InClusterConfig
			},
		},
		Secret:              &SecretRef{},
		K8sClientCreateFunc: NewSvcatSDK,
	}
}

// CreatePFlagsForK8SClient adds pflags relevant to the K8S client config
func CreatePFlagsForK8SClient(set *pflag.FlagSet) {
	env.CreatePFlags(set, DefaultSettings())
}

// NewConfig creates Settings from the provided environment
func NewConfig(env env.Environment) (*Settings, error) {
	settings := DefaultSettings()

	if err := env.Unmarshal(settings); err != nil {
		return nil, err
	}

	return settings, nil
}
