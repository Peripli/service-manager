package authn

// DefaultSettings for the authentication config
func DefaultSettings() *Settings {
	return &Settings{
		SkipSSLValidation: false,
	}
}

// Settings for authentication confnig
type Settings struct {
	User              string `mapstructure:"user"`
	Password          string `mapstructure:"password"`
	TokenIssuerURL    string `mapstructure:"token_issuer_url"`
	ClientID          string `mapstructure:"client_id"`
	SkipSSLValidation bool   `mapstructure:"skip_ssl_validation"`
}
