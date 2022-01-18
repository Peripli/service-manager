package authz

import (
	"errors"

	"github.com/Peripli/service-manager/pkg/web"

	httpsec "github.com/Peripli/service-manager/pkg/security/http"
	. "github.com/onsi/ginkgo/v2"
)

var _ = Describe("AndAuthorizer", func() {
	Describe("Authorize", func() {
		cases := []authorizerCase{
			{
				description:      "Allows if all authorizers allow with lowest access level",
				expectedDecision: httpsec.Allow,
				authorizers: []httpsec.Authorizer{
					&dummyAuthorizer{
						decision: httpsec.Allow,
						access:   web.GlobalAccess,
						err:      nil,
					},
					&dummyAuthorizer{
						decision: httpsec.Allow,
						access:   web.AllTenantAccess,
						err:      nil,
					},
				},
				expectedAccess: web.AllTenantAccess,
			},
			{
				description:      "Denies if one authorizer denies",
				expectedDecision: httpsec.Deny,
				swap:             true,
				authorizers: []httpsec.Authorizer{
					&dummyAuthorizer{
						decision: httpsec.Deny,
						err:      nil,
						access:   web.NoAccess,
					},
					&dummyAuthorizer{
						decision: httpsec.Allow,
						err:      nil,
						access:   web.GlobalAccess,
					},
				},
				expectedAccess: web.NoAccess,
			},
			{
				description:      "Denies if all authorizers deny",
				expectedDecision: httpsec.Deny,
				authorizers: []httpsec.Authorizer{
					&dummyAuthorizer{
						decision: httpsec.Deny,
						err:      nil,
						access:   web.NoAccess,
					},
					&dummyAuthorizer{
						decision: httpsec.Deny,
						err:      nil,
						access:   web.NoAccess,
					},
				},
				expectedAccess: web.NoAccess,
			},
			{
				description:      "Abstains if all authorizers abstain",
				expectedDecision: httpsec.Abstain,
				authorizers: []httpsec.Authorizer{
					&dummyAuthorizer{
						decision: httpsec.Abstain,
						err:      nil,
						access:   web.NoAccess,
					},
					&dummyAuthorizer{
						decision: httpsec.Abstain,
						err:      nil,
						access:   web.NoAccess,
					},
				},
				expectedAccess: web.NoAccess,
			},
			{
				description:      "Abstains if one authorizer abstains and other allows",
				expectedDecision: httpsec.Abstain,
				swap:             true,
				authorizers: []httpsec.Authorizer{
					&dummyAuthorizer{
						decision: httpsec.Abstain,
						err:      nil,
						access:   web.NoAccess,
					},
					&dummyAuthorizer{
						decision: httpsec.Allow,
						err:      nil,
						access:   web.GlobalAccess,
					},
				},
				expectedAccess: web.NoAccess,
			},
			{
				description:      "Denies if one authorizer abstains and other denies",
				expectedDecision: httpsec.Deny,
				swap:             true,
				authorizers: []httpsec.Authorizer{
					&dummyAuthorizer{
						decision: httpsec.Abstain,
						err:      nil,
						access:   web.NoAccess,
					},
					&dummyAuthorizer{
						decision: httpsec.Deny,
						err:      nil,
						access:   web.NoAccess,
					},
				},
				expectedAccess: web.NoAccess,
			},
			// error cases
			{
				description:      "Errors and abstains if one authorizer errors and other denies",
				expectedDecision: httpsec.Deny,
				expectError:      "abstained",
				authorizers: []httpsec.Authorizer{
					&dummyAuthorizer{
						decision: httpsec.Abstain,
						err:      errors.New("abstained"),
						access:   web.NoAccess,
					},
					&dummyAuthorizer{
						decision: httpsec.Deny,
						err:      nil,
						access:   web.NoAccess,
					},
				},
				expectedAccess: web.NoAccess,
			},
			{
				description:      "Denies if one authorizer denies, but the other throws an error and abstains",
				expectedDecision: httpsec.Deny,
				expectError:      "abstained",
				authorizers: []httpsec.Authorizer{
					&dummyAuthorizer{
						decision: httpsec.Deny,
						err:      nil,
						access:   web.NoAccess,
					},
					&dummyAuthorizer{
						decision: httpsec.Abstain,
						err:      errors.New("abstained"),
						access:   web.NoAccess,
					},
				},
				expectedAccess: web.NoAccess,
			},
			{
				description:      "Denies and errors if one authorizer allows and other errors and denies",
				expectedDecision: httpsec.Deny,
				expectError:      "denied",
				authorizers: []httpsec.Authorizer{
					&dummyAuthorizer{
						decision: httpsec.Allow,
						err:      nil,
						access:   web.TenantAccess,
					},
					&dummyAuthorizer{
						decision: httpsec.Deny,
						err:      errors.New("denied"),
						access:   web.NoAccess,
					},
				},
				expectedAccess: web.NoAccess,
			},
		}

		testAuthorizers(cases, NewAndAuthorizer)
	})
})
