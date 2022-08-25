package blocked_clients_test

import (
	"context"
	"fmt"
	"github.com/Peripli/service-manager/pkg/env"
	"github.com/Peripli/service-manager/pkg/sm"
	"github.com/Peripli/service-manager/pkg/web"
	"github.com/Peripli/service-manager/test/common"
	"github.com/gofrs/uuid"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"net/http"
	"testing"
)

const (
	SubaccountId = "subaccount-id"
)

func TestBlockedClients(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Blocked Client Tests")
}
func addBlockedClient(ctx *common.TestContext, clientId string, subaccountId string, blockedMethods []string) string {
	req := common.Object{
		"client_id":       clientId,
		"subaccount_id":   subaccountId,
		"blocked_methods": blockedMethods,
	}
	resp := ctx.SMWithOAuthForTenant.POST(web.BlockedClientsConfigURL).
		WithJSON(req).
		Expect().Status(http.StatusCreated)
	return resp.JSON().Object().Value("id").String().Raw()
}

var filterContext = &common.OverrideFilter{}
var userName string

var _ = Describe("test blocked clients", func() {
	var (
		ctx *common.TestContext
	)

	BeforeSuite(func() {
		ctx = common.NewTestContextBuilderWithSecurity().WithSMExtensions(func(ctx context.Context, smb *sm.ServiceManagerBuilder, e env.Environment) error {
			smb.RegisterFiltersBefore("BlockedClientsFilter", filterContext)
			return nil
		}).Build()
	})

	AfterSuite(func() {
		ctx.Cleanup()
	})

	var changeClientIdentifier = func() string {
		UUID, err := uuid.NewV4()
		Expect(err).ToNot(HaveOccurred())
		userName := UUID.String()
		filterContext.UserName = userName
		return userName
	}
	var blockedClientId string
	AfterEach(func() {
		if len(blockedClientId) > 0 {
			ctx.SMWithOAuthForTenant.DELETE(fmt.Sprintf(web.BlockedClientsConfigURL+"/%s", blockedClientId)).
				Expect().Status(http.StatusOK)
		}
	})

	Context("no client is blocked", func() {

		It("should allow to consume API", func() {
			ctx.SMWithOAuth.GET(web.BlockedClientsConfigURL)
			res := ctx.SMWithOAuth.GET(web.BlockedClientsConfigURL).
				Expect().
				Status(http.StatusOK).JSON().Object()
			res.Value("items").Array().Empty()
		})
	})
	Context("client is added the the black list", func() {
		When("blocked method is incorrect", func() {
			It("should return an error", func() {
				req := common.Object{
					"client_id":       common.UserNameInToken,
					"subaccount_id":   SubaccountId,
					"blocked_methods": []string{"GET", "Not Allow Method", "wrong"},
				}
				res := ctx.SMWithOAuthForTenant.POST(web.BlockedClientsConfigURL).
					WithJSON(req).
					Expect().Status(http.StatusBadRequest)
				res.Body().Contains("Invalid value for a blocked method")
			})
		})
		When("POST method is blocked", func() {
			It("should not allow call POST api for blocked client", func() {
				ctx.SMWithOAuth.GET(web.PlatformsURL).
					Expect().
					Status(http.StatusOK)
				blockedMethods := []string{"POST"}
				userName := changeClientIdentifier()
				blockedClientId = addBlockedClient(ctx, userName, SubaccountId, blockedMethods)
				platformJSON := common.MakePlatform("cf-platform", "cf-platform", "cloudfoundry", "test-platform-cf")

				Eventually(func() int {
					return ctx.SMWithOAuth.POST(web.PlatformsURL).WithJSON(platformJSON).Expect().Raw().StatusCode
				}).Should((Equal(http.StatusMethodNotAllowed)))
				//should allow get
				ctx.SMWithOAuth.GET(web.PlatformsURL).
					Expect().
					Status(http.StatusOK)
				//another client
				changeClientIdentifier()
				ctx.SMWithOAuth.GET(web.PlatformsURL).
					Expect().
					Status(http.StatusOK)

			})

		})
		When("all methods are blocked", func() {
			It("should blocked all API calls", func() {
				userName := changeClientIdentifier()
				blockedClientId = addBlockedClient(ctx, userName, SubaccountId, []string{"POST", "GET", "DELETE", "PATCH"})
				Eventually(func() int {
					return ctx.SMWithOAuth.GET(web.PlatformsURL).Expect().Raw().StatusCode
				}).Should(Equal(http.StatusMethodNotAllowed))
				res := ctx.SMWithOAuth.POST(web.PlatformsURL).WithJSON(common.Object{}).Expect().Status(http.StatusMethodNotAllowed)
				res.Body().Contains(fmt.Sprintf("You're blocked to execute this request. Client: %s", userName))
				res = ctx.SMWithOAuth.GET(web.PlatformsURL).Expect().Status(http.StatusMethodNotAllowed)
				res.Body().Contains(fmt.Sprintf("You're blocked to execute this request. Client: %s", userName))
				res = ctx.SMWithOAuth.PATCH(web.PlatformsURL + "/some-id").WithJSON(common.Object{}).Expect().Status(http.StatusMethodNotAllowed)
				res.Body().Contains(fmt.Sprintf("You're blocked to execute this request. Client: %s", userName))
				res = ctx.SMWithOAuth.DELETE(web.PlatformsURL + "/some-id").WithJSON(common.Object{}).Expect().Status(http.StatusMethodNotAllowed)
				res.Body().Contains(fmt.Sprintf("You're blocked to execute this request. Client: %s", userName))
				changeClientIdentifier()

			})

		})
		When("blocked clients resynced", func() {
			It("should resync blocked clients", func() {
				blockedClientId = addBlockedClient(ctx, "test_user", SubaccountId, []string{"POST"})
				//adding directly to cache, the object no stored in db
				ctx.SMCache.AddL("key-1", "val-1")
				ctx.SMCache.AddL("key-2", "val-2")
				Expect(ctx.SMCache.Length()).To(Equal(3))
				ctx.SMWithOAuthForTenant.GET(web.ResyncBlockedClients).
					Expect().Status(http.StatusOK)
				Expect(ctx.SMCache.Length()).To(Equal(1))
			})
		})

		When("blocked client is removed from the list", func() {
			It("should allow to consume API", func() {
				userName := changeClientIdentifier()
				blockedClientId = addBlockedClient(ctx, userName, SubaccountId, []string{"POST", "GET", "PATCH"})
				Eventually(func() int {
					return ctx.SMWithOAuth.GET(web.PlatformsURL).Expect().Raw().StatusCode
				}).Should(Equal(http.StatusMethodNotAllowed))
				res := ctx.SMWithOAuth.GET(web.PlatformsURL).Expect().Status(http.StatusMethodNotAllowed)
				res.Body().Contains(fmt.Sprintf("You're blocked to execute this request. Client: %s", userName))
				ctx.SMWithOAuthForTenant.DELETE(fmt.Sprintf(web.BlockedClientsConfigURL+"/%s", blockedClientId)).
					Expect().Status(http.StatusOK)
				blockedClientId = ""
				Eventually(func() int {
					return ctx.SMWithOAuth.GET(web.PlatformsURL).Expect().Raw().StatusCode
				}).Should(Equal(http.StatusOK))
			})

		})

	})

})
