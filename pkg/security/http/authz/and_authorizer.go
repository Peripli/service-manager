package authz

import (
	"fmt"
	"strings"

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

	errs := compositeError{}
	abstain := false
	denied := false

	accessLevels := make([]web.AccessLevel, 0)
	for _, authorizer := range a.authorizers {
		decision, level, err := authorizer.Authorize(request)
		if err != nil {
			logger.WithError(err).Debug("AndAuthorizer: error during evaluate authorizer")
			if decision == httpsec.Deny {
				logger.Debug("AndAuthorizer: one authorizer denied, stop evaluating")
				denied = true
				errs = append(errs, err)
				continue
			}
			return decision, web.NoAccess, err
		}

		if decision == httpsec.Deny {
			denied = true
		} else if decision == httpsec.Allow {
			accessLevels = append(accessLevels, level)
		} else {
			abstain = true
		}
	}

	if denied {
		return httpsec.Deny, web.NoAccess, errs
	} else if abstain {
		return httpsec.Abstain, web.NoAccess, nil
	}

	return httpsec.Allow, findMostRestrictiveAccessLevel(accessLevels), nil
}

type compositeError []error

func (c compositeError) Error() string {
	s := make([]string, 0, len(c))
	for _, e := range c {
		s = append(s, "cause: "+e.Error())
	}

	return fmt.Sprintf("(%s)", strings.Join(s, "; "))
}
