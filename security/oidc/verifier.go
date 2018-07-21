package oidc

import (
	"context"

	"github.com/Peripli/service-manager/security"
	"github.com/coreos/go-oidc"
)

type oidcVerifier struct {
	*oidc.IDTokenVerifier
}

func (v *oidcVerifier) Verify(ctx context.Context, idToken string) (security.Token, error) {
	return v.IDTokenVerifier.Verify(ctx, idToken)
}
