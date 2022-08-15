package authz

import (
	"fmt"
	"strings"

	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/log"
	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/web"

	httpsec "github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/security/http"
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

	var errs compositeError = nil
	finalDecision := httpsec.Allow

	accessLevels := make([]web.AccessLevel, 0)
	for _, authorizer := range a.authorizers {
		decision, level, err := authorizer.Authorize(request)
		if err != nil {
			if decision != httpsec.Deny {
				logger.WithError(err).Error("AndAuthorizer: error during evaluate authorizer")
				return httpsec.Deny, web.NoAccess, err
			}
			finalDecision = httpsec.Deny
			errs = append(errs, err)
		}

		if decision != httpsec.Allow && finalDecision != httpsec.Deny {
			finalDecision = decision
		}

		accessLevels = append(accessLevels, level)
	}

	level := web.NoAccess
	if finalDecision == httpsec.Allow {
		level = findMostRestrictiveAccessLevel(accessLevels)
	}

	if finalDecision == httpsec.Allow || finalDecision == httpsec.Abstain {
		return finalDecision, level, nil
	}
	return finalDecision, level, errs
}

type compositeError []error

func (c compositeError) Error() string {
	s := make([]string, 0, len(c))
	for _, e := range c {
		s = append(s, "cause: "+e.Error())
	}
	if len(s) > 0 {
		return fmt.Sprintf("(%s)", strings.Join(s, "; "))
	} else {
		return ""
	}
}
