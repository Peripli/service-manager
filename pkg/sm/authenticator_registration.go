package sm

import (
	httpsec "github.com/Peripli/service-manager/pkg/security/http"
)

type namedAuthenticator struct {
	authenticator httpsec.Authenticator
	name          string
}

type authenticatorBuilder struct {
	securityBuilder *securityBuilder
	parent          *authenticatorBuilder

	authenticators []namedAuthenticator
	optional       bool
}

// func (ab *authenticatorBuilder) With(name string, authenticator httpsec.Authenticator) *securityBuilder {
// 	ab.securityBuilder.authenticators = append(ab.securityBuilder.authenticators, namedAuthenticator{
// 		name:          name,
// 		authenticator: authenticator,
// 	})
// 	return ab.securityBuilder
// }

// func (ab *authenticatorBuilder) Register() *ServiceManagerBuilder {
// 	paths := ab.paths
// 	if len(paths) == 0 {
// 		paths = typesToPaths(ab.objectTypes)
// 	}
// 	if len(ab.methods) == 0 {
// 		log.D().Panicf("Cannot register authenticators at %v with no methods", paths)
// 	}
// 	if len(ab.authenticators) == 0 {
// 		log.D().Panicf("Cannot register 0 authenticators at %v for %v", paths, ab.methods)
// 	}
// 	for _, authenticator := range ab.authenticators {
// 		filter := filters.NewAuthenticationFilter(authenticator.authenticator, fmt.Sprintf("%v-AuthN-%s@%v", ab.methods, authenticator.name, paths), []web.FilterMatcher{
// 			{
// 				Matchers: []web.Matcher{
// 					web.Path(paths...),
// 					web.Methods(ab.methods...),
// 				},
// 			},
// 		})
// 		ab.attachFunc(filter)
// 	}
// 	if !ab.optional {
// 		ab.attachFunc(filters.NewRequiredAuthnFilter(fmt.Sprintf("%v-RequiredAuthN@v", ab.methods, paths), []web.FilterMatcher{
// 			{
// 				Matchers: []web.Matcher{
// 					web.Path(paths...),
// 					web.Methods(ab.methods...),
// 				},
// 			},
// 		}))
// 	}
// 	return ab.done()
// }
