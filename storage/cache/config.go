package cache

// Settings type to be loaded from the environment
type Settings struct {
	Enabled    bool   `mapstructure:"enabled" description:"true if cache is enabled"`
	Port       int    `mapstructure:"port" description:"port for redis-cache"`
	Host       string `mapstructure:"host" description:"true if cache is enabled"`
	Password   string `mapstructure:"password" description:"true if cache is enabled"`
	TLSEnabled bool   `mapstructure:"tls_enabled" description:"true if tls is enabled"`
}

// DefaultSettings returns default values for cache settings
func DefaultSettings() *Settings {
	return &Settings{
		Enabled:    true,
		TLSEnabled: true,
	}
}
