package authz

import (
	"github.com/Peripli/service-manager/pkg/log"
	"github.com/Peripli/service-manager/pkg/web"

	httpsec "github.com/Peripli/service-manager/pkg/security/http"
)

type andAuthorizer struct {
	authorizers []httpsec.Authorizer
}

func NewAndAuthorizer(authorizers ...httpsec.Authorizer) httpsec.Authorizer {
	return &andAuthorizer{
		authorizers: authorizers,
	}
}

// Authorize allows the request only if all nested authorizers allow it and denies even if one denies
func (a *andAuthorizer) Authorize(request *web.Request) (httpsec.Decision, web.AccessLevel, error) {
	ctx := request.Context()
	logger := log.C(ctx)

	allowed := true
	accessLevels := make([]web.AccessLevel, 0)
	for _, authorizer := range a.authorizers {
		decision, level, err := authorizer.Authorize(request)
		if err != nil {
			logger.WithError(err).Debug("AndAuthorizer: error during evaluate authorizer")
			return decision, web.NoAccess, err
		}

		if decision == httpsec.Deny {
			logger.Debug("AndAuthorizer: one authorizer denied, stop evaluating")
			return httpsec.Deny, web.NoAccess, nil
		}

		if decision != httpsec.Allow {
			allowed = false
		} else {
			accessLevels = append(accessLevels, level)
		}
	}

	if allowed {
		return httpsec.Allow, findMostRestrictiveAccessLevel(accessLevels), nil
	}

	return httpsec.Abstain, web.NoAccess, nil
}
