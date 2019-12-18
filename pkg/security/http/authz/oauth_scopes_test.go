package authz

import (
	"errors"

	"github.com/Peripli/service-manager/pkg/web"

	httpsec "github.com/Peripli/service-manager/pkg/security/http"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
)

var _ = Describe("ScopeAuthorizer", func() {
	DescribeTable("Required", func(t testCase) {
		runTestCase(t, NewScopesAuthorizer(t.params.([]string), web.GlobalAccess))
	}, []TableEntry{
		Entry("Fails if no user is authenticated", testCase{
			params:           []string{""},
			noUser:           true,
			expectedDecision: httpsec.Abstain,
			expectedAccess:   web.NoAccess,
		}),
		Entry("Fails if token claims cannot be extracted", testCase{
			params:           []string{""},
			claimsError:      errors.New("claims error"),
			expectError:      "could not extract scopes",
			expectedDecision: httpsec.Deny,
			expectedAccess:   web.NoAccess,
		}),
		Entry("Fails if there are no scopes in the token", testCase{
			params:           []string{"scope2"},
			claims:           `{}`,
			expectError:      `none of the scopes [scope2] are present`,
			expectedDecision: httpsec.Deny,
			expectedAccess:   web.NoAccess,
		}),
		Entry("Fails if scope does not match", testCase{
			params:           []string{"scope2"},
			claims:           `{"scope":["scope1","scope3"]}`,
			expectError:      `none of the scopes [scope2] are present in the user token scopes [scope1 scope3]`,
			expectedDecision: httpsec.Deny,
			expectedAccess:   web.NoAccess,
		}),
		Entry("Succeeds if scope matches", testCase{
			params:           []string{"scope2"},
			claims:           `{"scope":["scope1","scope2","scope3"]}`,
			expectedDecision: httpsec.Allow,
			expectedAccess:   web.GlobalAccess,
		}),
	}...)
})
