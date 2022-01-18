package authz

import (
	httpsec "github.com/Peripli/service-manager/pkg/security/http"
	"github.com/Peripli/service-manager/pkg/web"
	. "github.com/onsi/ginkgo/v2"
)

var _ = Describe("ClientID Authorizer", func() {
	DescribeTable("Run", func(tc testCase) {
		runTestCase(tc, NewOauthClientAuthorizer(tc.params.(string), web.GlobalAccess))
	}, []TableEntry{
		Entry("Fails if no user is authenticated", testCase{
			params:           "",
			noUser:           true,
			expectedDecision: httpsec.Abstain,
			expectedAccess:   web.NoAccess,
		}),
		Entry("Fails if token claims is invalid json", testCase{
			params:           "client-id",
			claims:           `{"invalid}`,
			expectError:      "invalid token: unexpected end of JSON input",
			expectedDecision: httpsec.Deny,
			expectedAccess:   web.NoAccess,
		}),
		Entry("Succeeds if token is generated from correct Oauth client", testCase{
			params:           "client-id",
			claims:           `{"cid": "client-id"}`,
			expectedDecision: httpsec.Allow,
			expectedAccess:   web.GlobalAccess,
		}),
		Entry("Fails if token is not generated from correct Oauth client", testCase{
			params:           "client-id",
			claims:           `{"cid": "wrong-client"}`,
			expectError:      `client id "wrong-client" from user token is not trusted`,
			expectedDecision: httpsec.Deny,
			expectedAccess:   web.NoAccess,
		}),
	})
})
