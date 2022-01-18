package authz

import (
	httpsec "github.com/Peripli/service-manager/pkg/security/http"
	"github.com/Peripli/service-manager/pkg/web"
	. "github.com/onsi/ginkgo/v2"
)

var _ = Describe("CloneFilter", func() {
	DescribeTable("Run", func(t testCase) {
		runTestCase(t, NewClientIDSuffixAuthorizer(t.params.(string), web.GlobalAccess))
	}, []TableEntry{
		Entry("Fails if no user is authenticated", testCase{
			params:           "",
			noUser:           true,
			expectedDecision: httpsec.Abstain,
			expectedAccess:   web.NoAccess,
		}),
		Entry("Fails if token claims is invalid json", testCase{

			params:           "|suffix",
			claims:           `{"invalid}`,
			expectError:      "invalid token: unexpected end of JSON input",
			expectedDecision: httpsec.Deny,
			expectedAccess:   web.NoAccess,
		}),
		Entry("Succeeds if token is generated from Master Oauth client", testCase{
			params:           "|suffix",
			claims:           `{"cid": "some-id|suffix"}`,
			expectedDecision: httpsec.Allow,
			expectedAccess:   web.GlobalAccess,
		}),
		Entry("Fails if token is not generated from Master Oauth client", testCase{
			params:           "|suffix",
			claims:           `{"cid": "wrong-sufix"}`,
			expectError:      `client id "wrong-sufix" from user token does not have the required suffix`,
			expectedDecision: httpsec.Deny,
			expectedAccess:   web.NoAccess,
		}),
	})
})
