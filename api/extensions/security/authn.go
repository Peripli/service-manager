package security

import (
	"context"
	"net/http"

	"github.com/Peripli/service-manager/pkg/security/authenticators"

	"github.com/Peripli/service-manager/config"

	"github.com/Peripli/service-manager/pkg/sm"

	"github.com/Peripli/service-manager/pkg/web"
)

// Register adds security configuration to the service manager builder
func Register(ctx context.Context, cfg *config.Settings, smb *sm.ServiceManagerBuilder) error {
	basicPlatformAuthenticator := &authenticators.Basic{
		Repository:             smb.Storage,
		BasicAuthenticatorFunc: authenticators.BasicPlatformAuthenticator,
	}

	smb.Security().Path(
		web.ServiceBrokersURL+"/*",
		web.PlatformsURL+"/*",
		web.ServiceOfferingsURL+"/*",
		web.ServicePlansURL+"/*",
		web.VisibilitiesURL+"/*",
		web.ServiceInstancesURL+"/*",
		web.ServiceBindingsURL+"/*",
		web.NotificationsURL+"/*").
		Method(http.MethodGet).
		WithAuthentication(basicPlatformAuthenticator).Required()

	smb.Security().
		Path(web.BrokerPlatformCredentialsURL + "/**").
		Method(http.MethodPut).
		WithAuthentication(basicPlatformAuthenticator).Required()

	basicOSBAuthenticator := &authenticators.Basic{
		Repository:             smb.Storage,
		BasicAuthenticatorFunc: authenticators.BasicOSBAuthenticator,
	}

	smb.Security().
		Path(web.OSBURL+"/**").
		Method(http.MethodGet, http.MethodPut, http.MethodPatch, http.MethodDelete).
		WithAuthentication(basicOSBAuthenticator).Required()

	bearerAuthenticator, _, err := authenticators.NewOIDCAuthenticator(ctx, &authenticators.OIDCOptions{
		IssuerURL: cfg.API.TokenIssuerURL,
		ClientID:  cfg.API.ClientID,
	})
	if err != nil {
		return err
	}

	smb.Security().Path(
		web.ServiceBrokersURL+"/**",
		web.PlatformsURL+"/**",
		web.ServiceOfferingsURL+"/**",
		web.ServicePlansURL+"/**",
		web.VisibilitiesURL+"/**",
		web.NotificationsURL+"/**",
		web.ServiceInstancesURL+"/**",
		web.ServiceBindingsURL+"/**",
		web.ConfigURL+"/**",
		web.ProfileURL+"/**",
	).
		Method(http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete).
		WithAuthentication(bearerAuthenticator).Required()

	return nil
}
