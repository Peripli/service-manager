package multitenancy_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/Peripli/service-manager/pkg/multitenancy"
	"github.com/Peripli/service-manager/pkg/web"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestMultitenancy(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Multitenancy Test Suite")
}

var _ = Describe("ExtractTenantFromToken", func() {
	var (
		clientID           string
		clientIDTokenClaim string
		tenantTokenClaim   string
		tenant             string
	)

	When("some configurations are missing", func() {
		It("should return error for missing clientID", func() {
			extractorFunc := multitenancy.ExtractTenatFromTokenWrapperFunc("", "clientid-claim", "tenant-claim")
			_, err := extractorFunc(nil)
			Expect(err).Should(HaveOccurred())
		})

		It("should return error for missing clientIDTokenClaim", func() {
			extractorFunc := multitenancy.ExtractTenatFromTokenWrapperFunc("clientid", "", "tenant-claim")
			_, err := extractorFunc(nil)
			Expect(err).Should(HaveOccurred())
		})

		It("should return error for missing tenantTokenClaim", func() {
			extractorFunc := multitenancy.ExtractTenatFromTokenWrapperFunc("clientid", "clientid-claim", "")
			_, err := extractorFunc(nil)
			Expect(err).Should(HaveOccurred())
		})
	})
	When("all configurations are provided", func() {
		var (
			ctx         context.Context
			fakeRequest *web.Request
		)

		BeforeEach(func() {
			clientID = "tenantClient"
			clientIDTokenClaim = "cid"
			tenantTokenClaim = "zid"
			tenant = "tenantID"

			ctx = context.TODO()
			req, err := http.NewRequest(http.MethodGet, "http://example.com", nil)
			Expect(err).ToNot(HaveOccurred())
			fakeRequest = &web.Request{
				Request: req,
			}
			fakeRequest.Request = fakeRequest.Request.WithContext(web.ContextWithUser(ctx, &web.UserContext{
				Data: func(data interface{}) error {
					return json.Unmarshal([]byte(fmt.Sprintf(`{"%s":"%s","%s":"%s"}`, clientIDTokenClaim, clientID, tenantTokenClaim, tenant)), data)
				},
				AuthenticationType: web.Bearer,
				Name:               "test-user",
			}))
		})

		When("user is missing from context", func() {
			It("should return empty tenant", func() {
				fakeRequest.Request = fakeRequest.Request.WithContext(context.TODO())
				extractorFunc := multitenancy.ExtractTenatFromTokenWrapperFunc(clientID, clientIDTokenClaim, tenantTokenClaim)
				extractedTenant, err := extractorFunc(fakeRequest)
				Expect(err).ShouldNot(HaveOccurred())
				Expect(extractedTenant).To(Equal(""))
			})
		})

		When("context user is not Bearer token", func() {
			It("should return empty tenant", func() {
				fakeRequest.Request = fakeRequest.Request.WithContext(web.ContextWithUser(ctx, &web.UserContext{
					Data: func(data interface{}) error {
						return nil
					},
					AuthenticationType: web.Basic,
					Name:               "test-user",
				}))
				extractorFunc := multitenancy.ExtractTenatFromTokenWrapperFunc(clientID, clientIDTokenClaim, tenantTokenClaim)
				extractedTenant, err := extractorFunc(fakeRequest)
				Expect(err).ShouldNot(HaveOccurred())
				Expect(extractedTenant).To(Equal(""))
			})
		})

		When("getting claims from token fails", func() {
			It("returns an error", func() {
				fakeRequest.Request = fakeRequest.Request.WithContext(web.ContextWithUser(ctx, &web.UserContext{
					Data: func(claims interface{}) error {
						return fmt.Errorf("error")
					},
					AuthenticationType: web.Bearer,
					Name:               "test-user",
				}))

				extractorFunc := multitenancy.ExtractTenatFromTokenWrapperFunc(clientID, clientIDTokenClaim, tenantTokenClaim)
				extractedTenant, err := extractorFunc(fakeRequest)
				Expect(err).Should(HaveOccurred())
				Expect(extractedTenant).To(Equal(""))
			})
		})

		When("client ID token claim is not found in the token claims", func() {
			It("should return empty tenant", func() {
				extractorFunc := multitenancy.ExtractTenatFromTokenWrapperFunc(clientID, "different-value", tenantTokenClaim)
				extractedTenant, err := extractorFunc(fakeRequest)
				Expect(err).ShouldNot(HaveOccurred())
				Expect(extractedTenant).To(Equal(""))
			})
		})

		When("tenant token claim is not found in the token claims", func() {
			It("should return an error", func() {
				extractorFunc := multitenancy.ExtractTenatFromTokenWrapperFunc(clientID, clientIDTokenClaim, "different-value")
				extractedTenant, err := extractorFunc(fakeRequest)
				Expect(err).Should(HaveOccurred())
				Expect(extractedTenant).To(Equal(""))
			})
		})

		When("authentication is bearer", func() {
			It("should extract tenant from token", func() {
				extractorFunc := multitenancy.ExtractTenatFromTokenWrapperFunc(clientID, clientIDTokenClaim, tenantTokenClaim)
				extractedTenant, err := extractorFunc(fakeRequest)
				Expect(err).ShouldNot(HaveOccurred())
				Expect(extractedTenant).To(Equal(tenant))
			})
		})
	})
})
