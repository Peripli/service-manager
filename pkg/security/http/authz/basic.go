package authz

import (
	"github.com/Peripli/service-manager/pkg/log"
	httpsec "github.com/Peripli/service-manager/pkg/security/http"
	"github.com/Peripli/service-manager/pkg/web"
)

func NewBasic(level web.AccessLevel) httpsec.Authorizer {
	return &basicAuthorizer{
		level: level,
	}
}

type basicAuthorizer struct {
	level web.AccessLevel
}

func (a *basicAuthorizer) Authorize(request *web.Request) (httpsec.Decision, web.AccessLevel, error) {
	logger := log.C(request.Context())
	if _, _, ok := request.BasicAuth(); ok {
		logger.Debugf("Authentication is Basic. No authorization required...")
		return httpsec.Allow, a.level, nil
	}
	logger.Debugf("Authentication is not Basic. Skipping basic authorization...")
	return httpsec.Abstain, web.NoAccess, nil
}
