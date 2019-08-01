package multitenancy

import (
	"github.com/Peripli/service-manager/pkg/log"
	"github.com/Peripli/service-manager/pkg/web"
)

// ExtractTenatFromTokenWrapperFunc returns function which extracts tenant from JWT token
func ExtractTenatFromTokenWrapperFunc() func(request *web.Request) (string, error) {
	return func(request *web.Request) (string, error) {
		ctx := request.Context()
		logger := log.C(ctx)

		user, ok := web.UserFromContext(ctx)
		if !ok {
			logger.Infof("No user found in user context. Proceeding with next filter...")
			return "", nil
		}

		return user.TenantID, nil
	}
}
