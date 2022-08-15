package multitenancy_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/multitenancy"
	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/web"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestMultitenancy(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Multitenancy Test Suite")
}

var _ = Describe("ExtractTenantFromToken", func() {
	var (
		tenantTokenClaim string
		tenant           string
	)

	When("tenant token claim is missing", func() {
		It("returns error", func() {
			extractorFunc := multitenancy.ExtractTenantFromTokenWrapperFunc("")
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
					return json.Unmarshal([]byte(fmt.Sprintf(`{"%s":"%s"}`, tenantTokenClaim, tenant)), data)
				},
				AuthenticationType: web.Bearer,
				Name:               "test-user",
				AccessLevel:        web.TenantAccess,
			}))
		})

		When("user is missing from context", func() {
			It("should return empty tenant", func() {
				fakeRequest.Request = fakeRequest.Request.WithContext(context.TODO())
				extractorFunc := multitenancy.ExtractTenantFromTokenWrapperFunc(tenantTokenClaim)
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
					AccessLevel:        web.TenantAccess,
				}))
				extractorFunc := multitenancy.ExtractTenantFromTokenWrapperFunc(tenantTokenClaim)
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
					AccessLevel:        web.TenantAccess,
				}))

				extractorFunc := multitenancy.ExtractTenantFromTokenWrapperFunc(tenantTokenClaim)
				extractedTenant, err := extractorFunc(fakeRequest)
				Expect(err).Should(HaveOccurred())
				Expect(extractedTenant).To(Equal(""))
			})
		})

		When("tenant token claim is not found in the token claims", func() {
			It("should return an error", func() {
				extractorFunc := multitenancy.ExtractTenantFromTokenWrapperFunc("different-value")
				extractedTenant, err := extractorFunc(fakeRequest)
				Expect(err).Should(HaveOccurred())
				Expect(extractedTenant).To(Equal(""))
			})
		})

		When("authentication is bearer", func() {
			It("should extract tenant from token", func() {
				extractorFunc := multitenancy.ExtractTenantFromTokenWrapperFunc(tenantTokenClaim)
				extractedTenant, err := extractorFunc(fakeRequest)
				Expect(err).ShouldNot(HaveOccurred())
				Expect(extractedTenant).To(Equal(tenant))
			})
		})
	})
})
