package authz

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/Peripli/service-manager/pkg/web"

	httpsec "github.com/Peripli/service-manager/pkg/security/http"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Oauth Scopes", func() {
	Describe("ScopeAuthorizer", func() {
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
		})
	})

	Describe("HasScope", func() {
		var (
			tokenDataNoScopes              = `{"scope": []}`
			tokenDataWithRequestedScope    = `{"scope": ["the-scope", "another-scope"]}`
			tokenDataWithoutRequestedScope = `{"scope": ["another-scope"]}`
		)

		getUserContextWithToken := func(token string, err error) (*web.UserContext, error) {
			ctx := web.ContextWithUser(context.Background(), &web.UserContext{
				AuthenticationType: web.Bearer,
				Data: func(data interface{}) error {
					if err != nil {
						return err
					}
					return json.Unmarshal([]byte(token), data)
				},
			})
			user, ok := web.UserFromContext(ctx)
			if !ok {
				return nil, fmt.Errorf("Failed to retrieve user from context")
			}

			return user, nil
		}

		When("no scopes are available", func() {
			user, _ := getUserContextWithToken(tokenDataNoScopes, nil)

			It("is not found", func() {
				found, err := HasScope(user, "the-scope")
				Expect(err).ToNot(HaveOccurred())
				Expect(found).To(BeFalse())
			})
		})

		When("requested scope is not available", func() {
			user, _ := getUserContextWithToken(tokenDataWithoutRequestedScope, nil)

			It("is not found", func() {
				found, err := HasScope(user, "the-scope")
				Expect(err).ToNot(HaveOccurred())
				Expect(found).To(BeFalse())
			})
		})

		When("requested scope is available", func() {
			user, _ := getUserContextWithToken(tokenDataWithRequestedScope, nil)

			It("is found", func() {
				found, err := HasScope(user, "the-scope")
				Expect(err).ToNot(HaveOccurred())
				Expect(found).To(BeTrue())
			})
		})

		When("scope cannot be extracted from token", func() {
			user, _ := getUserContextWithToken(tokenDataWithRequestedScope, errors.New("failed to get user data"))

			It("is fails with an error", func() {
				_, err := HasScope(user, "the-scope")
				Expect(err).To(HaveOccurred())
			})
		})

	})
})
