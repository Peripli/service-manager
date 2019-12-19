package security

import (
	"context"
	"net/http"

	"github.com/Peripli/service-manager/pkg/security/authenticators"

	"github.com/Peripli/service-manager/config"

	"github.com/Peripli/service-manager/pkg/web"

	"github.com/Peripli/service-manager/pkg/sm"
)

type SecurityExtendable struct {
	cfg *config.Settings
}

func NewSecurityExtension(cfg *config.Settings) *SecurityExtendable {
	return &SecurityExtendable{
		cfg: cfg,
	}
}

func (se *SecurityExtendable) Extend(ctx context.Context, smb *sm.ServiceManagerBuilder) error {
	basicAuthenticator := &authenticators.Basic{
		Repository: smb.Storage,
	}

	smb.Security().Path(
		web.ServiceBrokersURL+"/**",
		web.PlatformsURL+"/**",
		web.ServiceOfferingsURL+"/**",
		web.ServicePlansURL+"/**",
		web.VisibilitiesURL+"/**",
		web.ServiceInstancesURL+"/**",
		web.NotificationsURL+"/**").
		Method(http.MethodGet).
		WithAuthentication(basicAuthenticator).Required()

	smb.Security().
		Path(web.OSBURL+"/**").
		Method(http.MethodGet, http.MethodPut, http.MethodPatch, http.MethodDelete).
		WithAuthentication(basicAuthenticator).Required()

	bearerAuthenticator, _, err := authenticators.NewOIDCAuthenticator(ctx, &authenticators.OIDCOptions{
		IssuerURL: se.cfg.API.TokenIssuerURL,
		ClientID:  se.cfg.API.ClientID,
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
		web.ConfigURL+"/**").
		Method(http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete).
		WithAuthentication(bearerAuthenticator).Required()

	return nil
}
