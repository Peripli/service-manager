package authz

import (
	"strings"

	"github.com/Peripli/service-manager/pkg/web"

	"github.com/Peripli/service-manager/pkg/log"
	httpsec "github.com/Peripli/service-manager/pkg/security/http"
)

type orAuthorizer struct {
	authorizers []httpsec.Authorizer
}

func NewOrAuthorizer(authorizers ...httpsec.Authorizer) httpsec.Authorizer {
	return &orAuthorizer{
		authorizers: authorizers,
	}
}

// Authorize allows the request if at least one of the nested authorizers allows it
func (a *orAuthorizer) Authorize(request *web.Request) (httpsec.Decision, web.AccessLevel, error) {
	ctx := request.Context()
	logger := log.C(ctx)
	denied := false
	errs := compositeError{}

	for _, authorizer := range a.authorizers {
		decision, accessLevel, err := authorizer.Authorize(request)
		if err != nil {
			logger.WithError(err).Debug("OrAuthorizer: error during evaluate authorizer")
			if decision != httpsec.Deny {
				return decision, web.NoAccess, err
			}
			errs = append(errs, err)
		}

		if decision == httpsec.Allow {
			logger.Debug("OrAuthorizer: one authorizer allowed, stop evaluating")
			return httpsec.Allow, accessLevel, nil
		}

		if decision == httpsec.Deny {
			denied = true
		}
	}

	if denied {
		if len(errs) == 0 {
			return httpsec.Deny, web.NoAccess, nil
		}
		return httpsec.Deny, web.NoAccess, errs
	}

	return httpsec.Abstain, web.NoAccess, nil
}

type compositeError []error

func (c compositeError) Error() string {
	s := make([]string, 0, len(c))
	for _, e := range c {
		s = append(s, "Cause: "+e.Error())
	}

	return strings.Join(s, ". ")
}
