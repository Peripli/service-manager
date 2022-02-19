package authz

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

	httpsec "github.com/Peripli/service-manager/pkg/security/http"
	"github.com/Peripli/service-manager/pkg/web"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestAuthorization(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Authorization Suite")
}

type dummyAuthorizer struct {
	decision httpsec.Decision
	access   web.AccessLevel
	err      error
}

func (d *dummyAuthorizer) Authorize(req *web.Request) (httpsec.Decision, web.AccessLevel, error) {
	return d.decision, d.access, d.err
}

type testCase struct {
	params           interface{} // custom filter parameters
	noUser           bool
	claims           string // JSON
	claimsError      error
	expectError      string // HTTPError.Description
	expectedAccess   web.AccessLevel
	expectedDecision httpsec.Decision
}

type authorizerCase struct {
	description      string
	swap             bool
	expectError      string
	expectedDecision httpsec.Decision
	expectedAccess   web.AccessLevel
	authorizers      []httpsec.Authorizer
}

func runTestCase(tc testCase, authorizer httpsec.Authorizer) {
	baseReq := httptest.NewRequest("GET", "/", nil)

	token := tc.claims
	var user *web.UserContext
	if !tc.noUser {
		user = &web.UserContext{
			Data: func(data interface{}) error {
				if tc.claimsError != nil {
					return tc.claimsError
				}
				return json.Unmarshal([]byte(token), data)
			},
			AuthenticationType: web.Bearer,
			Name:               "test",
			AccessLevel:        web.NoAccess,
		}
	}
	webReq := &web.Request{Request: baseReq.WithContext(web.ContextWithUser(baseReq.Context(), user))}
	assertAuthorizer(authorizer, webReq, tc.expectError, tc.expectedDecision, tc.expectedAccess)
}

func testAuthorizers(cases []authorizerCase, createAuthorizer func(authorizers ...httpsec.Authorizer) httpsec.Authorizer) {
	webReq := &web.Request{
		Request: httptest.NewRequest("GET", "/", nil),
	}

	for _, tc := range cases {
		tc := tc
		It(tc.description, func() {
			authorizer := createAuthorizer(tc.authorizers...)
			assertAuthorizer(authorizer, webReq, tc.expectError, tc.expectedDecision, tc.expectedAccess)

			if tc.swap {
				swappedAuthorizers := make([]httpsec.Authorizer, len(tc.authorizers))
				swappedAuthorizers[0] = tc.authorizers[1]
				swappedAuthorizers[1] = tc.authorizers[0]
				authorizer = createAuthorizer(swappedAuthorizers...)
				assertAuthorizer(authorizer, webReq, tc.expectError, tc.expectedDecision, tc.expectedAccess)
			}
		})
	}
}

func assertAuthorizer(authorizator httpsec.Authorizer, webReq *web.Request, expectedError string, expectedDecision httpsec.Decision, expectedAccess web.AccessLevel) {
	decision, access, err := authorizator.Authorize(webReq)

	if expectedError != "" {
		Expect(err).NotTo(BeNil())
		Expect(err.Error()).To(ContainSubstring(expectedError))
	} else {
		Expect(err).To(SatisfyAny(BeNil(), BeEmpty()))
	}
	Expect(decision).To(Equal(expectedDecision))
	Expect(access).To(Equal(expectedAccess))
}
