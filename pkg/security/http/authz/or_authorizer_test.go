package authz

import (
	"errors"

	"github.com/Peripli/service-manager/pkg/web"

	httpsec "github.com/Peripli/service-manager/pkg/security/http"
	. "github.com/onsi/ginkgo"
)

var _ = Describe("AndAuthorizer", func() {
	Describe("Authorize", func() {
		cases := []authorizerCase{
			{
				description:      "Allows if all authorizers allow",
				expectedDecision: httpsec.Allow,
				expectedAccess:   web.GlobalAccess,
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
			},
			{
				description:      "Allows if at least one authorizer allows",
				expectedDecision: httpsec.Allow,
				expectedAccess:   web.TenantAccess,
				swap:             true,
				authorizers: []httpsec.Authorizer{
					&dummyAuthorizer{
						decision: httpsec.Deny,
						err:      nil,
					},
					&dummyAuthorizer{
						decision: httpsec.Allow,
						access:   web.TenantAccess,
						err:      nil,
					},
				},
			},
			{
				description:      "Denies if all authorizers deny",
				expectedDecision: httpsec.Deny,
				expectedAccess:   web.NoAccess,
				authorizers: []httpsec.Authorizer{
					&dummyAuthorizer{
						decision: httpsec.Deny,
						access:   web.NoAccess,
						err:      nil,
					},
					&dummyAuthorizer{
						decision: httpsec.Deny,
						access:   web.NoAccess,
						err:      nil,
					},
				},
			},
			{
				description:      "Abstains if all authorizers abstain",
				expectedDecision: httpsec.Abstain,
				expectedAccess:   web.NoAccess,
				authorizers: []httpsec.Authorizer{
					&dummyAuthorizer{
						decision: httpsec.Abstain,
						access:   web.NoAccess,
						err:      nil,
					},
					&dummyAuthorizer{
						decision: httpsec.Abstain,
						access:   web.NoAccess,
						err:      nil,
					},
				},
			},
			{
				description:      "Allows if one authorizer abstains and other allows",
				expectedDecision: httpsec.Allow,
				expectedAccess:   web.AllTenantAccess,
				swap:             true,
				authorizers: []httpsec.Authorizer{
					&dummyAuthorizer{
						decision: httpsec.Abstain,
						access:   web.NoAccess,
						err:      nil,
					},
					&dummyAuthorizer{
						decision: httpsec.Allow,
						access:   web.AllTenantAccess,
						err:      nil,
					},
				},
			},
			{
				description:      "Denies if one authorizer abstains and other denies",
				expectedDecision: httpsec.Deny,
				expectedAccess:   web.NoAccess,
				swap:             true,
				authorizers: []httpsec.Authorizer{
					&dummyAuthorizer{
						decision: httpsec.Abstain,
						access:   web.NoAccess,
						err:      nil,
					},
					&dummyAuthorizer{
						decision: httpsec.Deny,
						access:   web.NoAccess,
						err:      nil,
					},
				},
			},
			// error cases
			{
				description:      "Errors and abstains if one authorizer errors and other denies",
				expectedDecision: httpsec.Abstain,
				expectedAccess:   web.NoAccess,
				expectError:      "abstained",
				swap:             true,
				authorizers: []httpsec.Authorizer{
					&dummyAuthorizer{
						decision: httpsec.Abstain,
						access:   web.NoAccess,
						err:      errors.New("abstained"),
					},
					&dummyAuthorizer{
						decision: httpsec.Deny,
						access:   web.NoAccess,
						err:      nil,
					},
				},
			},
			{
				description:      "Allows if one authorizer allows and other errors and denies",
				expectedDecision: httpsec.Allow,
				expectedAccess:   web.GlobalAccess,
				swap:             true,
				authorizers: []httpsec.Authorizer{
					&dummyAuthorizer{
						decision: httpsec.Allow,
						access:   web.GlobalAccess,
						err:      nil,
					},
					&dummyAuthorizer{
						decision: httpsec.Deny,
						access:   web.NoAccess,
						err:      errors.New("denied"),
					},
				},
			},
		}

		testAuthorizers(cases, NewOrAuthorizer)
	})
})
