package authn

import (
	"strings"

	"github.com/Peripli/service-manager/pkg/web"

	"github.com/Peripli/service-manager/pkg/log"
	httpsec "github.com/Peripli/service-manager/pkg/security/http"
)

type orAuthenticator struct {
	authenticators []httpsec.Authenticator
}

func NewOrAuthenticator(authenticators ...httpsec.Authenticator) httpsec.Authenticator {
	return &orAuthenticator{
		authenticators: authenticators,
	}
}

// Authenticate allows the request if at least one of the nested authenticators allows it
func (a *orAuthenticator) Authenticate(request *web.Request) (*web.UserContext, httpsec.Decision, error) {
	ctx := request.Context()
	logger := log.C(ctx)
	finalDecision := httpsec.Abstain
	var errs compositeError
	for _, authenticator := range a.authenticators {
		userContext, decision, err := authenticator.Authenticate(request)
		if err != nil {
			if decision != httpsec.Deny {
				logger.WithError(err).Error("OrAuthenticator: error during evaluate authenticator")
				return userContext, httpsec.Deny, err
			}
			errs = append(errs, err)
		}

		if decision == httpsec.Allow {
			logger.Debug("OrAuthenticator: one authenticator allowed, stop evaluating")
			return userContext, decision, nil
		}

		if decision == httpsec.Deny {
			finalDecision = httpsec.Deny
		}
	}
	if finalDecision == httpsec.Allow || finalDecision == httpsec.Abstain {
		return nil, finalDecision, nil
	}
	return nil, finalDecision, errs
}

type compositeError []error

func (c compositeError) Error() string {
	s := make([]string, 0, len(c))
	for _, e := range c {
		s = append(s, "Cause: "+e.Error())
	}

	return strings.Join(s, ". ")
}
